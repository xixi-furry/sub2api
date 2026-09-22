package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func adaptModerationV2Payload(body map[string]json.RawMessage, p ModerationV2Provider) error {
	set := func(k string, v any) { body[k], _ = json.Marshal(v) }
	if p.StrictDecision {
		// Append independently of the optional payload script so it cannot accidentally
		// discard the output contract while constructing its own messages.
		var messages []map[string]any
		if json.Unmarshal(body["messages"], &messages) != nil {
			return errors.New("invalid messages")
		}
		messages = append(messages, map[string]any{"role": "system", "content": moderationV2DecisionContract})
		set("messages", messages)
	}
	set("stream", p.Stream)
	if p.APIFormat == "responses" {
		body["input"] = body["messages"]
		delete(body, "messages")
		delete(body, "n")
		delete(body, "max_tokens")
		delete(body, "max_completion_tokens")
		set("max_output_tokens", p.MaxOutputTokens)
		set("store", false)
		set("background", false)
		set("truncation", "disabled")
		// Reasoning APIs may reject sampling parameters, including the legacy default
		// temperature=0. Responses auditing uses its explicit channel reasoning setting.
		delete(body, "temperature")
		delete(body, "top_p")
		if format, ok := body["response_format"]; ok {
			var f map[string]any
			if json.Unmarshal(format, &f) != nil {
				return errors.New("invalid response format")
			}
			if f["type"] == "json_schema" {
				schema, ok := f["json_schema"].(map[string]any)
				if !ok {
					return errors.New("invalid JSON schema format")
				}
				schema["type"] = "json_schema"
				f = schema
			}
			set("text", map[string]any{"format": f})
			delete(body, "response_format")
		}
		delete(body, "reasoning_effort")
		delete(body, "thinking")
		delete(body, "enable_thinking")
		if p.ReasoningParameter == "effort" {
			set("reasoning", map[string]any{"effort": p.ReasoningEffort})
		}
	} else {
		if p.Stream {
			set("stream_options", map[string]any{"include_usage": true})
		}
		if p.ReasoningParameter != "" && p.ReasoningParameter != "none" {
			delete(body, "reasoning_effort")
			delete(body, "thinking")
			delete(body, "enable_thinking")
			switch p.ReasoningParameter {
			case "effort":
				set("reasoning_effort", p.ReasoningEffort)
			case "thinking":
				typ := "disabled"
				if p.ThinkingEnabled {
					typ = "enabled"
				}
				set("thinking", map[string]any{"type": typ})
			case "enable_thinking":
				set("enable_thinking", p.ThinkingEnabled)
			}
		}
	}
	return nil
}
func moderationV2Endpoint(p ModerationV2Provider) (string, error) {
	cfg := p.legacyConfig()
	cfg.BaseURL = strings.TrimSuffix(strings.TrimRight(cfg.BaseURL, "/"), "/responses")
	endpoint, e := moderationEndpoint(cfg)
	if e == nil && p.APIFormat == "responses" {
		endpoint = strings.TrimSuffix(endpoint, "/chat/completions") + "/responses"
	}
	return endpoint, e
}
func parseModerationV2ResponseUsage(raw []byte) *ModerationV2Usage {
	usage := gjson.GetBytes(raw, "usage")
	if !usage.Exists() || usage.Type == gjson.Null {
		return nil
	}
	// Convert only field names; the common parser performs all numeric checks.
	var data map[string]json.RawMessage
	if json.Unmarshal([]byte(usage.Raw), &data) != nil {
		return nil
	}
	data["prompt_tokens"] = data["input_tokens"]
	data["completion_tokens"] = data["output_tokens"]
	if v, ok := data["input_tokens_details"]; ok {
		data["prompt_tokens_details"] = v
	}
	normalized, e := json.Marshal(map[string]any{"usage": data})
	if e != nil {
		return nil
	}
	return parseModerationV2Usage(normalized)
}
func moderationV2ResponseText(raw []byte) (string, error) {
	root := gjson.ParseBytes(raw)
	if !gjson.ValidBytes(raw) || root.Get("status").String() != "completed" || (root.Get("error").Exists() && root.Get("error").Type != gjson.Null) || (root.Get("incomplete_details").Exists() && root.Get("incomplete_details").Type != gjson.Null) {
		return "", errors.New("incomplete response")
	}
	texts := []string{}
	for _, item := range root.Get("output").Array() {
		if item.Get("type").String() == "reasoning" {
			continue
		}
		if item.Get("type").String() != "message" || item.Get("role").String() != "assistant" {
			return "", errors.New("unsupported response output")
		}
		if status := item.Get("status").String(); status != "" && status != "completed" {
			return "", errors.New("incomplete message")
		}
		for _, part := range item.Get("content").Array() {
			if part.Get("type").String() != "output_text" || part.Get("text").Type != gjson.String {
				return "", errors.New("refused or unsupported output")
			}
			texts = append(texts, part.Get("text").String())
		}
	}
	if len(texts) != 1 {
		return "", errors.New("expected one final verdict")
	}
	return texts[0], nil
}
func parseModerationV2Verdict(text string, p ModerationV2Provider) (*ModerationV2Verdict, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "```json"), "```"))
	}
	if strings.HasPrefix(text, "```") {
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "```"), "```"))
	}
	var v struct {
		Decision     string   `json:"decision"`
		Confidence   *float64 `json:"confidence"`
		Reason       string   `json:"reason"`
		NeedsContext *bool    `json:"needs_context"`
		EvidenceIDs  []string `json:"evidence_ids"`
	}
	if json.Unmarshal([]byte(text), &v) != nil {
		return nil, errors.New("invalid verdict JSON")
	}
	if p.StrictDecision || v.Decision != "" {
		if (v.Decision != "allow" && v.Decision != "block" && v.Decision != "review") || v.NeedsContext == nil || v.Confidence == nil || math.IsNaN(*v.Confidence) || *v.Confidence < 0 || *v.Confidence > 1 || len(v.EvidenceIDs) > 128 {
			return nil, errors.New("invalid structured verdict")
		}
		if v.Decision == "block" && len(v.EvidenceIDs) == 0 {
			return nil, errors.New("missing block evidence")
		}
		reason := []rune(redactContentModerationSecrets(v.Reason))
		if len(reason) > 240 {
			reason = reason[:240]
		}
		return &ModerationV2Verdict{Flagged: v.Decision == "block", Decision: v.Decision, Score: *v.Confidence, Reason: string(reason), DecisionSource: "structured", NeedsContext: *v.NeedsContext, EvidenceIDs: v.EvidenceIDs, ProviderID: p.ID, Model: p.Model, Threshold: p.Threshold}, nil
	}
	wrapped, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": text}}}})
	result, e := parseChatModerationResponse(bytes.NewReader(wrapped), p.legacyConfig())
	if e != nil {
		return nil, e
	}
	out := &ModerationV2Verdict{Flagged: result.Flagged, Score: result.CategoryScores[customModerationCategory], ProviderID: p.ID, Model: p.Model, Threshold: p.Threshold}
	if result.EngineMeta != nil {
		out.Reason = result.EngineMeta.Reason
		out.DecisionSource = result.EngineMeta.DecisionSource
	}
	return out, nil
}
func moderationV2ChatText(raw []byte) (string, error) {
	root := gjson.ParseBytes(raw)
	choices := root.Get("choices").Array()
	if !gjson.ValidBytes(raw) || len(choices) != 1 {
		return "", errors.New("invalid choices")
	}
	c := choices[0]
	finish := c.Get("finish_reason").String()
	if finish != "" && finish != "stop" {
		return "", errors.New("incomplete verdict")
	}
	if c.Get("message.refusal").String() != "" || c.Get("message.tool_calls").Exists() || c.Get("message.content").Type != gjson.String {
		return "", errors.New("invalid final content")
	}
	return c.Get("message.content").String(), nil
}

