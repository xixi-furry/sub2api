package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

func moderationV2Hash(parts ...string) string {
	raw, _ := json.Marshal(parts)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func moderationV2EventID(in ContentModerationCheckInput) string {
	id := in.AuditEventID
	if id == "" {
		id = in.RequestID
	}
	if id == "" {
		id = uuid.NewString()
	}
	return moderationV2Hash(fmt.Sprint(in.UserID), fmt.Sprint(in.APIKeyID), id, string(in.Body))
}
func moderationV2CacheKey(in ContentModerationCheckInput, cfg *ModerationV2Config, shared *ContentModerationConfig) string {
	policy, _ := json.Marshal(cfg)
	scope, _ := json.Marshal(shared)
	// Hash the full body before any extraction/truncation, including conversation
	// branches, alongside authenticated identities and immutable policy snapshots.
	return moderationV2Hash(fmt.Sprint(in.UserID), fmt.Sprint(in.APIKeyID), fmt.Sprint(contentModerationLogGroupID(in.GroupID)), in.Protocol, string(in.Body), string(policy), string(scope))
}
func buildModerationV2Payload(ctx context.Context, p ModerationV2Provider, text string) ([]byte, int, error) {
	raw, e := buildCustomModerationPayload(ctx, p.legacyConfig(), text)
	if e != nil {
		return nil, 0, e
	}
	var body map[string]json.RawMessage
	if e = json.Unmarshal(raw, &body); e != nil {
		return nil, 0, e
	}
	allowed := map[string]bool{"model": true, "messages": true, "temperature": true, "top_p": true, "response_format": true, "max_tokens": true, "max_completion_tokens": true, "stream": true, "n": true, "reasoning_effort": true, "thinking": true, "enable_thinking": true}
	for k := range body {
		if !allowed[k] {
			return nil, 0, errors.New("payload contains an unsupported parameter")
		}
	}
	var model string
	if json.Unmarshal(body["model"], &model) != nil || model != p.Model {
		return nil, 0, errors.New("payload model must match the configured model")
	}
	var messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(body["messages"], &messages) != nil || len(messages) == 0 || len(messages) > 32 {
		return nil, 0, errors.New("invalid messages")
	}
	for _, m := range messages {
		if m.Role != "system" && m.Role != "user" && m.Role != "assistant" && m.Role != "developer" {
			return nil, 0, errors.New("unsupported message role")
		}
		var content string
		if json.Unmarshal(m.Content, &content) != nil {
			return nil, 0, errors.New("v2 currently supports text-only message content")
		}
	}
	delete(body, "max_tokens")
	delete(body, "max_completion_tokens")
	body[p.OutputParameter] = json.RawMessage(fmt.Sprint(p.MaxOutputTokens))
	body["stream"] = json.RawMessage("false")
	body["n"] = json.RawMessage("1")
	raw, e = json.Marshal(body)
	if e != nil {
		return nil, 0, e
	}
	estimated := moderationV2Estimate(raw)
	if estimated > p.MaxInputTokens {
		return nil, estimated, errors.New("input_budget_exceeded")
	}
	return raw, estimated, nil
}

// A verdict may be valid even when billing usage is absent. Missing or malformed
// usage never frees a reservation. Cached tokens are a subset of input tokens.
func parseModerationV2Usage(raw []byte) *ModerationV2Usage {
	var out struct {
		Usage *struct {
			Prompt     *int64 `json:"prompt_tokens"`
			Completion *int64 `json:"completion_tokens"`
			Total      *int64 `json:"total_tokens"`
			CacheHit   *int64 `json:"prompt_cache_hit_tokens"`
			Details    *struct {
				Cached *int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Usage == nil || out.Usage.Prompt == nil || out.Usage.Completion == nil {
		return nil
	}
	u := ModerationV2Usage{Input: *out.Usage.Prompt, Output: *out.Usage.Completion}
	if out.Usage.CacheHit != nil {
		u.CachedInput = *out.Usage.CacheHit
	}
	if out.Usage.Details != nil && out.Usage.Details.Cached != nil {
		if out.Usage.CacheHit != nil && u.CachedInput != *out.Usage.Details.Cached {
			return nil
		}
		u.CachedInput = *out.Usage.Details.Cached
	}
	if u.Input < 0 || u.Output < 0 || u.Input > 1000000000 || u.Output > 1000000000 || u.CachedInput < 0 || u.CachedInput > u.Input {
		return nil
	}
	if out.Usage.Total != nil && *out.Usage.Total != u.Input+u.Output {
		return nil
	}
	return &u
}

type moderationV2CallResult struct {
	verdict    *ModerationV2Verdict
	usage      *ModerationV2Usage
	httpStatus int
	retryable  bool
	reason     string
}

func (s *ContentModerationService) callModerationV2(ctx context.Context, p ModerationV2Provider, payload []byte) moderationV2CallResult {
	failed := moderationV2CallResult{reason: "provider_unavailable", retryable: true}
	cfg := p.legacyConfig()
	endpoint, e := moderationEndpoint(cfg)
	if e != nil {
		failed.retryable = false
		failed.reason = "invalid_endpoint"
		return failed
	}
	client, e := s.moderationHTTPClient(ctx, cfg)
	if e != nil {
		failed.reason = "proxy_unavailable"
		return failed
	}
	// Never forward bearer credentials or repeat a paid request across redirects.
	owned := *client
	owned.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()
	request, e := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if e != nil {
		return failed
	}
	key := p.APIKeys[int(s.apiKeyCursor.Add(1)-1)%len(p.APIKeys)]
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	//nolint:gosec // URL is administrator-configured; private operator endpoints are supported. Redirects are disabled.
	response, e := owned.Do(request)
	if e != nil {
		return failed
	}
	defer func() { _ = response.Body.Close() }()
	failed.httpStatus = response.StatusCode
	if response.StatusCode != http.StatusOK {
		failed.reason = "provider_http_error"
		failed.retryable = response.StatusCode == 429 || response.StatusCode >= 500
		return failed
	}
	raw, e := io.ReadAll(io.LimitReader(response.Body, maxModerationResponseBytes+1))
	if e != nil || len(raw) > maxModerationResponseBytes {
		failed.reason = "invalid_response"
		return failed
	}
	failed.usage = parseModerationV2Usage(raw)
	result, e := parseChatModerationResponse(bytes.NewReader(raw), cfg)
	if e != nil {
		failed.reason = "invalid_verdict"
		return failed
	}
	score := result.CategoryScores[customModerationCategory]
	failed.verdict = &ModerationV2Verdict{Flagged: result.Flagged, Score: score, ProviderID: p.ID, Model: p.Model, Threshold: p.Threshold}
	if result.EngineMeta != nil {
		failed.verdict.Reason = result.EngineMeta.Reason
		failed.verdict.DecisionSource = result.EngineMeta.DecisionSource
	}
	failed.reason = ""
	return failed
}

func moderationV2Content(in ContentModerationCheckInput) (ContentModerationInput, string) {
	if !gjson.ValidBytes(in.Body) {
		return ContentModerationInput{}, "invalid_input"
	}
	content := extractContentModerationInput(in.Protocol, in.Body, false)
	// Do not silently discard unsupported attachments next to otherwise valid text.
	var parts gjson.Result
	switch in.Protocol {
	case ContentModerationProtocolOpenAIChat, ContentModerationProtocolAnthropicMessages:
		messages := gjson.GetBytes(in.Body, "messages").Array()
		if len(messages) > 0 {
			parts = messages[len(messages)-1].Get("content")
		}
	case ContentModerationProtocolOpenAIResponses:
		input := gjson.GetBytes(in.Body, "input")
		if input.IsArray() {
			items := input.Array()
			if len(items) > 0 {
				parts = items[len(items)-1].Get("content")
			}
		}
	case ContentModerationProtocolGemini:
		messages := gjson.GetBytes(in.Body, "contents").Array()
		if len(messages) > 0 {
			parts = messages[len(messages)-1].Get("parts")
		}
	case ContentModerationProtocolOpenAIImages:
	default:
		return content, "unsupported_input_parts"
	}
	if parts.IsArray() {
		for _, part := range parts.Array() {
			if part.Type == gjson.String {
				continue
			}
			typ := part.Get("type").String()
			if typ == "image" || typ == "image_url" || typ == "input_image" || part.Get("inlineData").Exists() || part.Get("fileData").Exists() {
				return content, "images_not_supported"
			}
			if typ != "text" && typ != "input_text" && !(in.Protocol == ContentModerationProtocolGemini && part.Get("text").Type == gjson.String) {
				return content, "unsupported_input_parts"
			}
		}
	}
	if len(content.Images) > 0 {
		return content, "images_not_supported"
	}
	if content.IsEmpty() {
		return content, "no_current_user_text"
	}
	return content, ""
}
func (s *ContentModerationService) evaluateModerationV2(ctx context.Context, in ContentModerationCheckInput, cfg *ModerationV2Config, shared *ContentModerationConfig, source, eventID string) *ModerationV2Result {
	out := &ModerationV2Result{Status: "unresolved"}
	store, e := s.moderationV2Store()
	if e != nil {
		out.Reason = "accounting_unavailable"
		return out
	}
	content, reason := moderationV2Content(in)
	if reason != "" {
		out.Reason = reason
		return out
	}
	cacheKey := moderationV2CacheKey(in, cfg, shared)
	if source != "admin_test" && cfg.CacheTTLSeconds > 0 {
		verdict, e := store.GetModerationV2Cache(ctx, cacheKey, cfg.Revision)
		if e == nil && verdict != nil {
			out.Status = "reviewed"
			out.Verdict = verdict
			out.CacheHit = true
			return out
		}
	}
	ids := append([]string{cfg.PrimaryID}, cfg.FallbackIDs...)
	for _, id := range ids {
		if out.Attempts >= cfg.MaxAttempts {
			break
		}
		p := cfg.provider(id)
		if p == nil || !p.Enabled || len(p.APIKeys) == 0 {
			out.Reason = "provider_not_configured"
			continue
		}
		payload, estimated, e := buildModerationV2Payload(ctx, *p, content.Text)
		out.EstimatedInput = estimated
		if e != nil {
			out.Reason = "invalid_payload"
			if estimated > p.MaxInputTokens {
				out.Reason = "input_budget_exceeded"
			}
			return out
		}
		// Reserve worst configured input/cache price, never an assumed cache discount.
		prices := p.Prices
		regular, _ := decimalValue(prices.Input)
		cached, _ := decimalValue(prices.CachedInput)
		if cached.GreaterThan(regular) {
			prices.Input = prices.CachedInput
		}
		amount := moderationV2Cost(prices, ModerationV2Usage{Input: int64(estimated), Output: int64(p.MaxOutputTokens)})
		attemptID := moderationV2Hash(eventID, fmt.Sprint(out.Attempts))
		reservation := ModerationV2Reservation{ID: attemptID, Revision: cfg.Revision, ProviderID: p.ID, Currency: cfg.Currency, Amount: amount, Tokens: int64(estimated + p.MaxOutputTokens), MaxConcurrent: p.MaxConcurrent, TimeoutMS: p.TimeoutMS, Source: source, GlobalLimits: cfg.Limits, ProviderLimits: p.Limits}
		if e = store.ReserveModerationV2(ctx, reservation); e != nil {
			if errors.Is(e, ErrModerationV2ProviderUnavailable) {
				out.Reason = "provider_cooldown_or_limit"
				continue
			}
			out.Reason = "accounting_unavailable"
			if errors.Is(e, ErrModerationV2Budget) {
				out.Reason = "budget_or_concurrency_exhausted"
			}
			if errors.Is(e, ErrModerationV2Conflict) {
				out.Reason = "configuration_changed"
			}
			if errors.Is(e, ErrModerationV2Duplicate) {
				out.Reason = "request_already_running"
			}
			return out
		}
		out.Attempts++
		call := s.callModerationV2(ctx, *p, payload)
		out.Usage = call.usage
		settlement := ModerationV2Settlement{ID: attemptID, State: "unknown", Usage: call.usage, HTTPStatus: call.httpStatus}
		if call.usage != nil {
			settlement.LimitExceeded = call.usage.Input > int64(estimated) || call.usage.Output > int64(p.MaxOutputTokens)
		}
		if call.usage != nil && p.Prices.Input != "" && p.Prices.Output != "" {
			settlement.State = "settled"
			settlement.Amount = moderationV2Cost(p.Prices, *call.usage)
		}
		// Cancellation/disconnect does not prevent accounting for a completed call.
		settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		settleErr := store.SettleModerationV2(settleCtx, settlement)
		cancel()
		if settleErr != nil {
			out.Reason = "settlement_pending"
			return out
		}
		if call.usage != nil && (call.usage.Input > int64(estimated) || call.usage.Output > int64(p.MaxOutputTokens)) {
			out.Reason = "provider_limit_mismatch"
			return out
		}
		if call.verdict != nil {
			out.Status = "reviewed"
			out.Verdict = call.verdict
			out.Reason = ""
			if source != "admin_test" && cfg.CacheTTLSeconds > 0 {
				if e := store.PutModerationV2Cache(ctx, cacheKey, *call.verdict, cfg.CacheTTLSeconds); e != nil {
					slog.Warn("moderation_v2.cache_write_failed")
				}
			}
			return out
		}
		out.Reason = call.reason
		if !call.retryable || ctx.Err() != nil {
			return out
		}
	}
	if out.Reason == "" {
		out.Reason = "no_provider_available"
	}
	return out
}
func moderationV2Decision(result *ModerationV2Result, cfg *ModerationV2Config, shared *ContentModerationConfig) *ContentModerationDecision {
	decision := &ContentModerationDecision{Allowed: true, Action: ContentModerationActionAllow}
	if result.Status != "reviewed" || result.Verdict == nil {
		decision.Action = "unreviewed"
		decision.Message = "内容审核暂不可用 / Content moderation temporarily unavailable"
		if cfg.UnresolvedPolicy == "reject_temporary" && shared.Mode == ContentModerationModePreBlock {
			decision.Allowed = false
			decision.Blocked = true
			decision.StatusCode = 503
			decision.ErrorCode = "content_moderation_unavailable"
		}
		return decision
	}
	v := result.Verdict
	decision.Flagged = v.Flagged
	decision.HighestCategory = customModerationCategory
	decision.HighestScore = v.Score
	decision.CategoryScores = map[string]float64{customModerationCategory: v.Score}
	if v.Flagged && shared.Mode == ContentModerationModePreBlock {
		decision.Allowed = false
		decision.Blocked = true
		decision.Action = ContentModerationActionBlock
		decision.StatusCode = shared.BlockStatus
		decision.Message = shared.BlockMessage
	}
	return decision
}
func (s *ContentModerationService) recordModerationV2(ctx context.Context, in ContentModerationCheckInput, shared *ContentModerationConfig, result *ModerationV2Result, decision *ContentModerationDecision, eventID string) {
	store, e := s.moderationV2Store()
	if e != nil {
		return
	}
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	claimed, e := store.ClaimModerationV2Event(recordCtx, eventID, result.Status, result.CacheHit, result.Reason)
	if e != nil {
		slog.Warn("moderation_v2.event_record_failed")
		return
	}
	if !claimed {
		return
	}
	if !decision.Flagged && result.Status == "reviewed" && !shared.RecordNonHits {
		return
	}
	content := extractContentModerationInput(in.Protocol, in.Body, false)
	log := s.buildLog(in, shared, decision.Action, decision.Flagged, decision.HighestCategory, decision.HighestScore, decision.CategoryScores, content.Text, nil, nil, result.Reason)
	log.EngineMeta = &ContentModerationEngineMeta{Engine: ContentModerationEngineOpenAI, Status: result.Status, CacheHit: result.CacheHit}
	if v := result.Verdict; v != nil {
		log.ThresholdSnapshot = map[string]float64{customModerationCategory: v.Threshold}
		log.EngineMeta.ProviderID = v.ProviderID
		log.EngineMeta.Model = v.Model
		log.EngineMeta.Reason = v.Reason
		log.EngineMeta.DecisionSource = v.DecisionSource
	}
	// No legacy global hash entry: v2 verdicts are isolated by identity + policy.
	s.enqueueRecord(in, shared, log, "", false, decision.Flagged)
}
func (s *ContentModerationService) checkModerationV2(ctx context.Context, in ContentModerationCheckInput, cfg *ModerationV2Config, shared *ContentModerationConfig, eventID string) *ContentModerationDecision {
	result := s.evaluateModerationV2(ctx, in, cfg, shared, "gateway", eventID)
	decision := moderationV2Decision(result, cfg, shared)
	s.recordModerationV2(ctx, in, shared, result, decision, eventID)
	return decision
}
func (s *ContentModerationService) dispatchModerationV2(ctx context.Context, in ContentModerationCheckInput, cfg *ModerationV2Config, shared *ContentModerationConfig) *ContentModerationDecision {
	eventID := moderationV2EventID(in)
	if !shared.shouldSample(moderationV2Hash(string(in.Body))) {
		result := &ModerationV2Result{Status: "unresolved", Reason: "not_sampled"}
		decision := &ContentModerationDecision{Allowed: true, Action: "unreviewed"}
		s.recordModerationV2(ctx, in, shared, result, decision, eventID)
		return decision
	}
	if shared.Mode == ContentModerationModeObserve {
		task := contentModerationTask{input: in, config: shared, v2: cfg, v2EventID: eventID, enqueuedAt: time.Now()}
		// Detach the raw body from the gateway's request buffer before queueing.
		task.input.Body = bytes.Clone(in.Body)
		select {
		case s.asyncQueue <- task:
			s.asyncEnqueued.Add(1)
		default:
			s.asyncDropped.Add(1)
			result := &ModerationV2Result{Status: "unresolved", Reason: "queue_full"}
			decision := moderationV2Decision(result, cfg, shared)
			s.recordModerationV2(ctx, in, shared, result, decision, eventID)
		}
		return &ContentModerationDecision{Allowed: true, Action: "observe_queued"}
	}
	start := time.Now()
	s.preBlockActive.Add(1)
	defer s.preBlockActive.Add(-1)
	decision := s.checkModerationV2(ctx, in, cfg, shared, eventID)
	action := decision.Action
	if action == "unreviewed" {
		action = ContentModerationActionError
	}
	s.recordPreBlockSyncMetric(int(time.Since(start).Milliseconds()), action)
	return decision
}

type ModerationV2Preview struct {
	ProviderID     string `json:"provider_id"`
	EstimatedInput int    `json:"estimated_input"`
	MaxOutput      int    `json:"max_output"`
	ReservedAmount string `json:"reserved_amount"`
	Fits           bool   `json:"fits"`
	Reason         string `json:"reason"`
}

func (s *ContentModerationService) PreviewModerationV2(ctx context.Context, text string) (*ModerationV2Preview, error) {
	cfg, e := s.loadModerationV2Config(ctx)
	if e != nil {
		return nil, e
	}
	p := cfg.provider(cfg.PrimaryID)
	if p == nil {
		return nil, moderationV2Error("configure a primary provider first")
	}
	_, estimate, e := buildModerationV2Payload(ctx, *p, text)
	reason := ""
	if e != nil {
		reason = e.Error()
	}
	amount := ""
	if p.Prices.Input != "" && p.Prices.Output != "" {
		prices := p.Prices
		regular, _ := decimalValue(prices.Input)
		cached, _ := decimalValue(prices.CachedInput)
		if cached.GreaterThan(regular) {
			prices.Input = prices.CachedInput
		}
		amount = moderationV2Cost(prices, ModerationV2Usage{Input: int64(estimate), Output: int64(p.MaxOutputTokens)})
	}
	return &ModerationV2Preview{ProviderID: p.ID, EstimatedInput: estimate, MaxOutput: p.MaxOutputTokens, ReservedAmount: amount, Fits: e == nil, Reason: reason}, nil
}
func (s *ContentModerationService) TestModerationV2(ctx context.Context, text string) (*ModerationV2Result, error) {
	cfg, e := s.loadModerationV2Config(ctx)
	if e != nil {
		return nil, e
	}
	// A saved configuration establishes the authoritative budget. Draft prices or
	// request-supplied caps cannot bypass it. Tests never ban users or fill cache.
	if cfg.Revision == 0 {
		return nil, moderationV2Error("save provider configuration before a billable test")
	}
	old, e := s.loadConfig(ctx)
	if e != nil {
		return nil, e
	}
	body, _ := json.Marshal(map[string]any{"input": text})
	input := ContentModerationCheckInput{RequestID: uuid.NewString(), Protocol: ContentModerationProtocolOpenAIResponses, Body: body}
	eventID := moderationV2EventID(input)
	result := s.evaluateModerationV2(ctx, input, cfg, old, "admin_test", eventID)
	if store, e := s.moderationV2Store(); e == nil {
		recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_, _ = store.ClaimModerationV2Event(recordCtx, eventID, result.Status, false, result.Reason)
	}
	return result, nil
}
