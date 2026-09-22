package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

func moderationV2WorstCost(p ModerationV2Provider, estimate int) string {
	prices := p.Prices
	regular, _ := decimalValue(prices.Input)
	cached, _ := decimalValue(prices.CachedInput)
	if cached.GreaterThan(regular) {
		prices.Input = prices.CachedInput
	}
	return moderationV2Cost(prices, ModerationV2Usage{Input: int64(estimate), Output: int64(p.MaxOutputTokens)})
}
func moderationV2StageOrder(ctx context.Context, cfg *ModerationV2Config, evidence moderationEvidence, review bool) ([]string, string) {
	ids := append([]string{cfg.PrimaryID}, cfg.FallbackIDs...)
	if review {
		ids = cfg.Policy.ReviewIDs
	}
	if cfg.Routing == "lowest_cost" {
		ids = nil
		for _, p := range cfg.Providers {
			ids = append(ids, p.ID)
		}
	}
	candidates := []moderationPricedRoute{}
	reason := "no_qualified_channel"
	for _, id := range ids {
		p := cfg.provider(id)
		if p == nil || !p.Enabled || !p.AuditValidated || len(p.APIKeys) == 0 {
			continue
		}
		purpose := moderationProviderPurpose(*p)
		if (review && purpose != "review" && purpose != "both") || (!review && purpose != "primary" && purpose != "both") {
			continue
		}
		if cfg.Routing == "lowest_cost" && (p.Prices.Input == "" || p.Prices.Output == "" || !p.OutputLimitVerified) {
			continue
		}
		copy := *p
		copy.StrictDecision = true
		_, estimate, e := buildModerationV2Payload(ctx, copy, evidence.text)
		if e != nil {
			reason = "invalid_payload"
			if estimate > p.MaxInputTokens {
				reason = "input_budget_exceeded"
			}
			continue
		}
		cost, _ := decimalValue(moderationV2WorstCost(*p, estimate))
		if cfg.Policy.MaxRequestAmount != "" {
			cap, _ := decimalValue(cfg.Policy.MaxRequestAmount)
			if p.Prices.Input == "" || p.Prices.Output == "" || !p.OutputLimitVerified || cost.GreaterThan(cap) {
				reason = "request_budget_exhausted"
				continue
			}
		}
		candidates = append(candidates, moderationPricedRoute{id: id, cost: cost})
	}
	if cfg.Routing == "lowest_cost" {
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].cost.LessThan(candidates[j].cost) })
	}
	out := make([]string, 0, len(candidates))
	for _, p := range candidates {
		out = append(out, p.id)
	}
	if len(out) > 0 {
		reason = ""
	}
	return out, reason
}
func moderationV2EvidenceVerdict(v *ModerationV2Verdict, evidence moderationEvidence) bool {
	ids := map[string]bool{}
	for _, f := range evidence.fragments {
		ids[f.ID] = true
	}
	for _, id := range v.EvidenceIDs {
		if !ids[id] {
			return false
		}
	}
	return true
}
func moderationV2Uncertain(v *ModerationV2Verdict, margin float64) bool {
	return v.Decision == "review" || v.NeedsContext || math.Abs(v.Score-v.Threshold) <= margin || (v.Decision == "allow" && v.Score >= v.Threshold) || (v.Decision == "block" && v.Score < v.Threshold)
}
func (s *ContentModerationService) evaluateModerationV2Enhanced(ctx context.Context, in ContentModerationCheckInput, cfg *ModerationV2Config, shared *ContentModerationConfig, source, eventID string) (out *ModerationV2Result) {
	start := time.Now()
	out = &ModerationV2Result{Status: "unresolved"}
	defer func() { out.TotalMS = time.Since(start).Milliseconds() }()
	ctx, cancel := context.WithTimeout(ctx, cfg.requestTimeout())
	defer cancel()
	store, e := s.moderationV2Store()
	if e != nil {
		out.Reason = "accounting_unavailable"
		return
	}
	cacheKey := moderationV2CacheKey(in, cfg, shared)
	// Extract before cache lookup so malformed or unsupported evidence is never
	// converted to a pass by an old cached verdict.
	review := cfg.Policy.Mode == "quality"
	evidence := buildModerationV2Evidence(in, cfg.Policy, review)
	if evidence.reason != "" && !review {
		review = true
		evidence = buildModerationV2Evidence(in, cfg.Policy, true)
	}
	out.Coverage = &evidence.coverage
	if evidence.reason != "" {
		out.Reason = evidence.reason
		return
	}
	if source != "admin_test" && cfg.CacheTTLSeconds > 0 {
		if verdict, err := store.GetModerationV2Cache(ctx, cacheKey, cfg.Revision); err == nil && verdict != nil {
			out.Status = "reviewed"
			out.Verdict = verdict
			out.CacheHit = true
			out.Coverage = nil
			return
		}
	}
	spent := decimal.Zero
	allUsageKnown := true
	var aggregate ModerationV2Usage
	used := map[string]bool{}
	var primary *ModerationV2Verdict
	for out.Attempts < cfg.MaxAttempts {
		if ctx.Err() != nil {
			out.Reason = "request_deadline_exceeded"
			return
		}
		if review {
			evidence = buildModerationV2Evidence(in, cfg.Policy, true)
			out.Coverage = &evidence.coverage
			if evidence.reason != "" {
				out.Reason = evidence.reason
				return
			}
		}
		ids, reason := moderationV2StageOrder(ctx, cfg, evidence, review)
		if reason == "" {
			reason = out.Reason
		}
		called := false
		for _, id := range ids {
			routeKey := fmt.Sprintf("%t/%s", review, id)
			if used[routeKey] {
				continue
			}
			used[routeKey] = true
			original := cfg.provider(id)
			p := *original
			p.StrictDecision = true
			payload, estimate, err := buildModerationV2Payload(ctx, p, evidence.text)
			out.EstimatedInput = estimate
			if err != nil {
				reason = "invalid_payload"
				continue
			}
			amount := moderationV2WorstCost(p, estimate)
			reserved, _ := decimalValue(amount)
			if cfg.Policy.MaxRequestAmount != "" {
				cap, _ := decimalValue(cfg.Policy.MaxRequestAmount)
				if p.Prices.Input == "" || p.Prices.Output == "" || !p.OutputLimitVerified || spent.Add(reserved).GreaterThan(cap) {
					reason = "request_budget_exhausted"
					continue
				}
			}
			deadline, _ := ctx.Deadline()
			remaining := time.Until(deadline)
			if remaining <= 0 {
				out.Reason = "request_deadline_exceeded"
				return
			}
			p.TimeoutMS = min(p.TimeoutMS, int((remaining+time.Millisecond-1)/time.Millisecond))
			attemptID := moderationV2Hash(eventID, fmt.Sprint(out.Attempts))
			reservation := ModerationV2Reservation{ID: attemptID, Revision: cfg.Revision, ProviderID: p.ID, Currency: cfg.Currency, Amount: amount, Tokens: int64(estimate + p.MaxOutputTokens), MaxConcurrent: p.MaxConcurrent, TimeoutMS: p.TimeoutMS, Source: source, GlobalLimits: cfg.Limits, ProviderLimits: p.Limits}
			if err = store.ReserveModerationV2(ctx, reservation); err != nil {
				if errors.Is(err, ErrModerationV2ProviderUnavailable) {
					reason = "provider_cooldown_or_limit"
					continue
				}
				out.Reason = "accounting_unavailable"
				switch {
				case errors.Is(err, ErrModerationV2Budget):
					out.Reason = "budget_or_concurrency_exhausted"
				case errors.Is(err, ErrModerationV2Conflict):
					out.Reason = "configuration_changed"
				case errors.Is(err, ErrModerationV2Duplicate):
					out.Reason = "request_already_running"
				}
				return
			}
			out.Attempts++
			spent = spent.Add(reserved)
			called = true
			call := s.callModerationV2(ctx, p, payload)
			settlement := ModerationV2Settlement{ID: attemptID, State: "unknown", Usage: call.usage, HTTPStatus: call.httpStatus}
			if call.usage != nil {
				settlement.LimitExceeded = call.usage.Input > int64(estimate) || call.usage.Output > int64(p.MaxOutputTokens)
				aggregate.Input += call.usage.Input
				aggregate.CachedInput += call.usage.CachedInput
				aggregate.Output += call.usage.Output
				if p.Prices.Input != "" && p.Prices.Output != "" {
					settlement.State = "settled"
					settlement.Amount = moderationV2Cost(p.Prices, *call.usage)
				}
			} else {
				allUsageKnown = false
			}
			if allUsageKnown {
				copy := aggregate
				out.Usage = &copy
			} else {
				out.Usage = nil
			}
			stage := "primary"
			if review {
				stage = "review"
			}
			coverage := evidence.coverage
			trace := ModerationV2Trace{Stage: stage, ProviderID: p.ID, Reason: call.reason, ReservedAmount: amount, AccountingState: settlement.State, Usage: call.usage, HeaderMS: call.headerMS, FirstTextMS: call.firstTextMS, TotalMS: call.totalMS, Coverage: &coverage}
			if p.Prices.Input == "" || p.Prices.Output == "" {
				trace.ReservedAmount = ""
			}
			if call.verdict != nil {
				trace.Decision = call.verdict.Decision
			}
			out.Traces = append(out.Traces, trace)
			settleCtx, settleCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			settleErr := store.SettleModerationV2(settleCtx, settlement)
			settleCancel()
			if settleErr != nil {
				out.Traces[len(out.Traces)-1].AccountingState = "pending"
				out.Reason = "settlement_pending"
				return
			}
			if settlement.LimitExceeded {
				out.Reason = "provider_limit_mismatch"
				return
			}
			if ctx.Err() != nil {
				out.Reason = "request_deadline_exceeded"
				return
			}
			if call.verdict != nil && !moderationV2EvidenceVerdict(call.verdict, evidence) {
				call.verdict = nil
				call.reason = "invalid_evidence_reference"
			}
			if call.verdict == nil {
				reason = call.reason
				out.Traces[len(out.Traces)-1].Reason = reason
				break
			}
			v := call.verdict
			if !review && (v.Flagged || moderationV2Uncertain(v, cfg.Policy.UncertaintyMargin)) {
				primary = v
				reason = "review_required"
				out.Traces[len(out.Traces)-1].Reason = reason
				break
			}
			if moderationV2Uncertain(v, cfg.Policy.UncertaintyMargin) {
				out.Reason = "review_unresolved"
				return
			}
			if primary != nil && primary.Flagged && !v.Flagged {
				out.Reason = "review_disagreement"
				return
			}
			out.Status = "reviewed"
			out.Verdict = v
			out.Reason = ""
			if source != "admin_test" && cfg.CacheTTLSeconds > 0 {
				_ = store.PutModerationV2Cache(ctx, cacheKey, *v, cfg.CacheTTLSeconds)
			}
			return
		}
		out.Reason = reason
		if !review {
			review = true
			continue
		}
		if !called {
			break
		}
	}
	if out.Reason == "" {
		out.Reason = "no_qualified_channel"
	}
	return
}
