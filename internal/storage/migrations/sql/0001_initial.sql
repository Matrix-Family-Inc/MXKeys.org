-- 0001_initial.sql
-- Initial schema for MXKeys notary cache.
--
-- Tables:
--   server_keys           per-key rows, keyed by (server_name, key_id)
--   server_key_responses  whole-response cache, keyed by server_name
--
-- All statements are idempotent (IF NOT EXISTS) so re-running on an existing
-- schema is safe. Future schema changes must use a new numbered migration
-- file and must not modify this one.

CREATE TABLE IF NOT EXISTS server_keys (
    server_name  TEXT NOT NULL,
    key_id       TEXT NOT NULL,
    public_key   BYTEA NOT NULL,
    valid_until  TIMESTAMPTZ NOT NULL,
    fetched_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    raw_response JSONB,
    PRIMARY KEY (server_name, key_id)
);

CREATE INDEX IF NOT EXISTS idx_server_keys_server ON server_keys (server_name);
CREATE INDEX IF NOT EXISTS idx_server_keys_valid  ON server_keys (valid_until);

CREATE TABLE IF NOT EXISTS server_key_responses (
    server_name TEXT PRIMARY KEY,
    response    JSONB NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
