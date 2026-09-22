package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

var _ service.ModerationV2Repository = (*contentModerationRepository)(nil)

func (r *contentModerationRepository) SaveModerationV2Config(ctx context.Context, expected int64, raw string) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback() }()
	if _, e = tx.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES($1,'{"revision":0}') ON CONFLICT(key) DO NOTHING`, service.SettingKeyContentModerationV2); e != nil {
		return e
	}
	var previous string
	if e = tx.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1 FOR UPDATE`, service.SettingKeyContentModerationV2).Scan(&previous); e != nil {
		return e
	}
	var c service.ModerationV2Config
	if e = json.Unmarshal([]byte(previous), &c); e != nil {
		return e
	}
	if c.Revision != expected {
		return service.ErrModerationV2Conflict
	}
	if _, e = tx.ExecContext(ctx, `UPDATE settings SET value=$2,updated_at=now() WHERE key=$1`, service.SettingKeyContentModerationV2, raw); e != nil {
		return e
	}
	return tx.Commit()
}

func (r *contentModerationRepository) ReserveModerationV2(ctx context.Context, a service.ModerationV2Reservation) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback() }()
	var raw string
	if e = tx.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=$1 FOR SHARE`, service.SettingKeyContentModerationV2).Scan(&raw); e != nil {
		return e
	}
	var cfg service.ModerationV2Config
	if e = json.Unmarshal([]byte(raw), &cfg); e != nil {
		return e
	}
	if cfg.Revision != a.Revision || (!cfg.Enabled && a.Source != "admin_test") {
		return service.ErrModerationV2Conflict
	}
	// Database time defines the bucket, even when application clocks differ.
	var day string
	if e = tx.QueryRowContext(ctx, `SELECT (now() AT TIME ZONE 'UTC')::date::text`).Scan(&day); e != nil {
		return e
	}
	// Cross-midnight reservations still share the provider concurrency lock.
	if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "fork-moderation:"+a.ProviderID); e != nil {
		return e
	}
	scopes := []string{"all", "provider:" + a.ProviderID}
	limits := []service.ModerationV2Limits{a.GlobalLimits, a.ProviderLimits}
	for i, scope := range scopes {
		if _, e = tx.ExecContext(ctx, `INSERT INTO fork_moderation_budgets(day,scope,currency) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, day, scope, a.Currency); e != nil {
			return e
		}
		var amount string
		var calls, tokens int64
		if e = tx.QueryRowContext(ctx, `SELECT amount::text,calls,tokens FROM fork_moderation_budgets WHERE day=$1 AND scope=$2 AND currency=$3 FOR UPDATE`, day, scope, a.Currency).Scan(&amount, &calls, &tokens); e != nil {
			return e
		}
		current, e := decimal.NewFromString(amount)
		if e != nil {
			return e
		}
		reserve, e := decimal.NewFromString(a.Amount)
		if e != nil || reserve.IsNegative() {
			return errors.New("invalid reservation")
		}
		l := limits[i]
		if (l.DailyCalls > 0 && calls >= l.DailyCalls) || (l.DailyTokens > 0 && tokens+a.Tokens > l.DailyTokens) {
			if i == 1 {
				return service.ErrModerationV2ProviderUnavailable
			}
			return service.ErrModerationV2Budget
		}
		if l.DailyAmount != "" {
			cap, e := decimal.NewFromString(l.DailyAmount)
			if e != nil {
				return e
			}
			if cap.IsZero() || current.Add(reserve).GreaterThan(cap) {
				if i == 1 {
					return service.ErrModerationV2ProviderUnavailable
				}
				return service.ErrModerationV2Budget
			}
		}
	}
	// The all-bucket lock serializes reservations across nodes. Same attempt IDs
	// never make a second HTTP call, including after a process restart.
	var exists bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM fork_moderation_attempts WHERE id=$1)`, a.ID).Scan(&exists); e != nil {
		return e
	}
	if exists {
		return service.ErrModerationV2Duplicate
	}
	var active int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM fork_moderation_attempts WHERE provider_id=$1 AND state='pending' AND expires_at>now()`, a.ProviderID).Scan(&active); e != nil {
		return e
	}
	if active >= a.MaxConcurrent {
		return service.ErrModerationV2ProviderUnavailable
	}
	var unavailable bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM fork_moderation_attempts WHERE provider_id=$1 AND (limit_exceeded OR http_status>=400) AND ((day=$2 AND limit_exceeded) OR (http_status IN (401,403) AND created_at>now()-interval '10 minutes') OR (http_status IN (429,529) AND created_at>now()-interval '1 minute') OR (http_status>=500 AND created_at>now()-interval '10 seconds')))`, a.ProviderID, day).Scan(&unavailable); e != nil {
		return e
	}
	if unavailable {
		return service.ErrModerationV2ProviderUnavailable
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO fork_moderation_attempts(id,day,provider_id,currency,source,reserved_amount,reserved_tokens,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,now()+$8*interval '1 millisecond')`, a.ID, day, a.ProviderID, a.Currency, a.Source, a.Amount, a.Tokens, a.TimeoutMS+5000); e != nil {
		return e
	}
	for _, scope := range scopes {
		if _, e = tx.ExecContext(ctx, `UPDATE fork_moderation_budgets SET amount=amount+$4,calls=calls+1,tokens=tokens+$5 WHERE day=$1 AND scope=$2 AND currency=$3`, day, scope, a.Currency, a.Amount, a.Tokens); e != nil {
			return e
		}
	}
	return tx.Commit()
}

