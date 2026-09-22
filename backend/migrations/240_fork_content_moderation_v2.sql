-- Additive fork tables. UTC daily buckets; decimal currency amounts.
CREATE TABLE IF NOT EXISTS fork_moderation_budgets (
 day date NOT NULL, scope varchar(80) NOT NULL, currency varchar(3) NOT NULL,
 amount numeric(30,10) NOT NULL DEFAULT 0 CHECK (amount >= 0),
 calls bigint NOT NULL DEFAULT 0 CHECK (calls >= 0),
 tokens bigint NOT NULL DEFAULT 0 CHECK (tokens >= 0),
 PRIMARY KEY(day,scope,currency)
);
CREATE TABLE IF NOT EXISTS fork_moderation_attempts (
 id varchar(64) PRIMARY KEY, day date NOT NULL, provider_id varchar(64) NOT NULL,
 currency varchar(3) NOT NULL, source varchar(16) NOT NULL,
 reserved_amount numeric(30,10) NOT NULL, reserved_tokens bigint NOT NULL,
 actual_amount numeric(30,10), input_tokens bigint, cached_tokens bigint, output_tokens bigint,
 state varchar(16) NOT NULL DEFAULT 'pending', limit_exceeded boolean NOT NULL DEFAULT false, http_status integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL,
 settled_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_fork_moderation_attempts_day ON fork_moderation_attempts(day,provider_id,currency);
CREATE INDEX IF NOT EXISTS idx_fork_moderation_attempts_pending ON fork_moderation_attempts(provider_id,expires_at) WHERE state='pending';
CREATE TABLE IF NOT EXISTS fork_moderation_cache (
 key varchar(64) PRIMARY KEY, verdict jsonb NOT NULL, expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fork_moderation_cache_expiry ON fork_moderation_cache(expires_at);
CREATE TABLE IF NOT EXISTS fork_moderation_events (
 id varchar(64) PRIMARY KEY, status varchar(16) NOT NULL, cache_hit boolean NOT NULL,
 reason varchar(120) NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_fork_moderation_events_created ON fork_moderation_events(created_at);
