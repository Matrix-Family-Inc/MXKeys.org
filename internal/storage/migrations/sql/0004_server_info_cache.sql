-- 0004_server_info_cache.sql
--
-- Backing cache for the /_mxkeys/server-info enrichment endpoint.
-- A single row per Matrix server_name holds the last successful
-- lookup result (DNS + federation reachability + WHOIS).
-- The handler serves cached rows while they are still fresh and
-- schedules a re-fetch when they cross the TTL.
--
-- Schema:
--   server_name   - the Matrix server_name as canonicalised by
--                   internal/server/validation.go; primary key.
--   info          - JSON body returned to the client. Opaque to
--                   the database; shape owned by
--                   internal/server/serverinfo_types.go.
--   fetched_at    - timestamp of the last successful orchestration.
--   expires_at    - fetched_at plus server_info.cache_ttl. Used by
--                   the handler as the freshness pivot and by
--                   DeleteExpiredServerInfo() during cleanup.
--
-- Idempotent. Safe to re-run on an already-upgraded schema.

CREATE TABLE IF NOT EXISTS server_info_cache (
    server_name TEXT PRIMARY KEY,
    info        JSONB NOT NULL,
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_server_info_cache_expires
    ON server_info_cache (expires_at);
