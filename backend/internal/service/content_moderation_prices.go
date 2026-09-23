package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

const moderationPriceCatalogLimit = 16 << 20

type moderationPriceCatalogCache struct {
	mu        sync.Mutex
	registry  map[string]modelsDevProvider
	fetchedAt time.Time
	retryAt   time.Time
}

type ModerationModelPrice struct {
	ProviderID     string             `json:"provider_id"`
	ProviderName   string             `json:"provider_name"`
	ModelID        string             `json:"model_id"`
	ModelName      string             `json:"model_name"`
	MatchesChannel bool               `json:"matches_channel"`
	Prices         ModerationV2Prices `json:"prices"`
	Importable     bool               `json:"importable"`
	ManualReason   string             `json:"manual_reason,omitempty"`
	Reasoning      *bool              `json:"reasoning,omitempty"`
	Context        int64              `json:"context"`
}

type ModerationModelPriceResult struct {
	Source            string                 `json:"source"`
	Currency          string                 `json:"currency"`
	FetchedAt         string                 `json:"fetched_at"`
	Stale             bool                   `json:"stale"`
	MatchedProviderID string                 `json:"matched_provider_id"`
	Total             int                    `json:"total"`
	Items             []ModerationModelPrice `json:"items"`
}

// This catalog is used only in admin setup. Live moderation uses saved prices
// and never depends on models.dev being available.
func (s *ContentModerationService) LookupModerationModelPrices(ctx context.Context, query, baseURL string) (*ModerationModelPriceResult, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 || len(query) > 256 || len(baseURL) > 2048 {
		return nil, infraerrors.BadRequest("INVALID_PRICE_QUERY", "请填写 2–256 字符的模型名称 / Enter a model name of 2–256 characters")
	}
	registry, fetchedAt, stale, err := s.loadModerationPriceCatalog(ctx)
	if err != nil {
		return nil, infraerrors.ServiceUnavailable("MODEL_PRICE_CATALOG_UNAVAILABLE", "暂时无法读取公开价格，已保存的审核配置不受影响 / Public price catalog is unavailable")
	}
	result := searchModerationModelPrices(registry, query, baseURL)
	result.Source = modelsDevRegistryURL
	result.Currency = "USD"
	result.FetchedAt = fetchedAt.UTC().Format(time.RFC3339)
	result.Stale = stale
	return result, nil
}

