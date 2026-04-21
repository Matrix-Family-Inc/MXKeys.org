Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Wed Apr 22 2026 UTC
Status: Updated

# MXKeys Architecture

## Scope

This document describes the current runtime shape of MXKeys:

- HTTP request flow,
- cache and fetch path,
- transparency and analytics subsystems,
- authenticated cluster transport,
- Raft persistence (WAL + snapshots),
- signing-key provider abstraction,
- schema migration runner.

The normative API contract lives in `docs/federation-behavior.md`.

## Request Pipeline

Public request flow in `internal/server`:

1. `RequestIDMiddleware`
2. optional `RequestIDRequirementMiddleware`
3. `SecurityHeadersMiddleware`
4. request logging middleware
5. rate limiting middleware (LRU cache of per-IP token buckets, O(1) eviction)
6. route handler on `http.ServeMux`

Stable public routes:

- `GET /_matrix/key/v2/server`
- `GET /_matrix/key/v2/server/{keyID}`
- `POST /_matrix/key/v2/query`
- `GET /_matrix/federation/v1/version`
- `GET /_mxkeys/health`
- `GET /_mxkeys/live`
- `GET /_mxkeys/ready`
- `GET /_mxkeys/status`
- `GET /_mxkeys/metrics`

Protected operational routes:

- `/_mxkeys/transparency/*`
- `/_mxkeys/analytics/*`
- `/_mxkeys/cluster/*`
- `/_mxkeys/policy/*`

These routes are registered only when their feature is available and the enterprise access token is configured.

## Core Components

| Component | Responsibility |
|-----------|----------------|
| `cmd/mxkeys` | Process entry point; parses `-config`, initializes logging, wires signals |
| `internal/server` | HTTP routing, middleware, status, protected operational endpoints |
| `internal/keys/notary*` | Query orchestration, signing, cache selection, storage writes |
| `internal/keys/fetcher*` | Remote fetch, fallback notaries, retry, signature verification, SSRF checks |
| `internal/keys/resolver*` | `.well-known`, SRV, legacy SRV, host/port resolution |
| `internal/keys/storage.go` | PostgreSQL persistence for responses and individual keys (schema owned by migrations) |
| `internal/keys/keyprovider` | Pluggable signing-key backend (file, env, kms-stub); see ADR-0007 |
| `internal/keys/transparency*` | Append-only transparency log, Merkle proofs, verification |
| `internal/keys/analytics*` | Runtime key statistics and anomaly tracking |
| `internal/cluster/*` | Authenticated CRDT or Raft-backed replication |
| `internal/storage/migrations` | Embedded SQL migrations runner (see ADR-0008) |
| `internal/zero/*` | Internal infrastructure packages (config, canonical JSON, metrics, raft, logging) |

## Key Query Flow

```text
Client
  -> POST /_matrix/key/v2/query
  -> validate body, limits, server names, key IDs
  -> memory cache
  -> PostgreSQL cache
  -> resolver (.well-known -> SRV -> fallback)
  -> upstream fetch and signature verification
  -> raw-preserving perspective signature attach
  -> storage/cache update
  -> response with server_keys + failures
```

## Notary Reply Integrity (raw-preserving pipeline)

The reply pipeline preserves the origin-delivered
`/_matrix/key/v2/server` payload byte-for-byte from upstream fetch
all the way through to the `/_matrix/key/v2/query` response this
notary returns. This is what lets a downstream client verify the
origin self-signature end-to-end against our reply, not just the
perspective signature this notary adds.

Moving pieces:

- `ServerKeysResponse.Raw []byte` (`internal/keys/types.go`) carries
  the origin canonical JSON through cache, DB, cluster replication,
  and the wire response. A custom `MarshalJSON` emits `Raw`
  verbatim when populated so the query wire format keeps origin
  canonical form intact.
- `internal/keys/fetcher_direct.go` captures origin bytes into
  `Raw` immediately after self-signature verification.
- `internal/keys/notary_raw_response.go` holds
  `AttachNotarySignature(raw, notary, keyID, priv)`. It parses
  `raw` into a generic map with `UseNumber` (preserves
  `valid_until_ts` and other integers byte-exactly), attaches the
  notary's signature under `signatures[<notary>][<key_id>]`
  without reshaping any other field, and returns canonical JSON
  bytes. Presence/absence of `old_verify_keys` and any future
  Matrix schema extension survive the round-trip.
