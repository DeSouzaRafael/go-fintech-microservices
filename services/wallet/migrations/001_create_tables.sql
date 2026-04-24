CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS wallet_events (
    id         UUID PRIMARY KEY,
    wallet_id  UUID NOT NULL,
    type       TEXT NOT NULL,
    payload    JSONB NOT NULL,
    version    BIGINT NOT NULL,
    occured_at TIMESTAMPTZ NOT NULL,
    UNIQUE (wallet_id, version)
);

CREATE INDEX IF NOT EXISTS idx_wallet_events_wallet_id_version ON wallet_events(wallet_id, version);

CREATE TABLE IF NOT EXISTS wallet_snapshots (
    wallet_id     UUID PRIMARY KEY,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    reserved      BIGINT NOT NULL DEFAULT 0,
    currency      TEXT NOT NULL,
    user_id       UUID NOT NULL,
    version       BIGINT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
