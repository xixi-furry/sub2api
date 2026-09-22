package service

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
)

type ModerationV2TestInput struct {
	Text     string          `json:"text"`
	Protocol string          `json:"protocol"`
	Body     json.RawMessage `json:"body"`
}

func (r ModerationV2TestInput) checkInput() (ContentModerationCheckInput, error) {
	in := ContentModerationCheckInput{RequestID: uuid.NewString()}
	if len(r.Body) > 0 && r.Text != "" {
		return in, moderationV2Error("provide text or body, not both")
	}
	if len(r.Body) > 0 {
		if len(r.Body) > 256*1024 || !json.Valid(r.Body) {
			return in, moderationV2Error("invalid request body (maximum 256 KiB)")
		}
		switch r.Protocol {
		case ContentModerationProtocolOpenAIChat, ContentModerationProtocolOpenAIResponses, ContentModerationProtocolAnthropicMessages, ContentModerationProtocolGemini:
		default:
			return in, moderationV2Error("unsupported test protocol")
		}
		in.Body = r.Body
		in.Protocol = r.Protocol
	} else {
		if len(r.Text) == 0 || len(r.Text) > 256*1024 {
			return in, moderationV2Error("provide 1–262144 bytes of text")
		}
		in.Body, _ = json.Marshal(map[string]any{"input": r.Text})
		in.Protocol = ContentModerationProtocolOpenAIResponses
	}
	return in, nil
}
func (s *ContentModerationService) PreviewModerationV2Input(ctx context.Context, req ModerationV2TestInput) (*ModerationV2Preview, error) {
	in, e := req.checkInput()
	if e != nil {
		return nil, e
	}
	cfg, e := s.loadModerationV2Config(ctx)
	if e != nil {
		return nil, e
	}
	if !cfg.enhanced() {
		content, reason := moderationV2Content(in)
		if reason != "" {
			return &ModerationV2Preview{Reason: reason}, nil
		}
		return s.PreviewModerationV2(ctx, content.Text)
	}
	review := cfg.Policy.Mode == "quality"
	evidence := buildModerationV2Evidence(in, cfg.Policy, review)
	ids, reason := []string{}, evidence.reason
	if reason == "" {
		ids, reason = moderationV2StageOrder(ctx, cfg, evidence, review)
	}
	if len(ids) == 0 && !review {
		review = true
		evidence = buildModerationV2Evidence(in, cfg.Policy, true)
		reason = evidence.reason
		if reason == "" {
			ids, reason = moderationV2StageOrder(ctx, cfg, evidence, true)
		}
	}
	stage := "primary"
	if review {
		stage = "review"
	}
	out := &ModerationV2Preview{Reason: reason, Stage: stage, Coverage: &evidence.coverage, Fragments: append([]ModerationV2Fragment{}, evidence.fragments...)}
	// Display excerpts only; request size and coverage above refer to the full data.
	for i := range out.Fragments {
		runes := []rune(out.Fragments[i].Text)
		if len(runes) > 400 {
			out.Fragments[i].Text = string(runes[:400]) + "…"
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	for _, id := range ids {
		p := *cfg.provider(id)
		p.StrictDecision = true
		_, estimate, err := buildModerationV2Payload(ctx, p, evidence.text)
		out.ProviderID = id
		out.EstimatedInput = estimate
		out.MaxOutput = p.MaxOutputTokens
		out.Fits = err == nil
		if p.Prices.Input != "" && p.Prices.Output != "" {
			out.ReservedAmount = moderationV2WorstCost(p, estimate)
		}
		if cfg.Policy.MaxRequestAmount != "" {
			cap, _ := decimalValue(cfg.Policy.MaxRequestAmount)
			amount, _ := decimalValue(out.ReservedAmount)
			if cap.IsZero() || out.ReservedAmount == "" || amount.GreaterThan(cap) {
				out.Fits = false
				out.Reason = "request_budget_exhausted"
				continue
			}
		}
		out.Reason = ""
		return out, nil
	}
	return out, nil
}
