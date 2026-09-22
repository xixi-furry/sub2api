package service

// Fork extension: provider routing and accounting are isolated from upstream's
// legacy engine configuration. V2 is opt-in and shares the gateway's scope/mode.
import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const SettingKeyContentModerationV2 = "content_moderation_v2"

var (
	ErrModerationV2Conflict            = infraerrors.New(409, "MODERATION_CONFIG_CONFLICT", "配置已变化，请重新载入 / Reload the changed configuration")
	ErrModerationV2Budget              = errors.New("moderation budget or concurrency limit reached")
	ErrModerationV2ProviderUnavailable = errors.New("moderation provider limit or cooldown reached")
	ErrModerationV2Duplicate           = errors.New("moderation attempt already reserved")
	moderationProviderID               = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
)

type ModerationV2Limits struct {
	DailyCalls  int64  `json:"daily_calls"` // 0 means unlimited; UI must say so.
	DailyTokens int64  `json:"daily_tokens"`
	DailyAmount string `json:"daily_amount"` // empty means unlimited, decimal currency units.
}
type ModerationV2Prices struct {
	Input       string `json:"input"`
	CachedInput string `json:"cached_input"`
	Output      string `json:"output"`
	PerRequest  string `json:"per_request"`
}
type ModerationV2Provider struct {
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Enabled             bool               `json:"enabled"`
	BaseURL             string             `json:"base_url"`
	Model               string             `json:"model"`
	ProxyID             *int64             `json:"proxy_id"`
	APIKeys             []string           `json:"api_keys,omitempty"`
	KeyMasks            []string           `json:"key_masks,omitempty"`
	ClearKeys           bool               `json:"clear_keys,omitempty"`
	AuditPrompt         string             `json:"audit_prompt"`
	PayloadScript       string             `json:"payload_script"`
	Threshold           float64            `json:"threshold"`
	TimeoutMS           int                `json:"timeout_ms"`
	MaxInputTokens      int                `json:"max_input_tokens"`
	MaxOutputTokens     int                `json:"max_output_tokens"`
	OutputParameter     string             `json:"output_parameter"`
	OutputLimitVerified bool               `json:"output_limit_verified"`
	MaxConcurrent       int                `json:"max_concurrent"`
	Prices              ModerationV2Prices `json:"prices"`
	Limits              ModerationV2Limits `json:"limits"`
}
type ModerationV2Config struct {
	Revision         int64                  `json:"revision"`
	Enabled          bool                   `json:"enabled"`
	Currency         string                 `json:"currency"`
	UnresolvedPolicy string                 `json:"unresolved_policy"`
	PrimaryID        string                 `json:"primary_id"`
	FallbackIDs      []string               `json:"fallback_ids"`
	MaxAttempts      int                    `json:"max_attempts"`
	CacheTTLSeconds  int                    `json:"cache_ttl_seconds"`
	Limits           ModerationV2Limits     `json:"limits"`
	Providers        []ModerationV2Provider `json:"providers"`
}
type ModerationV2Usage struct {
	Input       int64 `json:"input"`
	CachedInput int64 `json:"cached_input"`
	Output      int64 `json:"output"`
}
type ModerationV2Reservation struct {
	ID             string
	Revision       int64
	ProviderID     string
	Currency       string
	Amount         string
	Tokens         int64
	MaxConcurrent  int
	TimeoutMS      int
	Source         string
	GlobalLimits   ModerationV2Limits
	ProviderLimits ModerationV2Limits
}
type ModerationV2Settlement struct {
	LimitExceeded bool
	ID            string
	State         string
	Amount        string
	Usage         *ModerationV2Usage
	HTTPStatus    int
}
type ModerationV2Verdict struct {
	Flagged        bool    `json:"flagged"`
	Score          float64 `json:"score"`
	Reason         string  `json:"reason"`
	DecisionSource string  `json:"decision_source"`
	ProviderID     string  `json:"provider_id"`
	Model          string  `json:"model"`
	Threshold      float64 `json:"threshold"`
}
type ModerationV2Result struct {
	Status         string               `json:"status"`
	Reason         string               `json:"reason"`
	CacheHit       bool                 `json:"cache_hit"`
	Attempts       int                  `json:"attempts"`
	EstimatedInput int                  `json:"estimated_input"`
	Verdict        *ModerationV2Verdict `json:"verdict,omitempty"`
	Usage          *ModerationV2Usage   `json:"usage,omitempty"`
}
type ModerationV2UsageRow struct {
	ProviderID      string `json:"provider_id"`
	Calls           int64  `json:"calls"`
	HeldAmount      string `json:"held_amount"`
	ConfirmedAmount string `json:"confirmed_amount"`
	UnknownCalls    int64  `json:"unknown_calls"`
	Input           int64  `json:"input"`
	Output          int64  `json:"output"`
	CachedInput     int64  `json:"cached_input"`
}
type ModerationV2UsageSummary struct {
	Day        string                 `json:"day"`
	Currency   string                 `json:"currency"`
	Providers  []ModerationV2UsageRow `json:"providers"`
	CacheHits  int64                  `json:"cache_hits"`
	Unresolved int64                  `json:"unresolved"`
}