func (s *ContentModerationService) loadModerationPriceCatalog(ctx context.Context) (map[string]modelsDevProvider, time.Time, bool, error) {
	cache := &s.priceCatalog
	cache.mu.Lock()
	defer cache.mu.Unlock()
	now := time.Now()
	if len(cache.registry) > 0 && now.Sub(cache.fetchedAt) < modelsDevRegistryTTL {
		return cache.registry, cache.fetchedAt, false, nil
	}
	staleResult := func() (map[string]modelsDevProvider, time.Time, bool, error) {
		if len(cache.registry) > 0 && time.Since(cache.fetchedAt) <= 7*24*time.Hour {
			return cache.registry, cache.fetchedAt, true, nil
		}
		return nil, time.Time{}, false, errors.New("model price catalog unavailable")
	}
	if now.Before(cache.retryAt) {
		return staleResult()
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsDevRegistryURL, nil)
	if err != nil {
		return nil, time.Time{}, false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sub2api-model-catalog/1.0")
	client := http.DefaultClient
	if s.httpClient != nil {
		client = s.httpClient
	}
	fixedClient := *client
	fixedClient.Jar = nil
	fixedClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	// Only the fixed public URL is fetched. Channel URLs, keys, prompts and
	// search terms never leave this server for catalog lookup.
	registry, err := downloadModerationPriceCatalog(&fixedClient, req)
	if err != nil {
		cache.retryAt = time.Now().Add(time.Minute)
		return staleResult()
	}
	cache.registry, cache.fetchedAt, cache.retryAt = registry, time.Now(), time.Time{}
	return cache.registry, cache.fetchedAt, false, nil
}

func downloadModerationPriceCatalog(client *http.Client, req *http.Request) (map[string]modelsDevProvider, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("price catalog HTTP %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, moderationPriceCatalogLimit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > moderationPriceCatalogLimit {
		return nil, errors.New("price catalog too large")
	}
	var registry map[string]modelsDevProvider
	if err := json.Unmarshal(body, &registry); err != nil {
		return nil, err
	}
	if len(registry) == 0 {
		return nil, errors.New("empty price catalog")
	}
	for id, provider := range registry {
		provider.ID = id
		if provider.Name == "" {
			provider.Name = id
		}
		registry[id] = provider
	}
	return registry, nil
}

// Match an actual provider endpoint, not a substring of its host or the model
// name. Bare shared hosts (e.g. OpenCode Zen and Go) remain ambiguous.
func moderationPriceProviderID(registry map[string]modelsDevProvider, baseURL string) string {
	parse := func(raw string) *url.URL {
		u, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil {
			return nil
		}
		u.Path = strings.TrimRight(u.Path, "/")
		for _, suffix := range []string{"/chat/completions", "/responses", "/models"} {
			u.Path = strings.TrimSuffix(u.Path, suffix)
		}
		return u
	}
	target := parse(baseURL)
	if target == nil {
		return ""
	}
	best, length, tied := "", -1, false
	for id, provider := range registry {
		candidate := parse(provider.API)
		if candidate == nil || !strings.EqualFold(candidate.Host, target.Host) || candidate.Scheme != target.Scheme {
			continue
		}
		if target.Path != candidate.Path && !strings.HasPrefix(target.Path, candidate.Path+"/") {
			continue
		}
		if len(candidate.Path) > length {
			best, length, tied = id, len(candidate.Path), false
		} else if len(candidate.Path) == length {
			tied = true
		}
	}
	if tied {
		return ""
	}
	if best != "" {
		return best
	}
	if strings.EqualFold(target.Host, "api.openai.com") && target.Scheme == "https" && (target.Path == "" || target.Path == "/v1") {
		if _, exists := registry["openai"]; exists {
			return "openai"
		}
	}
	return ""
}

func searchModerationModelPrices(registry map[string]modelsDevProvider, query, baseURL string) *ModerationModelPriceResult {
	matched := moderationPriceProviderID(registry, baseURL)
	result := &ModerationModelPriceResult{MatchedProviderID: matched, Items: []ModerationModelPrice{}}
	q := strings.ToLower(query)
	for providerID, provider := range registry {
		for modelID, model := range provider.Models {
			if !strings.Contains(strings.ToLower(modelID), q) && !strings.Contains(strings.ToLower(model.Name), q) {
				continue
			}
			prices, reason := moderationCatalogPrices(model.Cost)
			name := model.Name
			if name == "" {
				name = modelID
			}
			providerName := provider.Name
			if providerName == "" {
				providerName = providerID
			}
			result.Items = append(result.Items, ModerationModelPrice{ProviderID: providerID, ProviderName: providerName, ModelID: modelID, ModelName: name, MatchesChannel: providerID == matched, Prices: prices, Importable: reason == "", ManualReason: reason, Reasoning: model.Reasoning, Context: model.Limit.Context})
		}
	}
	sort.Slice(result.Items, func(i, j int) bool {
		a, b := result.Items[i], result.Items[j]
		if a.MatchesChannel != b.MatchesChannel {
			return a.MatchesChannel
		}
		ae, be := strings.EqualFold(a.ModelID, query), strings.EqualFold(b.ModelID, query)
		if ae != be {
			return ae
		}
		if a.ProviderID != b.ProviderID {
			return a.ProviderID < b.ProviderID
		}
		return a.ModelID < b.ModelID
	})
	result.Total = len(result.Items)
	if len(result.Items) > 100 {
		result.Items = result.Items[:100]
	}
	return result
}

func moderationCatalogDecimal(raw json.RawMessage) (string, bool) {
	value := strings.TrimSpace(string(raw))
	if len(value) == 0 || len(value) > 64 {
		return "", false
	}
	// Bound exponent parsing before using exact decimal arithmetic.
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		exponent, err := strconv.Atoi(value[index+1:])
		if err != nil || exponent < -100 || exponent > 100 {
			return "", false
		}
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) || f < 0 || f > 1e9 {
		return "", false
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		return "", false
	}
	return d.RoundCeil(10).String(), true
}

func moderationCatalogPrices(cost map[string]json.RawMessage) (ModerationV2Prices, string) {
	prices := ModerationV2Prices{}
	input, inputOK := moderationCatalogDecimal(cost["input"])
	output, outputOK := moderationCatalogDecimal(cost["output"])
	prices.Input, prices.Output = input, output
	if !inputOK || !outputOK {
		return prices, "missing_price"
	}
	if cached, exists := cost["cache_read"]; exists {
		value, ok := moderationCatalogDecimal(cached)
		if !ok {
			return prices, "invalid_price"
		}
		prices.CachedInput = value
	}
	// These tariffs cannot be represented by the current flat-rate ledger.
	// Show their base prices for reference, but never silently import them.
	for key, raw := range cost {
		switch key {
		case "input", "output", "cache_read", "input_audio", "output_audio":
		case "cache_write":
			value, ok := moderationCatalogDecimal(raw)
			if !ok || value != "0" {
				return prices, "cache_write"
			}
		case "reasoning":
			value, ok := moderationCatalogDecimal(raw)
			if !ok || value != output {
				return prices, "reasoning_price"
			}
		case "tiers", "context_over_200k":
			if string(raw) != "null" && string(raw) != "[]" && string(raw) != "{}" {
				return prices, "tiered_price"
			}
		default:
			return prices, "special_price"
		}
	}
	return prices, ""
}
