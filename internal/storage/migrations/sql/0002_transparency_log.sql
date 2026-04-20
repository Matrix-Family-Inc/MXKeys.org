-- 0002_transparency_log.sql
-- Transparency log table for the default operator configuration
-- (transparency.table_name = "key_transparency_log"). Table name is
-- fixed here to bring transparency DDL under migrations runner
-- ownership per ADR-0008.
--
-- Operators who set transparency.table_name to a non-default value
-- continue to create their table lazily via the legacy initTable()
-- path in internal/keys/transparency.go, which logs a deprecation
-- warning. The custom-name path is documented as a tolerated
-- exception until a proper namespaced-migration mechanism lands.
--
-- Idempotent: safe to re-run on an existing schema.

CREATE TABLE IF NOT EXISTS key_transparency_log (
    id             BIGSERIAL PRIMARY KEY,
    timestamp      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    server_name    TEXT NOT NULL,
    key_id         TEXT,
    event_type     TEXT NOT NULL,
    details        TEXT,
    key_hash       TEXT,
    valid_until_ts BIGINT,
    previous_hash  TEXT,
    entry_hash     TEXT NOT NULL,
    CONSTRAINT key_transparency_log_entry_hash_unique UNIQUE (entry_hash)
);

CREATE INDEX IF NOT EXISTS key_transparency_log_server_name_idx ON key_transparency_log (server_name);
CREATE INDEX IF NOT EXISTS key_transparency_log_timestamp_idx   ON key_transparency_log (timestamp);
CREATE INDEX IF NOT EXISTS key_transparency_log_event_type_idx  ON key_transparency_log (event_type);
