//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestModerationV2PostgresAccounting(t *testing.T) {
	ctx := context.Background()
	r := &contentModerationRepository{db: integrationDB}
	reset := func() {
		_, e := integrationDB.ExecContext(ctx, `TRUNCATE fork_moderation_budgets,fork_moderation_attempts,fork_moderation_cache,fork_moderation_events`)
		require.NoError(t, e)
		_, e = integrationDB.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES($1,'{"revision":1,"enabled":true}') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, service.SettingKeyContentModerationV2)
		require.NoError(t, e)
	}
	a := func(id string) service.ModerationV2Reservation {
		return service.ModerationV2Reservation{ID: id, Revision: 1, ProviderID: "provider", Currency: "CNY", Amount: "0.03", Tokens: 100, MaxConcurrent: 100, TimeoutMS: 30000, Source: "gateway", GlobalLimits: service.ModerationV2Limits{DailyAmount: "0.30"}}
	}
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM settings WHERE key=$1`, service.SettingKeyContentModerationV2)
	})
	t.Run("concurrent_reservation_and_idempotent_settlement", func(t *testing.T) {
		reset()
		var accepted atomic.Int64
		var wg sync.WaitGroup
		var ids sync.Map
		failures := make(chan error, 50)
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				id := fmt.Sprint(i)
				e := r.ReserveModerationV2(ctx, a(id))
				if e == nil {
					accepted.Add(1)
					ids.Store(id, true)
				} else {
					failures <- e
				}
			}(i)
		}
		wg.Wait()
		close(failures)
		for e := range failures {
			require.ErrorIs(t, e, service.ErrModerationV2Budget)
		}
		require.EqualValues(t, 10, accepted.Load())
		var amount string
		var calls, tokens int64
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT amount::text,calls,tokens FROM fork_moderation_budgets WHERE scope='all'`).Scan(&amount, &calls, &tokens))
		require.True(t, decimal.RequireFromString(amount).Equal(decimal.RequireFromString("0.30")))
		require.EqualValues(t, 10, calls)
		require.EqualValues(t, 1000, tokens)
		var one string
		ids.Range(func(k, v any) bool { one = k.(string); return false })
		settlement := service.ModerationV2Settlement{ID: one, State: "settled", Amount: "0.005", Usage: &service.ModerationV2Usage{Input: 10, CachedInput: 2, Output: 5}, HTTPStatus: 200}
		require.NoError(t, r.SettleModerationV2(ctx, settlement))
		require.NoError(t, r.SettleModerationV2(ctx, settlement))
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT amount::text,calls,tokens FROM fork_moderation_budgets WHERE scope='all'`).Scan(&amount, &calls, &tokens))
		require.True(t, decimal.RequireFromString(amount).Equal(decimal.RequireFromString("0.275")))
		require.EqualValues(t, 10, calls)
		require.EqualValues(t, 915, tokens)
		// This successful settlement is not overwritten by a late timeout callback.
		require.NoError(t, r.SettleModerationV2(ctx, service.ModerationV2Settlement{ID: one, State: "unknown"}))
		summary, e := r.ModerationV2Usage(ctx, "CNY")
		require.NoError(t, e)
		require.Len(t, summary.Providers, 1)
		require.EqualValues(t, 9, summary.Providers[0].UnknownCalls)
	})
	t.Run("unknown_usage_keeps_reservation_and_concurrency_is_shared", func(t *testing.T) {
		reset()
		first := a("first")
		first.MaxConcurrent = 1
		require.NoError(t, r.ReserveModerationV2(ctx, first))
		second := a("second")
		second.MaxConcurrent = 1
		require.ErrorIs(t, r.ReserveModerationV2(ctx, second), service.ErrModerationV2ProviderUnavailable)
		require.NoError(t, r.SettleModerationV2(ctx, service.ModerationV2Settlement{ID: "first", State: "unknown"}))
		require.NoError(t, r.ReserveModerationV2(ctx, second))
		var amount string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT amount::text FROM fork_moderation_budgets WHERE scope='all'`).Scan(&amount))
		require.True(t, decimal.RequireFromString(amount).Equal(decimal.RequireFromString("0.06")))
		require.ErrorIs(t, r.ReserveModerationV2(ctx, second), service.ErrModerationV2Duplicate)
	})
	t.Run("previous_day_settlement_does_not_refund_today", func(t *testing.T) {
		reset()
		require.NoError(t, r.ReserveModerationV2(ctx, a("old")))
		_, e := integrationDB.ExecContext(ctx, `UPDATE fork_moderation_attempts SET day=day-1; UPDATE fork_moderation_budgets SET day=day-1`)
		require.NoError(t, e)
		require.NoError(t, r.ReserveModerationV2(ctx, a("today")))
		require.NoError(t, r.SettleModerationV2(ctx, service.ModerationV2Settlement{ID: "old", State: "settled", Amount: "0.001", Usage: &service.ModerationV2Usage{Input: 1, Output: 1}}))
		var amount string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT amount::text FROM fork_moderation_budgets WHERE scope='all' AND day=(now() AT TIME ZONE 'UTC')::date`).Scan(&amount))
		require.True(t, decimal.RequireFromString(amount).Equal(decimal.RequireFromString("0.03")))
	})
	t.Run("caps_cooldown_and_changed_configuration", func(t *testing.T) {
		reset()
		zero := a("zero")
		zero.GlobalLimits.DailyAmount = "0"
		zero.Amount = "0"
		require.ErrorIs(t, r.ReserveModerationV2(ctx, zero), service.ErrModerationV2Budget)
		limited := a("limited")
		limited.ProviderLimits.DailyTokens = 99
		require.ErrorIs(t, r.ReserveModerationV2(ctx, limited), service.ErrModerationV2ProviderUnavailable)
		require.NoError(t, r.ReserveModerationV2(ctx, a("over")))
		require.NoError(t, r.SettleModerationV2(ctx, service.ModerationV2Settlement{ID: "over", State: "settled", Amount: "0.04", Usage: &service.ModerationV2Usage{Input: 100, Output: 100}, LimitExceeded: true}))
		require.ErrorIs(t, r.ReserveModerationV2(ctx, a("next")), service.ErrModerationV2ProviderUnavailable)
		require.NoError(t, r.SaveModerationV2Config(ctx, 1, `{"revision":2,"enabled":false}`))
		require.ErrorIs(t, r.ReserveModerationV2(ctx, a("stale")), service.ErrModerationV2Conflict)
		require.ErrorIs(t, r.SaveModerationV2Config(ctx, 1, `{"revision":2}`), service.ErrModerationV2Conflict)
	})
	t.Run("cache_ttl_revision_and_event_deduplication", func(t *testing.T) {
		reset()
		v := service.ModerationV2Verdict{Flagged: true, Score: .95, ProviderID: "provider"}
		require.NoError(t, r.PutModerationV2Cache(ctx, "key", v, 60))
		read, e := r.GetModerationV2Cache(ctx, "key", 1)
		require.NoError(t, e)
		require.NotNil(t, read)
		read, e = r.GetModerationV2Cache(ctx, "key", 2)
		require.NoError(t, e)
		require.Nil(t, read)
		_, e = integrationDB.ExecContext(ctx, `UPDATE fork_moderation_cache SET expires_at=now()-interval '1 second'`)
		require.NoError(t, e)
		read, e = r.GetModerationV2Cache(ctx, "key", 1)
		require.NoError(t, e)
		require.Nil(t, read)
		fresh, e := r.ClaimModerationV2Event(ctx, "request", "reviewed", true, "")
		require.NoError(t, e)
		require.True(t, fresh)
		fresh, e = r.ClaimModerationV2Event(ctx, "request", "reviewed", true, "")
		require.NoError(t, e)
		require.False(t, fresh)
		summary, e := r.ModerationV2Usage(ctx, "CNY")
		require.NoError(t, e)
		require.EqualValues(t, 1, summary.CacheHits)
	})
}

func TestModerationChannelsAtomicSettings(t *testing.T) {
	ctx := context.Background()
	r := &contentModerationRepository{db: integrationDB}
	keys := []string{service.SettingKeyContentModerationV2, service.SettingKeyContentModerationConfig}
	previous := map[string]string{}
	for _, key := range keys {
		var raw string
		if integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1`, key).Scan(&raw) == nil {
			previous[key] = raw
		}
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if raw, ok := previous[key]; ok {
				_, _ = integrationDB.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, key, raw)
			} else {
				_, _ = integrationDB.ExecContext(ctx, `DELETE FROM settings WHERE key=$1`, key)
			}
		}
	})
	_, err := integrationDB.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES($1,'{"revision":1}') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, keys[0])
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES($1,'{"mode":"pre_block"}') ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, keys[1])
	require.NoError(t, err)
	require.ErrorIs(t, r.SaveContentModerationChannels(ctx, 0, `{"revision":2}`, `{"mode":"observe"}`), service.ErrModerationV2Conflict)
	var shared string
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1`, keys[1]).Scan(&shared))
	require.JSONEq(t, `{"mode":"pre_block"}`, shared)
	require.NoError(t, r.SaveContentModerationChannels(ctx, 1, `{"revision":2}`, `{"mode":"observe"}`))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1`, keys[1]).Scan(&shared))
	require.JSONEq(t, `{"mode":"observe"}`, shared)
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1`, keys[0]).Scan(&shared))
	require.JSONEq(t, `{"revision":2}`, shared)
}
