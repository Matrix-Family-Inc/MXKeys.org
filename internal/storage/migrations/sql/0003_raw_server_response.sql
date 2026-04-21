-- 0003_raw_server_response.sql
--
-- Adds a raw_response column to server_key_responses so the notary can
-- preserve origin-delivered canonical JSON byte-for-byte across cache
-- and replication boundaries. Without this column the notary had to
-- re-marshal the parsed struct on every read, which silently dropped
-- omitempty-tagged fields (notably `old_verify_keys: {}`) and broke
-- origin self-signature verification for downstream clients.
--
-- Semantics:
--   * raw_response is NULL for rows written by older code. Reader
--     code keeps a struct-only fallback path for those rows and will
--     backfill raw_response on the next successful fetch.
--   * raw_response holds the raw bytes delivered by origin (or by a
--     trusted fallback notary) for /_matrix/key/v2/server. Callers
--     treat the bytes as canonical.
--
-- This migration is idempotent: running it on an already-upgraded
-- schema is a no-op.

ALTER TABLE server_key_responses
    ADD COLUMN IF NOT EXISTS raw_response BYTEA;
