package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/url"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/dop251/goja"
)

const (
	ContentModerationAPIFormatModerations = "moderations"
	ContentModerationAPIFormatChat        = "chat_completions"
	customModerationCategory              = "custom"
	maxModerationScriptBytes              = 64 * 1024
	maxModerationPromptBytes              = 64 * 1024
	maxModerationPayloadBytes             = 16 * 1024 * 1024
	maxModerationResponseBytes            = 1024 * 1024
	moderationScriptTimeout               = 100 * time.Millisecond
)

type ContentModerationCustomConfig struct {
	APIFormat           string  `json:"api_format"`
	AuditPrompt         string  `json:"audit_prompt"`
	PayloadScript       string  `json:"payload_script"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
}

type ContentModerationCustomInput struct {
	APIFormat           *string  `json:"api_format"`
	AuditPrompt         *string  `json:"audit_prompt"`
	PayloadScript       *string  `json:"payload_script"`
	ConfidenceThreshold *float64 `json:"confidence_threshold"`
}

func (cfg *ContentModerationCustomConfig) applyCustomInput(input ContentModerationCustomInput) {
	if input.APIFormat != nil {
		cfg.APIFormat = strings.TrimSpace(*input.APIFormat)
	}
	if input.AuditPrompt != nil {
		cfg.AuditPrompt = *input.AuditPrompt
	}
	if input.PayloadScript != nil {
		cfg.PayloadScript = *input.PayloadScript
	}
	if input.ConfidenceThreshold != nil {
		cfg.ConfidenceThreshold = *input.ConfidenceThreshold
	}
}

func moderationEndpoint(cfg *ContentModerationConfig) (string, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("moderation Base URL must be an HTTP(S) URL without credentials, query or fragment")
	}
	path := strings.TrimRight(u.Path, "/")
	explicitEndpoint := strings.HasSuffix(path, "/moderations") || strings.HasSuffix(path, "/chat/completions")
	path = strings.TrimSuffix(strings.TrimSuffix(path, "/chat/completions"), "/moderations")
	if path == "" {
		path = "/v1"
	} else if cfg.APIFormat != ContentModerationAPIFormatChat && !explicitEndpoint && !strings.HasSuffix(path, "/v1") {
		// Preserve the legacy /prefix/v1/moderations base URL convention.
		path += "/v1"
	}
	if cfg.APIFormat == ContentModerationAPIFormatChat {
		path += "/chat/completions"
	} else {
		path += "/moderations"
	}
	u.Path, u.RawPath = path, ""
	return u.String(), nil
}

func validateCustomModerationConfig(ctx context.Context, cfg *ContentModerationConfig) error {
	bad := func(message string) error { return infraerrors.BadRequest("INVALID_CUSTOM_MODERATION_CONFIG", message) }
	if cfg.APIFormat != ContentModerationAPIFormatModerations && cfg.APIFormat != ContentModerationAPIFormatChat {
		return bad("审核接口类型无效 / Invalid moderation API format")
	}
	if math.IsNaN(cfg.ConfidenceThreshold) || math.IsInf(cfg.ConfidenceThreshold, 0) || cfg.ConfidenceThreshold < 0 || cfg.ConfidenceThreshold > 1 {
		return bad("置信度阈值必须在 0 到 1 之间 / Confidence threshold must be between 0 and 1")
	}
	if len(cfg.AuditPrompt) > maxModerationPromptBytes || len(cfg.PayloadScript) > maxModerationScriptBytes {
		return bad("系统提示词和 payload 代码各不能超过 64 KiB / Prompt and payload script must each fit in 64 KiB")
	}
	if cfg.Engine == ContentModerationEngineTypeSafe {
		return nil
	}
	if cfg.APIFormat == ContentModerationAPIFormatChat && strings.TrimSpace(cfg.AuditPrompt) == "" {
		return bad("请填写系统提示词 / A system prompt is required for chat moderation")
	}
	if _, err := moderationEndpoint(cfg); err != nil {
		return bad(err.Error())
	}
	if strings.TrimSpace(cfg.PayloadScript) != "" {
		if _, err := buildCustomModerationPayload(ctx, cfg, "hello"); err != nil {
			return bad(err.Error())
		}
	}
	return nil
}

// Each invocation gets an isolated runtime. Only data is exposed: no API keys,
// filesystem, process, network APIs or shared state. Input is never interpolated
// into JavaScript source. Scripts are administrator-authored, not user input.
func buildCustomModerationPayload(ctx context.Context, cfg *ContentModerationConfig, input any) ([]byte, error) {
	text, _ := typeSafeText(input)
	if strings.TrimSpace(cfg.PayloadScript) == "" {
		if cfg.APIFormat != ContentModerationAPIFormatChat {
			return json.Marshal(moderationAPIRequest{Model: cfg.Model, Input: input})
		}
		wrapped := "<user_input>\n" + strings.ReplaceAll(strings.ReplaceAll(text, "<", "&lt;"), ">", "&gt;") + "\n</user_input>"
		var userContent any = wrapped
		if parts, ok := input.([]moderationAPIInputPart); ok {
			content := []moderationAPIInputPart{{Type: "text", Text: wrapped}}
			for _, part := range parts {
				if part.Type == "image_url" {
					content = append(content, part)
				}
			}
			userContent = content
		}
		return json.Marshal(map[string]any{
			"model":       cfg.Model,
			"messages":    []map[string]any{{"role": "system", "content": cfg.AuditPrompt}, {"role": "user", "content": userContent}},
			"temperature": 0, "stream": false,
		})
	}
	if len(cfg.PayloadScript) > maxModerationScriptBytes {
		return nil, errors.New("payload script exceeds 64 KiB")
	}
	runtime := goja.New()
	runtime.SetMaxCallStackSize(256)
	// Preserve the JSON field names of image parts in the exposed input data.
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var inputValue any
	if err := json.Unmarshal(inputJSON, &inputValue); err != nil {
		return nil, err
	}
	for name, value := range map[string]any{
		"text": text, "input": inputValue,
		"isModerationEndpoint": cfg.APIFormat != ContentModerationAPIFormatChat,
		"config":               map[string]any{"model": cfg.Model, "auditPrompt": cfg.AuditPrompt},
	} {
		if err := runtime.Set(name, value); err != nil {
			return nil, fmt.Errorf("initialize payload script: %w", err)
		}
	}
	scriptCtx, cancel := context.WithTimeout(ctx, moderationScriptTimeout)
	defer cancel()
	stop := context.AfterFunc(scriptCtx, func() { runtime.Interrupt("payload script time limit exceeded") })
	defer stop()
	// Support the supplied const requestBody = ... snippet without a return statement.
	value, err := runtime.RunString(cfg.PayloadScript + "\n;JSON.stringify(typeof requestBody === 'string' ? JSON.parse(requestBody) : requestBody);")
	if err != nil {
		// Thrown values may contain audited content; do not echo them into logs.
		return nil, errors.New("payload script failed: check syntax, requestBody and the 100 ms execution limit")
	}
	if err := scriptCtx.Err(); err != nil {
		return nil, fmt.Errorf("payload script: %w", err)
	}
	if goja.IsUndefined(value) || goja.IsNull(value) {
		return nil, errors.New("payload script must define requestBody as a JSON object or JSON string")
	}
	raw := []byte(value.String())
	if len(raw) > maxModerationPayloadBytes {
		return nil, errors.New("payload exceeds 16 MiB")
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil || body == nil {
		return nil, errors.New("requestBody must be a JSON object")
	}
	var model string
	if json.Unmarshal(body["model"], &model) != nil || strings.TrimSpace(model) == "" {
		return nil, errors.New("requestBody.model must be a non-empty string")
	}
	if stream, exists := body["stream"]; exists && string(stream) != "false" {
		return nil, errors.New("moderation requires requestBody.stream to be false or omitted")
	}
	if cfg.APIFormat == ContentModerationAPIFormatChat {
		var messages []json.RawMessage
		if json.Unmarshal(body["messages"], &messages) != nil || len(messages) == 0 {
			return nil, errors.New("requestBody.messages must be a non-empty array")
		}
	} else if value, exists := body["input"]; !exists || string(value) == "null" {
		return nil, errors.New("requestBody.input is required for the Moderations API")
	}
	return raw, nil
}

func parseChatModerationResponse(body io.Reader, cfg *ContentModerationConfig) (*moderationAPIResult, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxModerationResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxModerationResponseBytes {
		return nil, errors.New("chat moderation response exceeds 1 MiB")
	}
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
				Refusal string `json:"refusal"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("invalid chat moderation response JSON")
	}
	if len(out.Choices) != 1 {
		return nil, errors.New("chat moderation must return exactly one choice")
	}
	choice := out.Choices[0]
	if choice.Message.Refusal != "" || (choice.FinishReason != "" && choice.FinishReason != "stop") {
		return nil, errors.New("chat moderation refused or did not finish normally")
	}
	content := strings.TrimSpace(choice.Message.Content)
	if strings.HasPrefix(content, "```json\n") && strings.HasSuffix(content, "```") {
		content = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(content, "```json\n"), "```"))
	}
	var verdict struct {
		Confidence *float64 `json:"confidence"`
		Flagged    *bool    `json:"flagged"`
		Reason     string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content), &verdict); err != nil || (verdict.Confidence == nil && verdict.Flagged == nil) {
		return nil, errors.New("chat moderation must return a JSON object containing confidence (0..1) or flagged (boolean)")
	}
	score, flagged := 0.0, false
	decisionSource := "flagged"
	if verdict.Confidence != nil {
		decisionSource = "confidence"
		score = *verdict.Confidence
		if math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 1 {
			return nil, errors.New("chat moderation confidence must be between 0 and 1")
		}
		flagged = score >= cfg.ConfidenceThreshold
	} else {
		flagged = *verdict.Flagged
		if flagged {
			score = 1
		}
	}
	return &moderationAPIResult{
		Flagged: flagged, Custom: true,
		CategoryScores:  map[string]float64{customModerationCategory: score},
		CustomThreshold: cfg.ConfidenceThreshold,
		EngineMeta:      &ContentModerationEngineMeta{DecisionSource: decisionSource, Engine: ContentModerationEngineOpenAI, Model: out.Model, RulesVersion: "custom-chat-v1", Reason: trimRunes(redactContentModerationSecrets(verdict.Reason), 240)},
	}, nil
}

func evaluateModerationResult(result *moderationAPIResult, thresholds map[string]float64) (bool, string, float64) {
	if result.Custom {
		return result.Flagged, customModerationCategory, result.CategoryScores[customModerationCategory]
	}
	return evaluateModerationScores(result.CategoryScores, thresholds)
}
