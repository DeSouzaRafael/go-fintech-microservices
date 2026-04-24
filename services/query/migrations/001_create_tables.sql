CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS balance_projections (
    wallet_id     UUID PRIMARY KEY,
    user_id       UUID NOT NULL,
    currency      TEXT NOT NULL,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    version       BIGINT NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_balance_projections_user_id ON balance_projections(user_id);

CREATE TABLE IF NOT EXISTS statement_entries (
    event_id     UUID PRIMARY KEY,
    wallet_id    UUID NOT NULL,
    type         TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    occurred_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_statement_entries_wallet_id ON statement_entries(wallet_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS user_stats (
    user_id              UUID PRIMARY KEY,
    total_transactions   BIGINT NOT NULL DEFAULT 0,
    total_deposit_cents  BIGINT NOT NULL DEFAULT 0,
    total_withdraw_cents BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS query_processed_events (
    event_id     UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
