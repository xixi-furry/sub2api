package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type priceCatalogTransport func(*http.Request) (*http.Response, error)

func (fn priceCatalogTransport) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }
func priceCatalogReply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

const priceCatalogFixture = `{
  "opencode":{"name":"OpenCode Zen","api":"https://opencode.ai/zen/v1","models":{"deepseek-flash":{"name":"DeepSeek Flash","cost":{"input":0.1,"output":0.2,"cache_read":0}}}},
  "opencode-go":{"name":"OpenCode Go","api":"https://opencode.ai/zen/go/v1","models":{"deepseek-flash":{"cost":{"input":0.15,"output":0.6}}}},
  "deepseek":{"name":"DeepSeek","api":"https://api.deepseek.com","models":{"deepseek-flash":{"cost":{"input":0.5,"output":1}}}},
  "openai":{"models":{"same-model":{"cost":{"input":2,"output":4}}}}
}`

func priceRegistry(t *testing.T) map[string]modelsDevProvider {
	t.Helper()
	var registry map[string]modelsDevProvider
	require.NoError(t, json.Unmarshal([]byte(priceCatalogFixture), &registry))
	return registry
}
func TestModerationPriceProviderMatch(t *testing.T) {
	registry := priceRegistry(t)
	for _, tc := range []struct{ url, id string }{
		{"https://opencode.ai/zen/v1/chat/completions", "opencode"},
		{"https://opencode.ai/zen/go/v1/responses", "opencode-go"},
		{"https://opencode.ai", ""}, {"https://opencode.ai/zen", ""},
		{"https://api.deepseek.com/v1", "deepseek"},
		{"https://api.openai.com/v1/responses", "openai"},
		{"https://chatgpt.com/backend-api/codex", ""},
		{"https://api.openai.com.evil.example/v1", ""},
		{"https://unknown.example/v1", ""}, {"https://user:secret@api.openai.com/v1", ""},
	} {
		t.Run(tc.url, func(t *testing.T) { require.Equal(t, tc.id, moderationPriceProviderID(registry, tc.url)) })
	}
	result := searchModerationModelPrices(registry, "deepseek-flash", "https://opencode.ai/zen/v1")
	require.Equal(t, 3, result.Total)
	require.Equal(t, "opencode", result.Items[0].ProviderID)
	require.Equal(t, "0.1", result.Items[0].Prices.Input)
	require.Equal(t, "0", result.Items[0].Prices.CachedInput)
	require.True(t, result.Items[0].MatchesChannel)
	require.False(t, result.Items[1].MatchesChannel)
}
func TestModerationCatalogPricesDoNotInventFreeOrFlattenSpecialRates(t *testing.T) {
	for _, tc := range []struct{ raw, reason string }{
		{`{"input":0,"output":0}`, ""},
		{`{"input":1e-7,"output":0.2,"reasoning":0.2}`, ""},
		{`{"output":1}`, "missing_price"},
		{`{"input":null,"output":1}`, "missing_price"},
		{`{"input":-1,"output":1}`, "missing_price"},
		{`{"input":1e-2147483647,"output":1}`, "missing_price"},
		{`{"input":0.1,"output":0.2,"cache_read":null}`, "invalid_price"},
		{`{"input":0.1,"output":0.2,"cache_write":0.1}`, "cache_write"},
		{`{"input":0.1,"output":0.2,"reasoning":0.4}`, "reasoning_price"},
		{`{"input":0.1,"output":0.2,"tiers":[{"input":1}]}`, "tiered_price"},
		{`{"input":0.1,"output":0.2,"context_over_200k":{"input":1}}`, "tiered_price"},
		{`{"input":0.1,"output":0.2,"per_call":1}`, "special_price"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			var cost map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &cost))
			_, reason := moderationCatalogPrices(cost)
			require.Equal(t, tc.reason, reason)
		})
	}
	var cost map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(`{"input":1e-7,"output":0.12345678901}`), &cost))
	prices, reason := moderationCatalogPrices(cost)
	require.Empty(t, reason)
	require.Equal(t, "0.0000001", prices.Input)
	require.Equal(t, "0.1234567891", prices.Output)
	require.Empty(t, prices.CachedInput)
}
func TestModerationPriceCatalogCachesAndDoesNotForwardChannelData(t *testing.T) {
	var calls atomic.Int32
	var badRequest atomic.Bool
	svc := &ContentModerationService{httpClient: &http.Client{Transport: priceCatalogTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.URL.String() != modelsDevRegistryURL || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Body != nil || r.Header.Get("User-Agent") == "" {
			badRequest.Store(true)
		}
		return priceCatalogReply(200, priceCatalogFixture), nil
	})}}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() { defer group.Done(); _, _, _, _ = svc.loadModerationPriceCatalog(context.Background()) }()
	}
	group.Wait()
	result, err := svc.LookupModerationModelPrices(context.Background(), "deepseek-flash", "https://opencode.ai/zen/v1")
	require.NoError(t, err)
	require.Equal(t, "USD", result.Currency)
	require.False(t, result.Stale)
	require.EqualValues(t, 1, calls.Load())
	require.False(t, badRequest.Load())
	svc.priceCatalog.fetchedAt = time.Now().Add(-7 * time.Hour)
	svc.httpClient.Transport = priceCatalogTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		return priceCatalogReply(503, "offline"), nil
	})
	result, err = svc.LookupModerationModelPrices(context.Background(), "deepseek-flash", "https://opencode.ai/zen/v1")
	require.NoError(t, err)
	require.True(t, result.Stale)
	_, err = svc.LookupModerationModelPrices(context.Background(), "deepseek-flash", "")
	require.NoError(t, err)
	require.EqualValues(t, 2, calls.Load())
	svc.priceCatalog.fetchedAt = time.Now().Add(-8 * 24 * time.Hour)
	_, err = svc.LookupModerationModelPrices(context.Background(), "deepseek-flash", "")
	require.Error(t, err)
}
func TestModerationPriceCatalogRejectsBadResponses(t *testing.T) {
	for _, body := range []string{"not-json", "{}", strings.Repeat("x", moderationPriceCatalogLimit+1)} {
		svc := &ContentModerationService{httpClient: &http.Client{Transport: priceCatalogTransport(func(*http.Request) (*http.Response, error) { return priceCatalogReply(200, body), nil })}}
		_, err := svc.LookupModerationModelPrices(context.Background(), "model", "")
		require.Error(t, err)
	}
	svc := &ContentModerationService{httpClient: &http.Client{Transport: priceCatalogTransport(func(*http.Request) (*http.Response, error) {
		res := priceCatalogReply(302, "")
		res.Header.Set("Location", "http://127.0.0.1/private")
		return res, nil
	})}}
	_, err := svc.LookupModerationModelPrices(context.Background(), "model", "")
	require.Error(t, err)
	_, err = svc.LookupModerationModelPrices(context.Background(), "", "")
	require.Error(t, err)
}
func TestModerationPriceCatalogLive(t *testing.T) {
	if os.Getenv("MODELS_DEV_LIVE") != "1" {
		t.Skip("opt-in public catalog check")
	}
	svc := &ContentModerationService{}
	result, err := svc.LookupModerationModelPrices(context.Background(), "deepseek-v4-flash", "https://opencode.ai/zen/v1")
	require.NoError(t, err)
	require.NotEmpty(t, result.Items)
	require.Equal(t, "opencode", result.MatchedProviderID)
	require.True(t, result.Items[0].MatchesChannel)
	require.Equal(t, "USD", result.Currency)
}
