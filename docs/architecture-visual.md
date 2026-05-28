Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Fri Apr 25 2026 UTC
Status: Created

# MXKeys Architecture (Visual Overview)

Visual diagrams and component summaries. For implementation details
see `ARCHITECTURE.md` in the repository root.

## Overview

MXKeys is a Matrix key notary server that verifies and caches server signing keys.

```
┌─────────────────────────────────────────────────────────────────┐
│                         MXKeys                                  │
│                                                                 │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐      │
│  │  HTTP    │   │  Rate    │   │  Request │   │  Security│      │
│  │  Server  │──▶│  Limiter │──▶│  ID MW   │──▶│  Headers │      │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘      │
│                                      │                          │
│                                      ▼                          │
│                            ┌──────────────┐                     │
│                            │   Handlers   │                     │
│                            └──────┬───────┘                     │
│                                   │                             │
│          ┌────────────────────────┼────────────────────────┐    │
│          │                        │                        │    │
│          ▼                        ▼                        ▼    │
│  ┌──────────────┐        ┌──────────────┐        ┌────────────┐ │
│  │   /health    │        │   /query     │        │  /server   │ │
│  │   /ready     │        │              │        │            │ │
│  │   /status    │        │              │        │            │ │
│  └──────────────┘        └──────┬───────┘        └────────────┘ │
│                                 │                               │
│                                 ▼                               │
│                        ┌──────────────┐                         │
│                        │    Notary    │                         │
│                        └──────┬───────┘                         │
│                               │                                 │
│            ┌──────────────────┼──────────────────┐              │
│            │                  │                  │              │
│            ▼                  ▼                  ▼              │
│    ┌──────────────┐   ┌──────────────┐   ┌──────────────┐       │
│    │ Memory Cache │   │   Storage    │   │   Fetcher    │       │
│    │  (in-proc)   │   │  (Postgres)  │   │  (remote)    │       │
│    └──────────────┘   └──────────────┘   └──────┬───────┘       │
│                                                  │              │
│                              ┌───────────────────┼──────┐       │
│                              │                   │      │       │
│                              ▼                   ▼      ▼       │
│                      ┌────────────┐      ┌────────┐ ┌────────┐  │
│                      │  Resolver  │      │Fallback│ │Circuit │  │
│                      │(well-known,│      │Notaries│ │Breaker │  │
│                      │ SRV, IP)   │      │        │ │        │  │
│                      └────────────┘      └────────┘ └────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### HTTP Layer

| Component | Purpose |
|-----------|---------|
| Server | net/http with Go 1.22+ routing |
| Rate Limiter | Per-IP token bucket (golang.org/x/time) |
| Request ID | UUID generation, X-Request-ID header |
| Security Headers | X-Content-Type-Options, X-Frame-Options |

### Notary

Core key verification service:
- Signs responses with own Ed25519 key
- Adds perspective (notary) signature
- Manages cache lifecycle

### Storage

PostgreSQL-backed persistent storage:
- Server key responses (JSON)
- Individual keys (binary)
- Automatic expiry cleanup

### Fetcher

Remote key fetching:
- Direct fetch with server discovery
- Fallback to trusted notaries
- Circuit breaker for failing hosts
- Retry with exponential backoff
- Concurrent fetch semaphore

### Resolver

Matrix server name resolution:
1. `.well-known/matrix/server`
2. SRV records (`_matrix._tcp.`)
3. Direct connection (port 8448)
4. IP literal support

## Data Flow

### Key Query

```
Client                    MXKeys                     Remote Server
   │                        │                              │
   │  POST /query           │                              │
   │  {server_keys: {...}}  │                              │
   │───────────────────────▶│                              │
   │                        │                              │
   │                        │  1. Check memory cache       │
   │                        │  2. Check DB cache           │
   │                        │                              │
   │                        │  3. If miss, resolve server  │
   │                        │─────────────────────────────▶│
   │                        │                              │
   │                        │  GET /_matrix/key/v2/server  │
   │                        │◀─────────────────────────────│
   │                        │                              │
   │                        │  4. Verify signature         │
   │                        │  5. Add notary signature     │
   │                        │  6. Cache response           │
   │                        │                              │
   │  {server_keys: [...]}  │                              │
   │◀───────────────────────│                              │
   │                        │                              │