// Each body read resets transport inactivity only. Reasoning/heartbeat frames do
// not reset the absolute request deadline or count as visible output.
type moderationIdleReader struct {
	r     io.Reader
	timer *time.Timer
	idle  time.Duration
}

func (r moderationIdleReader) Read(p []byte) (int, error) {
	n, e := r.r.Read(p)
	if n > 0 && r.timer != nil {
		r.timer.Reset(r.idle)
	}
	return n, e
}
func readModerationV2Stream(reader io.Reader, responses bool, start time.Time) (raw []byte, first *int64, err error) {
	scanner := bufio.NewScanner(io.LimitReader(reader, 4*1024*1024+1))
	scanner.Buffer(make([]byte, 4096), maxModerationResponseBytes)
	var data []string
	var text strings.Builder
	var usage json.RawMessage
	defer func() {
		if !responses {
			raw, _ = json.Marshal(map[string]any{"choices": []any{map[string]any{"finish_reason": "stop", "message": map[string]any{"content": text.String()}}}, "usage": usage})
		}
	}()
	terminal := false
	stopped := false
	total := 0
	process := func() error {
		if len(data) == 0 {
			return nil
		}
		line := strings.Join(data, "\n")
		data = nil
		if line == "[DONE]" {
			if responses || !stopped {
				return errors.New("premature stream end")
			}
			terminal = true
			return nil
		}
		if !gjson.Valid(line) {
			return errors.New("invalid stream JSON")
		}
		v := gjson.Parse(line)
		if responses {
			switch v.Get("type").String() {
			case "response.output_text.delta":
				if v.Get("delta").String() != "" && first == nil {
					ms := time.Since(start).Milliseconds()
					first = &ms
				}
			case "response.completed":
				raw = []byte(v.Get("response").Raw)
				terminal = true
			case "response.failed", "response.incomplete", "error":
				raw = []byte(v.Get("response").Raw)
				return errors.New("incomplete stream")
			}
		} else {
			if v.Get("error").Exists() {
				return errors.New("stream error")
			}
			if u := v.Get("usage"); u.Exists() && u.Type != gjson.Null {
				usage = json.RawMessage(u.Raw)
			}
			for _, c := range v.Get("choices").Array() {
				if c.Get("index").Int() != 0 {
					return errors.New("multiple stream choices")
				}
				if c.Get("delta.refusal").String() != "" || c.Get("delta.tool_calls").Exists() {
					return errors.New("refusal or tool output")
				}
				delta := c.Get("delta.content").String()
				if delta != "" {
					if stopped {
						return errors.New("text after finish")
					}
					if first == nil {
						ms := time.Since(start).Milliseconds()
						first = &ms
					}
					text.WriteString(delta)
				}
				if f := c.Get("finish_reason"); f.Exists() && f.Type != gjson.Null {
					if f.String() != "stop" {
						return errors.New("incomplete verdict")
					}
					stopped = true
				}
			}
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > 4*1024*1024 {
			return raw, first, errors.New("oversized stream")
		}
		if line == "" {
			if err = process(); err != nil {
				return raw, first, err
			}
			if terminal {
				break
			}
		} else if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if !terminal {
		if len(data) > 0 {
			err = process()
		}
		if err == nil {
			err = scanner.Err()
		}
		if err == nil && !terminal {
			err = errors.New("unterminated stream")
		}
	}
	return raw, first, err
}

func (s *ContentModerationService) callModerationV2(ctx context.Context, p ModerationV2Provider, payload []byte) (out moderationV2CallResult) {
	start := time.Now()
	out.reason = "provider_unavailable"
	out.retryable = true
	defer func() { out.totalMS = time.Since(start).Milliseconds() }()
	endpoint, e := moderationV2Endpoint(p)
	if e != nil {
		out.reason = "invalid_endpoint"
		out.retryable = false
		return
	}
	client, e := s.moderationHTTPClient(ctx, p.legacyConfig())
	if e != nil {
		out.reason = "proxy_unavailable"
		return
	}
	owned := *client
	owned.Timeout = 0
	owned.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	callCtx, cancel := context.WithTimeout(ctx, time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()
	request, e := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if e != nil {
		return
	}
	if len(p.APIKeys) == 0 {
		out.reason = "provider_not_configured"
		return
	}
	key := p.APIKeys[int(s.apiKeyCursor.Add(1)-1)%len(p.APIKeys)]
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Content-Type", "application/json")
	var headerTimer *time.Timer
	if p.HeaderTimeoutMS > 0 {
		headerTimer = time.AfterFunc(time.Duration(p.HeaderTimeoutMS)*time.Millisecond, cancel)
		defer headerTimer.Stop()
	}
	//nolint:gosec // Administrator configured endpoints; credentials never follow redirects.
	response, e := owned.Do(request)
	if headerTimer != nil {
		headerTimer.Stop()
	}
	out.headerMS = time.Since(start).Milliseconds()
	if e != nil {
		return
	}
	defer func() { _ = response.Body.Close() }()
	out.httpStatus = response.StatusCode
	if response.StatusCode != http.StatusOK {
		out.reason = "provider_http_error"
		out.retryable = response.StatusCode == 429 || response.StatusCode >= 500
		return
	}
	var idleTimer *time.Timer
	idle := time.Duration(p.IdleTimeoutMS) * time.Millisecond
	if idle > 0 {
		idleTimer = time.AfterFunc(idle, cancel)
		defer idleTimer.Stop()
	}
	reader := moderationIdleReader{r: response.Body, timer: idleTimer, idle: idle}
	var raw []byte
	if p.Stream {
		if !strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
			out.reason = "invalid_response"
			return
		}
		raw, out.firstTextMS, e = readModerationV2Stream(reader, p.APIFormat == "responses", start)
	} else {
		raw, e = io.ReadAll(io.LimitReader(reader, maxModerationResponseBytes+1))
		if len(raw) > maxModerationResponseBytes {
			e = fmt.Errorf("oversized response")
		}
	}
	if p.APIFormat == "responses" {
		out.usage = parseModerationV2ResponseUsage(raw)
	} else {
		out.usage = parseModerationV2Usage(raw)
	}
	if e != nil {
		out.reason = "invalid_response"
		return
	}
	var finalText string
	if p.APIFormat == "responses" {
		finalText, e = moderationV2ResponseText(raw)
	} else {
		finalText, e = moderationV2ChatText(raw)
		if p.StrictDecision && gjson.GetBytes(raw, "choices.0.finish_reason").String() != "stop" {
			e = errors.New("missing completion marker")
		}
	}
	if e != nil {
		out.reason = "invalid_verdict"
		return
	}
	out.verdict, e = parseModerationV2Verdict(finalText, p)
	if e != nil {
		out.reason = "invalid_verdict"
		return
	}
	if !p.StrictDecision && (out.verdict.Decision == "review" || out.verdict.NeedsContext) {
		out.verdict = nil
		out.reason = "review_unresolved"
		return
	}
	out.reason = ""
	return
}
