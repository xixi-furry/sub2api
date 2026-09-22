package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModerationChannelCostUsesChannelPricesAndCapability(t *testing.T) {
	svc, cfg, shared, store := v2Fixture(t, func(w http.ResponseWriter, r *http.Request) { v2Response(w, "0.1") })
	cfg.Routing = "lowest_cost"
	cfg.Providers[0].AuditValidated = true
	cfg.Providers[0].Prices.Input = "10"
	cheap := cfg.Providers[0]
	cheap.ID = "opencode-deepseek"
	cheap.Prices.Input = "0.1"
	unknown := cheap
	unknown.ID = "unknown-price"
	unknown.Prices.Input = ""
	unverified := cheap
	unverified.ID = "unverified"
	unverified.OutputLimitVerified = false
	cfg.Providers = append(cfg.Providers, unknown, unverified, cheap)
	ids, reason := moderationChannelOrder(context.Background(), cfg, "same request")
	require.Empty(t, reason)
	require.Equal(t, []string{cheap.ID, "primary"}, ids)
	result := svc.evaluateModerationV2(context.Background(), v2Input("r", "same request"), cfg, shared, "gateway", "price")
	require.Equal(t, cheap.ID, result.Verdict.ProviderID)
	require.Equal(t, 1, result.Attempts)
	require.Equal(t, cheap.ID, store.reservations[0].ProviderID)
	// A very small input cap must not silently truncate text to win on price.
	cfg.Providers[3].MaxInputTokens = 256
	ids, _ = moderationChannelOrder(context.Background(), cfg, strings.Repeat("x", 600))
	require.Equal(t, []string{"primary"}, ids)
}
func TestModerationChannelOrderIncludesOutputAndRequestFees(t *testing.T) {
	_, cfg, _, _ := v2Fixture(t, func(w http.ResponseWriter, r *http.Request) { v2Response(w, "0.1") })
	cfg.Routing = "lowest_cost"
	cfg.Providers[0].AuditValidated = true
	p := cfg.Providers[0]
	p.ID = "cheap-input-expensive-output"
	p.Prices.Input = "0"
	p.Prices.Output = "1000000"
	cfg.Providers = append(cfg.Providers, p)
	ids, _ := moderationChannelOrder(context.Background(), cfg, "hello")
	require.Equal(t, "primary", ids[0])
	cfg.Providers[1].Prices.Output = "0"
	cfg.Providers[1].Prices.PerRequest = "100"
	ids, _ = moderationChannelOrder(context.Background(), cfg, "hello")
	require.Equal(t, "primary", ids[0])
	cfg.Providers[1].Prices.PerRequest = "0"
	ids, _ = moderationChannelOrder(context.Background(), cfg, "hello")
	require.Equal(t, p.ID, ids[0])
}
func TestModerationChannelFailoverRespectsSharedCapAndCooldown(t *testing.T) {
	for _, mode := range []string{"error", "cooldown", "global-budget"} {
		t.Run(mode, func(t *testing.T) {
			svc, cfg, shared, store := v2Fixture(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) })
			backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { v2Response(w, "0.95") }))
			defer backup.Close()
			cfg.Routing = "lowest_cost"
			cfg.Providers[0].AuditValidated = true
			cfg.CacheTTLSeconds = 0
			p := cfg.Providers[0]
			p.ID = "other"
			p.BaseURL = backup.URL
			p.Prices.PerRequest = "1"
			cfg.Providers = append(cfg.Providers, p)
			if mode == "cooldown" {
				store.providerErrors = map[string]error{"primary": ErrModerationV2ProviderUnavailable}
			}
			if mode == "global-budget" {
				store.reserveErr = ErrModerationV2Budget
			}
			result := svc.evaluateModerationV2(context.Background(), v2Input("r", "hello"), cfg, shared, "gateway", "test")
			if mode == "global-budget" {
				require.Equal(t, "unresolved", result.Status)
				require.Zero(t, result.Attempts)
				return
			}
			require.Equal(t, "reviewed", result.Status)
			require.Equal(t, "other", result.Verdict.ProviderID)
			expected := 2
			if mode == "cooldown" {
				expected = 1
			}
			require.Equal(t, expected, result.Attempts)
			cfg.MaxAttempts = 1
			store.reservations = nil
			store.settlements = nil
			store.providerErrors = nil
			result = svc.evaluateModerationV2(context.Background(), v2Input("r2", "hello"), cfg, shared, "gateway", "limit")
			require.Equal(t, 1, result.Attempts)
			require.Equal(t, "unresolved", result.Status)
		})
	}
}
func TestModerationChannelUnifiedSettingsConflictLeavesSharedConfig(t *testing.T) {
	svc, cfg, _, store := v2Fixture(t, func(w http.ResponseWriter, r *http.Request) { v2Response(w, "0.1") })
	view, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, view.Channels)
	require.Empty(t, view.Channels.Providers[0].APIKeys)
	oldShared := store.settings.values[SettingKeyContentModerationConfig]
	mode := ContentModerationModeObserve
	cfg.Revision = 0
	_, err = svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{Mode: &mode, Channels: cfg})
	require.ErrorIs(t, err, ErrModerationV2Conflict)
	require.Equal(t, oldShared, store.settings.values[SettingKeyContentModerationConfig])
	cfg.Revision = 1
	cfg.Routing = "lowest_cost"
	cfg.Providers[0].AuditValidated = true
	updated, err := svc.UpdateConfig(context.Background(), UpdateContentModerationConfigInput{Mode: &mode, Channels: cfg})
	require.NoError(t, err)
	require.Equal(t, mode, updated.Mode)
	require.EqualValues(t, 2, updated.Channels.Revision)
	require.Empty(t, updated.Channels.Providers[0].APIKeys)
	var saved ModerationV2Config
	require.NoError(t, json.Unmarshal([]byte(store.settings.values[SettingKeyContentModerationV2]), &saved))
	require.Equal(t, "lowest_cost", saved.Routing)
	require.Equal(t, []string{"test-key"}, saved.Providers[0].APIKeys)
}
