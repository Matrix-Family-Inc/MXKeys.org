Project: MXKeys (mxkeys.org)
Company: Matrix.Family Inc. (https://matrix.family)
Owner: Matrix.Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# MXKeys Architecture

## Scope

This document describes the current runtime shape of MXKeys:

- HTTP request flow,
- cache and fetch path,
- transparency and analytics subsystems,
- optional cluster transport (CRDT or Raft),
- signing-key provider abstraction,
- schema migration runner.

The usual process is one node and PostgreSQL. Cluster code starts only when `cluster.enabled` is true.

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

Admin-only operational routes:

- `/_mxkeys/transparency/*`
- `/_mxkeys/analytics/*`
- `/_mxkeys/cluster/*`
- `/_mxkeys/policy/*`

These routes are registered only when the underlying subsystem is
enabled and `security.admin_access_token` is configured. They are local
ops/debug surfaces gated by a bearer token, not a product tier.

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
| `internal/cluster/*` | Optional CRDT or Raft replication of verified key updates |
| `internal/storage/migrations` | Embedded SQL migrations runner (see ADR-0008) |
| `internal/zero/*` | Internal infrastructure packages (config, canonical JSON, metrics, logging, Raft) |

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
`/_matrix/key/v2/server` object from the upstream fetch through to
the `/_matrix/key/v2/query` response this notary returns. The reply
is canonical JSON with this notary's signature added. Origin fields,
including `old_verify_keys`, are not dropped, so a downstream client
can verify the origin self-signature on the reply.

Moving pieces:

- `ServerKeysResponse.Raw []byte` (`internal/keys/types.go`) carries
  the origin canonical JSON through cache, the database, and the
  wire response. A custom `MarshalJSON` emits `Raw`
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
- `internal/keys/notary_query.go` `countersign` calls
  `AttachNotarySignature` on `Raw` and refuses a document that has
  no raw bytes. There is no struct-remarshal fallback.
- Storage: `server_key_responses.raw_response BYTEA` (migration
  `0003_raw_server_response.sql`). A row with NULL `raw_response`
  is not served. The next successful fetch stores the signed bytes.

Regression guard: unit tests under `notary_raw_response_test.go`
cover origin signatures surviving `old_verify_keys` being
omitted / empty / populated, notary-signature validity, and
bad-input rejection. Production `mxkeys.org` runs as a single node.

## Server Info Enrichment (`/_mxkeys/server-info`)

Optional operator-facing endpoint that exposes the same Matrix
discovery work the notary performs internally, plus optional
WHOIS (TCP 43). Disabled by default; WHOIS is a separate
enable switch so operators whose egress blocks TCP 43 still ship
the same binary with DNS + reachability only.

Pipeline (orchestrated in `internal/server/serverinfo_service.go`):

1. Parse `server_name`, apply `ValidateServerName` and the
   same rate-limit bucket as `/_matrix/key/v2/query` so an
   anonymous client cannot weaponise the endpoint as a
   reconnaissance proxy.
2. Check the PostgreSQL-backed cache
   (`server_info_cache` from migration 0004). A fresh row
   short-circuits the fan-out.
3. Fan out under the configured `request_timeout` budget:
   - DNS + well-known + SRV + TCP/TLS probe
     (`serverinfo_reachability.go`).
   - WHOIS (`serverinfo_whois.go`), when `whois_enabled`.
4. Aggregate. Every sub-task reports its own short error
   string into the top-level `errors` map when it times out or
   fails; the rest of the response still flows back.
5. Upsert the combined response into the cache so the next
   query within TTL is served from a single SELECT.

Wire contract lives in `internal/server/serverinfo_types.go`.
Frontend counterpart is `landing/src/features/server-info/` and
the panel renders under the notary lookup result.

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

## Cluster

See ADR-0001. `server.New` calls `cluster.NewCluster` only when `cluster.enabled` is true.

- `crdt`: eventually consistent key updates, last-write-wins by timestamp.
- `raft`: quorum commit. The write-ahead log and snapshot live under `cluster.raft_state_dir`.

Cluster messages use HMAC-SHA256 over canonical JSON, with a shared secret of at least 32 characters. A replicated server response is applied only after its origin self-signature checks again. `GET /_mxkeys/cluster/status` and `GET /_mxkeys/cluster/nodes` exist when the cluster is on and `security.admin_access_token` is set.

## Security Notes

- Upstream key material is verified cryptographically before first local acceptance.
- Request decoding enforces size and JSON depth limits.
- SSRF checks reject resolved private IPs when enabled.
- Admin-only operational routes require bearer-token access via `security.admin_access_token`.
- Canonical JSON parser is continuously fuzzed (`FuzzJSON`, `FuzzMarshalRoundTrip`) against round-trip and idempotence invariants.

## Metrics

The metrics endpoint is `GET /_mxkeys/metrics`.
Alert definitions live in `docs/prometheus-alerts.yaml`.
Grafana dashboard assets live in `docs/grafana/`.