// Implemented by the existing SQL repository without expanding the upstream
// interface or changing its constructor/test doubles.
type ModerationV2Repository interface {
	SaveModerationV2Config(context.Context, int64, string) error
	ReserveModerationV2(context.Context, ModerationV2Reservation) error
	SettleModerationV2(context.Context, ModerationV2Settlement) error
	GetModerationV2Cache(context.Context, string, int64) (*ModerationV2Verdict, error)
	PutModerationV2Cache(context.Context, string, ModerationV2Verdict, int) error
	ClaimModerationV2Event(context.Context, string, string, bool, string) (bool, error)
	ModerationV2Usage(context.Context, string) (*ModerationV2UsageSummary, error)
}

func defaultModerationV2Config() *ModerationV2Config {
	return &ModerationV2Config{Currency: "CNY", MaxAttempts: 2, CacheTTLSeconds: 900, Providers: []ModerationV2Provider{}, FallbackIDs: []string{}}
}
func parseModerationV2Config(raw string) (*ModerationV2Config, error) {
	cfg := defaultModerationV2Config()
	if strings.TrimSpace(raw) == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, errors.New("invalid moderation v2 configuration")
	}
	return cfg, nil
}
func (p ModerationV2Provider) legacyConfig() *ContentModerationConfig {
	return &ContentModerationConfig{Engine: ContentModerationEngineOpenAI, BaseURL: p.BaseURL, Model: p.Model, ProxyID: p.ProxyID, TimeoutMS: p.TimeoutMS, ContentModerationCustomConfig: ContentModerationCustomConfig{APIFormat: ContentModerationAPIFormatChat, AuditPrompt: p.AuditPrompt, PayloadScript: p.PayloadScript, ConfidenceThreshold: p.Threshold}}
}
func (c *ModerationV2Config) provider(id string) *ModerationV2Provider {
	for i := range c.Providers {
		if c.Providers[i].ID == id {
			return &c.Providers[i]
		}
	}
	return nil
}
func (s *ContentModerationService) moderationV2Store() (ModerationV2Repository, error) {
	if r, ok := s.repo.(ModerationV2Repository); ok {
		return r, nil
	}
	return nil, infraerrors.InternalServer("MODERATION_V2_UNAVAILABLE", "多服务审核数据库不可用 / Moderation accounting unavailable")
}
func (s *ContentModerationService) loadModerationV2Config(ctx context.Context) (*ModerationV2Config, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyContentModerationV2)
	if err != nil && !errors.Is(err, ErrSettingNotFound) {
		return nil, err
	}
	cfg, err := parseModerationV2Config(raw)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		// Offer a disabled copy. Secrets remain on the server; enabling requires an
		// explicit save with prices, caps and an unresolved policy.
		old, err := s.loadConfig(ctx)
		if err == nil {
			old = old.effectiveEngine(old.Engine)
			if old.Engine == ContentModerationEngineOpenAI && old.APIFormat == ContentModerationAPIFormatChat {
				cfg.PrimaryID = "existing"
				cfg.Providers = []ModerationV2Provider{{ID: "existing", Name: "现有审核服务 / Existing service", Enabled: true, BaseURL: old.BaseURL, Model: old.Model, ProxyID: old.ProxyID, APIKeys: old.apiKeys(), AuditPrompt: old.AuditPrompt, PayloadScript: old.PayloadScript, Threshold: old.ConfidenceThreshold, TimeoutMS: old.TimeoutMS, MaxInputTokens: 4096, MaxOutputTokens: 512, OutputParameter: "max_tokens", MaxConcurrent: 4}}
			}
		}
	}
	return cfg, nil
}
func moderationV2Public(cfg *ModerationV2Config) *ModerationV2Config {
	c := *cfg
	c.Providers = append([]ModerationV2Provider{}, cfg.Providers...)
	for i := range c.Providers {
		p := &c.Providers[i]
		p.KeyMasks = []string{}
		for _, key := range p.APIKeys {
			p.KeyMasks = append(p.KeyMasks, maskSecretTail(key))
		}
		p.APIKeys = nil
		p.ClearKeys = false
	}
	return &c
}
func (s *ContentModerationService) GetModerationV2Config(ctx context.Context) (*ModerationV2Config, error) {
	cfg, err := s.loadModerationV2Config(ctx)
	if err != nil {
		return nil, err
	}
	return moderationV2Public(cfg), nil
}
func (s *ContentModerationService) UpdateModerationV2Config(ctx context.Context, cfg ModerationV2Config) (*ModerationV2Config, error) {
	store, err := s.moderationV2Store()
	if err != nil {
		return nil, err
	}
	old, err := s.loadModerationV2Config(ctx)
	if err != nil {
		return nil, err
	}
	if cfg.Revision != old.Revision {
		return nil, ErrModerationV2Conflict
	}
	if old.Revision > 0 && old.Currency != cfg.Currency {
		return nil, infraerrors.BadRequest("INVALID_MODERATION_CURRENCY", "币种建立账本后不可变更 / Ledger currency cannot be changed")
	}
	for i := range cfg.Providers {
		p := &cfg.Providers[i]
		if p.ClearKeys {
			p.APIKeys = nil
		} else if len(p.APIKeys) == 0 {
			if previous := old.provider(p.ID); previous != nil {
				p.APIKeys = previous.APIKeys
			}
		}
		keys := []string{}
		seen := map[string]bool{}
		for _, key := range p.APIKeys {
			key = strings.TrimSpace(key)
			if key != "" && !seen[key] {
				keys = append(keys, key)
				seen[key] = true
			}
		}
		p.APIKeys = keys
		p.KeyMasks = nil
		p.ClearKeys = false
	}
	if err := validateModerationV2Config(ctx, &cfg); err != nil {
		return nil, err
	}
	cfg.Revision++
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	if err := store.SaveModerationV2Config(ctx, old.Revision, string(raw)); err != nil {
		return nil, err
	}
	// Other nodes refresh normally; reservation also checks revision atomically.
	s.runtimeRefreshMu.Lock()
	s.runtimeSnapshot.Store(nil)
	s.runtimeRefreshMu.Unlock()
	return moderationV2Public(&cfg), nil
}
func decimalValue(raw string) (decimal.Decimal, error) {
	if raw == "" {
		return decimal.Zero, nil
	}
	if len(raw) > 30 || !regexp.MustCompile(`^\d+(\.\d{1,10})?$`).MatchString(raw) {
		return decimal.Zero, errors.New("use a non-negative decimal with at most 10 decimal places")
	}
	d, err := decimal.NewFromString(raw)
	if err != nil || d.IsNegative() || d.GreaterThan(decimal.NewFromInt(1000000000)) {
		return decimal.Zero, errors.New("decimal out of range")
	}
	return d, nil
}
func validateModerationV2Config(ctx context.Context, c *ModerationV2Config) error {
	bad := func(s string) error { return infraerrors.BadRequest("INVALID_MODERATION_V2", s) }
	if c.Revision < 0 || len(c.Providers) > 20 || len(c.FallbackIDs) > 1 || c.MaxAttempts < 1 || c.MaxAttempts > 2 || c.CacheTTLSeconds < 0 || c.CacheTTLSeconds > 86400 {
		return bad("最多 20 个服务、2 次调用；缓存 0–86400 秒 / Invalid limits")
	}
	if c.Currency != "CNY" && c.Currency != "USD" {
		return bad("币种只能为 CNY 或 USD / Unsupported currency")
	}
	if c.UnresolvedPolicy != "" && c.UnresolvedPolicy != "reject_temporary" && c.UnresolvedPolicy != "allow_record" {
		return bad("审核未完成时的处置无效 / Invalid unresolved policy")
	}
	if c.Enabled && c.UnresolvedPolicy == "" {
		return bad("请选择审核未完成时的处置 / Choose an unresolved policy")
	}
	checkLimits := func(l ModerationV2Limits) error {
		if l.DailyCalls < 0 || l.DailyTokens < 0 || l.DailyCalls > 1000000000 || l.DailyTokens > 1000000000000 {
			return bad("每日上限无效 / Invalid daily limit")
		}
		_, e := decimalValue(l.DailyAmount)
		return e
	}
	if err := checkLimits(c.Limits); err != nil {
		return bad(err.Error())
	}
	seen := map[string]bool{}
	for i := range c.Providers {
		p := &c.Providers[i]
		if !moderationProviderID.MatchString(p.ID) || seen[p.ID] || len(p.Name) > 120 || p.Name == "" {
			return bad("服务 ID 必须唯一且稳定，名称必填 / Invalid provider ID or name")
		}
		seen[p.ID] = true
		if err := checkLimits(p.Limits); err != nil {
			return bad(err.Error())
		}
		if len(p.APIKeys) > 100 {
			return bad("每个服务最多 100 个 Key / Too many keys")
		}
		for _, k := range p.APIKeys {
			if len(k) > 8192 || strings.ContainsAny(k, "\r\n") {
				return bad("API Key 无效 / Invalid API key")
			}
		}
		for _, v := range []string{p.Prices.Input, p.Prices.CachedInput, p.Prices.Output, p.Prices.PerRequest} {
			if _, err := decimalValue(v); err != nil {
				return bad(err.Error())
			}
		}
		if !p.Enabled {
			continue
		}
		if p.TimeoutMS < 500 || p.TimeoutMS > 30000 || p.MaxInputTokens < 256 || p.MaxInputTokens > 65536 || p.MaxOutputTokens < 32 || p.MaxOutputTokens > 4096 || p.MaxConcurrent < 1 || p.MaxConcurrent > 100 {
			return bad("服务超时、输入、输出或并发上限无效 / Invalid provider limits")
		}
		if p.OutputParameter != "max_tokens" && p.OutputParameter != "max_completion_tokens" {
			return bad("请选择输出限制参数 / Invalid output limit parameter")
		}
		if p.ProxyID != nil && *p.ProxyID <= 0 {
			return bad("代理 ID 必须为正数 / Invalid proxy ID")
		}
		if len(p.Model) > 256 || strings.TrimSpace(p.Model) == "" || math.IsNaN(p.Threshold) || p.Threshold < 0 || p.Threshold > 1 {
			return bad("模型和阈值无效 / Invalid model or threshold")
		}
		if err := validateCustomModerationConfig(ctx, p.legacyConfig()); err != nil {
			return err
		}
		if c.Enabled && len(p.APIKeys) == 0 {
			return bad("启用的服务需要 API Key / Enabled providers require keys")
		}
		if (c.Limits.DailyAmount != "" || p.Limits.DailyAmount != "") && (p.Prices.Input == "" || p.Prices.Output == "" || !p.OutputLimitVerified) {
			return bad("使用金额预算前请填写价格，并确认运营商支持输出上限 / Monetary caps require prices and verified output limits")
		}
	}
	route := append([]string{c.PrimaryID}, c.FallbackIDs...)
	used := map[string]bool{}
	for _, id := range route {
		if id == "" && !c.Enabled {
			continue
		}
		p := c.provider(id)
		if p == nil || used[id] || (c.Enabled && !p.Enabled) {
			return bad("主服务和备用服务必须存在、启用且不重复 / Invalid routing")
		}
		used[id] = true
	}
	return nil
}
func moderationV2Cost(prices ModerationV2Prices, u ModerationV2Usage) string {
	parse := func(v string) decimal.Decimal { d, _ := decimalValue(v); return d }
	cached := prices.CachedInput
	if cached == "" {
		cached = prices.Input
	}
	total := parse(prices.Input).Mul(decimal.NewFromInt(u.Input - u.CachedInput)).Add(parse(cached).Mul(decimal.NewFromInt(u.CachedInput))).Add(parse(prices.Output).Mul(decimal.NewFromInt(u.Output))).Div(decimal.NewFromInt(1000000)).Add(parse(prices.PerRequest))
	return total.RoundCeil(10).String()
}
func (s *ContentModerationService) GetModerationV2Usage(ctx context.Context) (*ModerationV2UsageSummary, error) {
	c, e := s.loadModerationV2Config(ctx)
	if e != nil {
		return nil, e
	}
	r, e := s.moderationV2Store()
	if e != nil {
		return nil, e
	}
	return r.ModerationV2Usage(ctx, c.Currency)
}

// Input estimation deliberately overestimates ordinary byte-tokenized text.
// It is not a universal tokenizer guarantee; provider billing remains authoritative.
func moderationV2Estimate(payload []byte) int { return len(payload) + 256 }
func moderationV2Error(reason string) error {
	return infraerrors.BadRequest("INVALID_MODERATION_V2", reason)
}
