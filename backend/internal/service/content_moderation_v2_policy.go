package service

import (
	"errors"
	"math"
	"time"
)

// Existing saved configurations stay in legacy mode until the administrator opts in.
type ModerationV2Policy struct {
	Mode                  string   `json:"mode"`
	ContextMode           string   `json:"context_mode"`
	HistoryMessages       int      `json:"history_messages"`
	HistoryBytes          int      `json:"history_bytes"`
	ReviewHistoryMessages int      `json:"review_history_messages"`
	ReviewHistoryBytes    int      `json:"review_history_bytes"`
	ReviewIDs             []string `json:"review_ids"`
	UncertaintyMargin     float64  `json:"uncertainty_margin"`
	RequestTimeoutMS      int      `json:"request_timeout_ms"`
	MaxRequestAmount      string   `json:"max_request_amount"`
}

func defaultModerationV2Policy() ModerationV2Policy {
	return ModerationV2Policy{Mode: "legacy", ContextMode: "bounded", HistoryMessages: 4, HistoryBytes: 8192, ReviewHistoryMessages: 12, ReviewHistoryBytes: 32768, UncertaintyMargin: .1, RequestTimeoutMS: 60000, ReviewIDs: []string{}}
}
func (c *ModerationV2Config) enhanced() bool {
	return c.Policy.Mode == "balanced" || c.Policy.Mode == "quality"
}
func (c *ModerationV2Config) requestTimeout() time.Duration {
	ms := c.Policy.RequestTimeoutMS
	if ms <= 0 {
		ms = 60000
	}
	return time.Duration(ms) * time.Millisecond
}
func moderationProviderPurpose(p ModerationV2Provider) string {
	if p.Purpose == "" {
		return "primary"
	}
	return p.Purpose
}
func validateModerationV2Policy(c *ModerationV2Config) error {
	p := &c.Policy
	if p.Mode == "" {
		*p = defaultModerationV2Policy()
	}
	if p.Mode != "legacy" && p.Mode != "balanced" && p.Mode != "quality" {
		return errors.New("invalid audit policy mode")
	}
	if p.ContextMode != "current" && p.ContextMode != "bounded" && p.ContextMode != "full" {
		return errors.New("invalid context mode")
	}
	if p.HistoryMessages < 0 || p.HistoryMessages > 256 || p.ReviewHistoryMessages < p.HistoryMessages || p.ReviewHistoryMessages > 256 || p.HistoryBytes < 0 || p.HistoryBytes > 262144 || p.ReviewHistoryBytes < p.HistoryBytes || p.ReviewHistoryBytes > 1048576 || p.RequestTimeoutMS < 1000 || p.RequestTimeoutMS > 240000 || math.IsNaN(p.UncertaintyMargin) || p.UncertaintyMargin < 0 || p.UncertaintyMargin > .5 || len(p.ReviewIDs) > 2 {
		return errors.New("invalid context, review or request limits")
	}
	if _, e := decimalValue(p.MaxRequestAmount); e != nil {
		return e
	}
	if !c.enhanced() {
		return nil
	}
	if c.Enabled && c.UnresolvedPolicy != "reject_temporary" {
		return errors.New("增强审核需选择审核未完成时暂时拒绝 / Enhanced auditing requires temporary rejection for unresolved requests")
	}
	seen := map[string]bool{}
	for _, id := range p.ReviewIDs {
		candidate := c.provider(id)
		if seen[id] || candidate == nil || (moderationProviderPurpose(*candidate) != "review" && moderationProviderPurpose(*candidate) != "both") {
			return errors.New("invalid review channel")
		}
		seen[id] = true
	}
	if !c.Enabled {
		return nil
	}
	review := false
	for _, candidate := range c.Providers {
		purpose := moderationProviderPurpose(candidate)
		if candidate.Enabled && candidate.AuditValidated && (purpose == "review" || purpose == "both") && (c.Routing == "lowest_cost" || seen[candidate.ID]) {
			if c.Routing != "lowest_cost" || (candidate.Prices.Input != "" && candidate.Prices.Output != "" && candidate.OutputLimitVerified) {
				review = true
			}
		}
	}
	if !review {
		return errors.New("请配置并验证强审渠道 / Configure a validated review channel")
	}
	return nil
}
func validateModerationV2Protocol(p ModerationV2Provider) error {
	if p.APIFormat != "" && p.APIFormat != "chat_completions" && p.APIFormat != "responses" {
		return errors.New("invalid audit API format")
	}
	if p.Purpose != "" && p.Purpose != "primary" && p.Purpose != "review" && p.Purpose != "both" {
		return errors.New("invalid channel purpose")
	}
	if p.ReasoningParameter != "" && p.ReasoningParameter != "none" && p.ReasoningParameter != "effort" && p.ReasoningParameter != "thinking" && p.ReasoningParameter != "enable_thinking" {
		return errors.New("invalid reasoning parameter")
	}
	if p.APIFormat == "responses" && p.ReasoningParameter != "" && p.ReasoningParameter != "none" && p.ReasoningParameter != "effort" {
		return errors.New("Responses only supports the reasoning effort option")
	}
	if p.ReasoningParameter == "effort" {
		switch p.ReasoningEffort {
		case "none", "minimal", "low", "medium", "high", "xhigh":
		default:
			return errors.New("invalid reasoning effort")
		}
	}
	if p.HeaderTimeoutMS < 0 || p.IdleTimeoutMS < 0 || (p.TimeoutMS > 0 && (p.HeaderTimeoutMS > p.TimeoutMS || p.IdleTimeoutMS > p.TimeoutMS)) {
		return errors.New("header and idle timeouts must fit within the channel timeout")
	}
	return nil
}

type ModerationV2Trace struct {
	Stage           string                `json:"stage"`
	ProviderID      string                `json:"provider_id"`
	Reason          string                `json:"reason"`
	Decision        string                `json:"decision,omitempty"`
	ReservedAmount  string                `json:"reserved_amount"`
	AccountingState string                `json:"accounting_state"`
	Usage           *ModerationV2Usage    `json:"usage,omitempty"`
	HeaderMS        int64                 `json:"header_ms"`
	FirstTextMS     *int64                `json:"first_text_ms,omitempty"`
	TotalMS         int64                 `json:"total_ms"`
	Coverage        *ModerationV2Coverage `json:"coverage,omitempty"`
}