func (r *contentModerationRepository) SettleModerationV2(ctx context.Context, s service.ModerationV2Settlement) error {
	tx, e := r.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback() }()
	// Read immutable routing data first, lock budgets in the same order as reserve,
	// then lock the attempt. This avoids reserve/settle deadlocks.
	var day, provider, currency string
	if e = tx.QueryRowContext(ctx, `SELECT day::text,provider_id,currency FROM fork_moderation_attempts WHERE id=$1`, s.ID).Scan(&day, &provider, &currency); e != nil {
		return e
	}
	for _, scope := range []string{"all", "provider:" + provider} {
		var ignored string
		if e = tx.QueryRowContext(ctx, `SELECT amount::text FROM fork_moderation_budgets WHERE day=$1 AND scope=$2 AND currency=$3 FOR UPDATE`, day, scope, currency).Scan(&ignored); e != nil {
			return e
		}
	}
	var state, reserved string
	var tokens int64
	if e = tx.QueryRowContext(ctx, `SELECT state,reserved_amount::text,reserved_tokens FROM fork_moderation_attempts WHERE id=$1 FOR UPDATE`, s.ID).Scan(&state, &reserved, &tokens); e != nil {
		return e
	}
	if state == "settled" || state == "usage_only" {
		return tx.Commit()
	}
	if s.State != "settled" || s.Usage == nil {
		state = "unknown"
		if s.Usage != nil {
			state = "usage_only"
		}
		var in, cache, out any
		if s.Usage != nil {
			in = s.Usage.Input
			cache = s.Usage.CachedInput
			out = s.Usage.Output
		}
		_, e = tx.ExecContext(ctx, `UPDATE fork_moderation_attempts SET state=$2,http_status=$3,input_tokens=$4,cached_tokens=$5,output_tokens=$6,limit_exceeded=$7,settled_at=now() WHERE id=$1`, s.ID, state, s.HTTPStatus, in, cache, out, s.LimitExceeded)
		if e != nil {
			return e
		}
		return tx.Commit()
	}
	actual, e := decimal.NewFromString(s.Amount)
	if e != nil || actual.IsNegative() {
		return errors.New("invalid settlement")
	}
	prev, e := decimal.NewFromString(reserved)
	if e != nil {
		return e
	}
	for _, scope := range []string{"all", "provider:" + provider} {
		if _, e = tx.ExecContext(ctx, `UPDATE fork_moderation_budgets SET amount=GREATEST(0,amount+$4),tokens=GREATEST(0,tokens+$5) WHERE day=$1 AND scope=$2 AND currency=$3`, day, scope, currency, actual.Sub(prev).String(), s.Usage.Input+s.Usage.Output-tokens); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, `UPDATE fork_moderation_attempts SET state='settled',actual_amount=$2,input_tokens=$3,cached_tokens=$4,output_tokens=$5,http_status=$6,limit_exceeded=$7,settled_at=now() WHERE id=$1`, s.ID, s.Amount, s.Usage.Input, s.Usage.CachedInput, s.Usage.Output, s.HTTPStatus, s.LimitExceeded); e != nil {
		return e
	}
	return tx.Commit()
}
func (r *contentModerationRepository) GetModerationV2Cache(ctx context.Context, key string, revision int64) (*service.ModerationV2Verdict, error) {
	var raw []byte
	e := r.db.QueryRowContext(ctx, `SELECT verdict FROM fork_moderation_cache WHERE key=$1 AND expires_at>now() AND EXISTS(SELECT 1 FROM settings WHERE key='content_moderation_v2' AND (value::jsonb->>'revision')::bigint=$2 AND (value::jsonb->>'enabled')::boolean)`, key, revision).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var v service.ModerationV2Verdict
	if e = json.Unmarshal(raw, &v); e != nil {
		return nil, e
	}
	return &v, nil
}
func (r *contentModerationRepository) PutModerationV2Cache(ctx context.Context, key string, v service.ModerationV2Verdict, ttl int) error {
	if ttl <= 0 {
		return nil
	}
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	_, e = r.db.ExecContext(ctx, `INSERT INTO fork_moderation_cache(key,verdict,expires_at) VALUES($1,$2::jsonb,now()+$3*interval '1 second') ON CONFLICT(key) DO UPDATE SET verdict=EXCLUDED.verdict,expires_at=EXCLUDED.expires_at`, key, string(raw), ttl)
	if e == nil && len(key) >= 2 && key[:2] == "00" {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM fork_moderation_cache WHERE key IN (SELECT key FROM fork_moderation_cache WHERE expires_at<now() LIMIT 1000)`)
	}
	return e
}
func (r *contentModerationRepository) ClaimModerationV2Event(ctx context.Context, id, status string, hit bool, reason string) (bool, error) {
	result, e := r.db.ExecContext(ctx, `INSERT INTO fork_moderation_events(id,status,cache_hit,reason) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, id, status, hit, reason)
	if e != nil {
		return false, e
	}
	n, e := result.RowsAffected()
	return n > 0, e
}
func (r *contentModerationRepository) ModerationV2Usage(ctx context.Context, currency string) (*service.ModerationV2UsageSummary, error) {
	out := &service.ModerationV2UsageSummary{Currency: currency, Providers: []service.ModerationV2UsageRow{}}
	if e := r.db.QueryRowContext(ctx, `SELECT (now() AT TIME ZONE 'UTC')::date::text`).Scan(&out.Day); e != nil {
		return nil, e
	}
	rows, e := r.db.QueryContext(ctx, `SELECT provider_id,count(*),COALESCE(sum(CASE WHEN state='settled' THEN actual_amount ELSE reserved_amount END),0)::text,COALESCE(sum(actual_amount),0)::text,count(*) FILTER(WHERE state<>'settled'),COALESCE(sum(input_tokens),0),COALESCE(sum(output_tokens),0),COALESCE(sum(cached_tokens),0) FROM fork_moderation_attempts WHERE day=$1 AND currency=$2 GROUP BY provider_id ORDER BY provider_id`, out.Day, currency)
	if e != nil {
		return nil, e
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v service.ModerationV2UsageRow
		if e = rows.Scan(&v.ProviderID, &v.Calls, &v.HeldAmount, &v.ConfirmedAmount, &v.UnknownCalls, &v.Input, &v.Output, &v.CachedInput); e != nil {
			return nil, e
		}
		out.Providers = append(out.Providers, v)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	e = r.db.QueryRowContext(ctx, `SELECT count(*) FILTER(WHERE cache_hit),count(*) FILTER(WHERE status='unresolved') FROM fork_moderation_events WHERE created_at >= $1::date AT TIME ZONE 'UTC' AND created_at < ($1::date+1) AT TIME ZONE 'UTC'`, out.Day).Scan(&out.CacheHits, &out.Unresolved)
	return out, e
}

// Called by the existing maintenance cycle. Never remove today's unsettled
// reservations; old daily buckets remain immutable historical totals.
func (r *contentModerationRepository) cleanupModerationV2(ctx context.Context) {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM fork_moderation_cache WHERE key IN (SELECT key FROM fork_moderation_cache WHERE expires_at<now() LIMIT 5000)`)
	cutoff := time.Now().UTC().AddDate(0, 0, -90)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM fork_moderation_events WHERE id IN (SELECT id FROM fork_moderation_events WHERE created_at<$1 LIMIT 5000)`, cutoff)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM fork_moderation_attempts WHERE id IN (SELECT id FROM fork_moderation_attempts WHERE day<$1::date LIMIT 5000)`, cutoff)
}
