package service

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type v2AllowlistTestStore struct {
	*v2TestStore
	penaltyChecks atomic.Int64
}

func (s *v2AllowlistTestStore) CountFlaggedByUserSince(context.Context, int64, time.Time, bool) (int, error) {
	s.penaltyChecks.Add(1)
	return 1, nil
}

func saveV2AllowlistFixture(t *testing.T, cfg *ModerationV2Config, shared *ContentModerationConfig, store *v2TestStore) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	require.NoError(t, err)
	store.settings.values[SettingKeyContentModerationV2] = string(raw)
	raw, err = json.Marshal(shared)
	require.NoError(t, err)
	store.settings.values[SettingKeyContentModerationConfig] = string(raw)
	store.settings.values[SettingKeyCyberPolicyUserAllowlist] = "1"
}

func assertV2AllowlistAudit(t *testing.T, svc *ContentModerationService, store *v2AllowlistTestStore, flagged bool) {
	t.Helper()
	require.Len(t, svc.asyncQueue, 1)
	task := <-svc.asyncQueue
	require.NotNil(t, task.log)
	require.True(t, task.input.riskControlLogOnly)
	require.Equal(t, ContentModerationModeRiskControlLogOnly, task.log.Mode)
	require.Equal(t, ContentModerationActionAllow, task.log.Action)
	require.Equal(t, flagged, task.log.Flagged)
	svc.persistContentModerationLog(context.Background(), task.config, task.log, task.inputHash, task.recordHash, task.applySideEffects)
	require.Zero(t, store.penaltyChecks.Load())
	logs := store.snapshotLogs()
	require.Len(t, logs, 1)
	require.False(t, logs[0].AutoBanned)
	require.False(t, logs[0].EmailSent)
	require.Zero(t, logs[0].ViolationCount)
}

func TestModerationV2AllowlistKeepsAuditAndBudgetWithoutBlocking(t *testing.T) {
	for _, mode := range []string{"compat", "balanced"} {
		for _, outcome := range []string{"flagged", "invalid", "budget"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				var calls atomic.Int64
				handler := func(w http.ResponseWriter, _ *http.Request) {
					calls.Add(1)
					if outcome == "invalid" {
						_, _ = w.Write([]byte(`{}`))
					} else if mode == "balanced" {
						enhancedResponse(w, "block", .99)
					} else {
						v2Response(w, "0.99")
					}
				}
				fixture := v2Fixture
				if mode == "balanced" {
					fixture = enhancedFixture
				}
				svc, cfg, shared, baseStore := fixture(t, handler)
				shared.AutoBanEnabled, shared.EmailOnHit = true, true
				shared.BanThreshold = 1
				saveV2AllowlistFixture(t, cfg, shared, baseStore)
				store := &v2AllowlistTestStore{v2TestStore: baseStore}
				svc.repo = store
				if outcome == "budget" {
					store.reserveErr = ErrModerationV2Budget
				}
				decision, err := svc.Check(context.Background(), v2Input(mode+outcome, "audit this input"))
				require.NoError(t, err)
				require.True(t, decision.Allowed)
				require.False(t, decision.Blocked)
				require.Equal(t, outcome == "flagged", decision.Flagged)
				require.Equal(t, ContentModerationActionAllow, decision.Action)
				require.Zero(t, decision.StatusCode)
				require.Empty(t, decision.Message)
				require.Empty(t, decision.ErrorCode)
				require.Zero(t, svc.preBlockBlocked.Load())
				require.EqualValues(t, 1, svc.preBlockAllowed.Load())
				require.Zero(t, svc.preBlockErrors.Load())
				require.Len(t, store.events, 1)
				if outcome == "budget" {
					require.Zero(t, calls.Load())
					require.Empty(t, store.reservations)
				} else {
					require.Positive(t, calls.Load())
					require.Len(t, store.reservations, int(calls.Load()))
					require.Len(t, store.settlements, int(calls.Load()))
					if outcome == "invalid" {
						require.Equal(t, "unknown", store.settlements[0].State)
					} else {
						require.Equal(t, "settled", store.settlements[0].State)
					}
				}
				assertV2AllowlistAudit(t, svc, store, outcome == "flagged")
			})
		}
	}
}

func TestModerationV2AllowlistRemovalReappliesCachedVerdict(t *testing.T) {
	var calls atomic.Int64
	svc, cfg, shared, baseStore := v2Fixture(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		v2Response(w, "0.99")
	})
	shared.AutoBanEnabled = true
	shared.BanThreshold = 1
	saveV2AllowlistFixture(t, cfg, shared, baseStore)
	store := &v2AllowlistTestStore{v2TestStore: baseStore}
	svc.repo = store
	input := v2Input("trusted", "same audited input")
	decision, err := svc.Check(context.Background(), input)
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	store.settings.values[SettingKeyCyberPolicyUserAllowlist] = ""
	_, err = svc.refreshRuntimeSnapshot(context.Background())
	require.NoError(t, err)
	// A delayed record keeps the membership captured when the request arrived.
	assertV2AllowlistAudit(t, svc, store, true)
	input.RequestID = "removed"
	decision, err = svc.Check(context.Background(), input)
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.True(t, decision.Blocked)
	require.True(t, decision.Flagged)
	require.Equal(t, shared.BlockStatus, decision.StatusCode)
	require.EqualValues(t, 1, calls.Load(), "reuse the verdict, not a prior exemption")
	require.EqualValues(t, 1, svc.preBlockBlocked.Load())
	task := <-svc.asyncQueue
	require.False(t, task.input.riskControlLogOnly)
	require.True(t, task.log.EngineMeta.CacheHit)
	require.Equal(t, ContentModerationModePreBlock, task.log.Mode)
}

func TestModerationV2AllowlistObserveTaskRetainsAdmissionSnapshot(t *testing.T) {
	svc, cfg, shared, baseStore := v2Fixture(t, func(w http.ResponseWriter, _ *http.Request) {
		v2Response(w, "0.99")
	})
	shared.Mode = ContentModerationModeObserve
	shared.AutoBanEnabled, shared.EmailOnHit = true, true
	shared.BanThreshold = 1
	saveV2AllowlistFixture(t, cfg, shared, baseStore)
	store := &v2AllowlistTestStore{v2TestStore: baseStore}
	svc.repo = store
	decision, err := svc.Check(context.Background(), v2Input("observe", "audit this input"))
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	require.Len(t, svc.asyncQueue, 1)
	task := <-svc.asyncQueue
	require.NotNil(t, task.v2)
	require.True(t, task.input.riskControlLogOnly)
	store.settings.values[SettingKeyCyberPolicyUserAllowlist] = ""
	_, err = svc.refreshRuntimeSnapshot(context.Background())
	require.NoError(t, err)
	decision = svc.checkModerationV2(context.Background(), task.input, task.v2, task.config, task.v2EventID)
	require.True(t, decision.Allowed)
	require.True(t, decision.Flagged)
	require.Len(t, store.reservations, 1)
	assertV2AllowlistAudit(t, svc, store, true)
}
