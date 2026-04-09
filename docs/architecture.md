# MXKeys Architecture

## Scope

This document describes the current runtime shape of MXKeys:

- HTTP request flow,
- cache and fetch path,
- transparency and analytics subsystems,
- authenticated cluster transport.

The normative API contract lives in `docs/federation-behavior.md`.

## Request Pipeline

Public request flow in `internal/server`:

1. `RequestIDMiddleware`
2. optional `RequestIDRequirementMiddleware`
3. `SecurityHeadersMiddleware`
4. request logging middleware
5. rate limiting middleware
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
| `internal/server` | HTTP routing, middleware, status, protected operational endpoints |
| `internal/keys/notary*` | query orchestration, signing, cache selection, storage writes |
| `internal/keys/fetcher*` | remote fetch, fallback notaries, retry, signature verification, SSRF checks |
| `internal/keys/resolver*` | `.well-known`, SRV, legacy SRV, host/port resolution |
| `internal/keys/storage.go` | PostgreSQL persistence for responses and individual keys |
| `internal/keys/transparency*` | append-only transparency log, Merkle proofs, verification |
| `internal/keys/analytics*` | runtime key statistics and anomaly tracking |
| `internal/cluster/*` | authenticated CRDT or Raft-backed replication |
| `internal/zero/*` | internal infrastructure packages (config, canonical JSON, metrics, raft, logging) |

## Key Query Flow

```text
Client
  -> POST /_matrix/key/v2/query
  -> validate body, limits, server names, key IDs
  -> memory cache
  -> PostgreSQL cache
  -> resolver (.well-known -> SRV -> fallback)
  -> upstream fetch and signature verification
  -> perspective signature
  -> storage/cache update
  -> response with server_keys + failures
```

## Cache and Persistence

- memory cache stores short-lived `ServerKeysResponse` objects,
- PostgreSQL stores full responses and per-key rows,
- expired entries are cleaned periodically,
- stale fallback responses are returned only under explicit logic in the notary path.

## Cluster Model

Cluster mode supports:

- `crdt` for eventually consistent replication,
- `raft` for replicated command flow.

Cluster invariants:

- transport uses shared-secret HMAC authentication,
- wire messages are bounded in size and time-limited,
- wildcard bind addresses require explicit advertise address,
- replicated server responses are treated as trusted cluster data and written into cache/storage on peers.

## Security Notes

- upstream key material is verified cryptographically before first local acceptance,
- request decoding enforces size and JSON depth limits,
- SSRF checks reject resolved private IPs when enabled,
- enterprise operational routes require token-based access.

## Metrics

The metrics endpoint is `GET /_mxkeys/metrics`.
Alert definitions live in `docs/prometheus-alerts.yaml`.
Grafana dashboard assets live in `docs/grafana/`.