```

### Cache Hierarchy

```
┌─────────────────────────────────────────┐
│           Memory Cache (fast)           │
│         TTL: cache_ttl_hours            │
│         Size: unlimited                 │
└─────────────────┬───────────────────────┘
                  │ miss
                  ▼
┌─────────────────────────────────────────┐
│         PostgreSQL Cache (durable)      │
│         TTL: valid_until_ts             │
│         Cleanup: cleanup_hours          │
└─────────────────┬───────────────────────┘
                  │ miss
                  ▼
┌─────────────────────────────────────────┐
│            Remote Fetch                 │
│         Timeout: fetch_timeout_s        │
│         Retries: 3 with backoff         │
└─────────────────────────────────────────┘
```

## Zero-Dependency Design

Internal packages replace external dependencies:

| Package | Replaces | Purpose |
|---------|----------|---------|
| zero/metrics | prometheus/client_golang | Prometheus text format |
| zero/log | sirupsen/logrus | slog wrapper |
| zero/config | spf13/viper | YAML + env config |
| zero/canonical | mautrix canonical JSON | Matrix canonical JSON |

Only 3 external dependencies:
- `github.com/lib/pq` — PostgreSQL driver
- `golang.org/x/sync` — Singleflight
- `golang.org/x/time` — Rate limiter

## Security Model

### Signature Verification

1. Parse JSON response
2. Remove `signatures` and `unsigned` fields
3. Convert to canonical JSON (sorted keys, no whitespace)
4. Verify Ed25519 signature
5. Check key length (32 bytes public, 64 bytes signature)

### Trust Model

```
┌─────────────────────────────────────────────────────────────┐
│                     Trust Hierarchy                         │
│                                                             │
│  ┌─────────────────────────────────────────────-────────┐   │
│  │              MXKeys (this server)                    │   │
│  │                                                      │   │
│  │  - Generates own Ed25519 keypair                     │   │
│  │  - Signs all responses                               │   │
│  │  - Verifies all upstream responses                   │   │
│  └───────────────────────────────────────────────-──────┘   │
│                           │                                 │
│                           │ queries                         │
│                           ▼                                 │
│  ┌────────────────────────────────────────────────-─────┐   │
│  │              Remote Servers                          │   │
│  │                                                      │   │
│  │  - Verified via TLS + signature                      │   │
│  │  - Self-signed server_name must match request        │   │
│  └─────────────────────────────────────────────────-────┘   │
│                           │                                 │
│                           │ fallback                        │
│                           ▼                                 │
│  ┌──────────────────────────────────────────────────-───┐   │
│  │              Fallback Notaries                       │   │
│  │                                                      │   │
│  │  - Trusted via TLS (default)                         │   │
│  │  - Optional: pinned key verification                 │   │
│  └───────────────────────────────────────────────────-──┘   │
│                                                             │
└───────────────────────────────────────────────────────-─────┘
```

## Metrics

All metrics prefixed with `mxkeys_`:

| Metric | Type | Description |
|--------|------|-------------|
| http_requests_total | Counter | HTTP requests by method/route/status |
| http_request_duration_seconds | Histogram | Request latency |
| in_flight_requests | Gauge | Current in-flight requests |
| keys_cache_hits_total | Counter | Cache hits by type |
| keys_cache_misses_total | Counter | Cache misses by type |
| keys_fetch_attempts_total | Counter | Fetch attempts by status/source |
| rate_limited_requests_total | Counter | Rate limited requests |
| request_rejections_total | Counter | Rejected requests by reason |
| go_goroutines | Gauge | Current goroutines |
| go_memstats_heap_* | Gauge | Memory statistics |