- `internal/keys/notary_query.go` prefers `AttachNotarySignature`
  when `Raw` is present; the struct-based `addNotarySignature`
  stays as a fallback for legacy rows and notary-fallback fetches
  where raw bytes are not available.
- Storage: `server_key_responses.raw_response BYTEA` (migration
  `0003_raw_server_response.sql`). Legacy NULL rows transparently
  fall back to the struct-based path until the next successful
  fetch backfills the column.
- Cluster replication moves `raw_response` between peers
  verbatim, so any node that has the raw bytes can serve a
  signature-verifiable notary reply. There is no leader-only hot
  spot for federation reads.

Regression guard: unit tests under `notary_raw_response_test.go`
cover origin signatures surviving `old_verify_keys` being
omitted / empty / populated, notary-signature validity, and
bad-input rejection. End-to-end proof lives in the 3-node smoke
test against live Synapse A/B in `test_servers/` (see the
`federation_edge_test` harness under `mfos.sdk`).

## Cache and Persistence

- Memory cache: short-lived `ServerKeysResponse` objects with defensive cloning on every lookup.
- PostgreSQL: full responses + per-key rows, schema managed via `internal/storage/migrations`.
- Expired entries cleaned by a cleanup goroutine (config `keys.cleanup_hours`).
- Stale fallback responses are returned only under explicit logic in the notary path (cache kept alive after a failed fetch when still within TTL).

## Schema Migrations

See ADR-0008.

- `internal/storage/migrations` owns all DDL.
- Embedded SQL files named `NNNN_name.sql` applied in version order.
- `schema_migrations` table tracks applied versions.
- Each migration runs in its own transaction: a failed migration rolls back without corrupting bookkeeping.
- `server.New` invokes `migrations.Apply` before any dependent component touches the database.

## Signing Key Provider

See ADR-0007.

- `internal/keys/keyprovider.Provider` interface: `LoadOrGenerate`, `PublicKey`, `Sign`, `Kind`.
- `FileProvider` (default, backward compatible): raw ed25519 bytes at 0600 under a 0700 directory.
- `EnvProvider`: base64 seed or full key from an env variable.
- `KMSStub`: documents the future external-KMS contract; returns `ErrNotImplemented`.

## Cluster Model

See ADR-0001.

- `crdt` (default): eventually consistent replication, LWW by timestamp.
- `raft` (production): quorum commit with persistent WAL + snapshots under `cluster.raft_state_dir`.

### Cluster Transport Invariants

- Transport uses shared-secret HMAC-SHA256 authentication over canonical JSON of the message fields (type, from, timestamp, payload_hex). Canonical encoding eliminates structural ambiguity from ad-hoc string concatenation.
- Wire messages are bounded in size and time-limited; 5-minute skew window.
- Replay protection: MAC signatures are cached for the skew window; reuses are rejected.
- Wildcard bind addresses require explicit advertise address.
- Replicated server responses are treated as trusted cluster data and written into cache/storage on peers only after the embedded self-signature re-verifies.

### Raft Persistence

- WAL: append-only `raft.wal` with per-record length + CRC32C header. Group-commit batcher amortizes fsync across bursts with bounded queue backpressure.
- Snapshot: `raft.snapshot` with magic prefix, metadata header, and CRC32C payload. Written via temp-file + rename for crash safety.
- InstallSnapshot RPC catches up followers whose `nextIndex` sits below the leader's compaction boundary.
- `SetSnapshotProvider` / `SetSnapshotInstaller` callbacks keep the state-machine payload opaque to the Raft layer.

## Security Notes

- Upstream key material is verified cryptographically before first local acceptance.
- Request decoding enforces size and JSON depth limits.
- SSRF checks reject resolved private IPs when enabled.
- Enterprise operational routes require token-based access.
- `cluster.shared_secret` rejects known example placeholders and enforces minimum length 32.
- Canonical JSON parser is continuously fuzzed (`FuzzJSON`, `FuzzMarshalRoundTrip`) against round-trip and idempotence invariants.

## Metrics

The metrics endpoint is `GET /_mxkeys/metrics`.
Alert definitions live in `docs/prometheus-alerts.yaml`.
Grafana dashboard assets live in `docs/grafana/`.
