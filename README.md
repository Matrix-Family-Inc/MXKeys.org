# MXKeys

![Go](https://img.shields.io/badge/go-1.22+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Matrix](https://img.shields.io/badge/matrix-federation-purple)
![Prometheus](https://img.shields.io/badge/metrics-prometheus-orange)
![Zero Dependencies](https://img.shields.io/badge/core-zero%20deps-brightgreen)

**Matrix Federation Trust Infrastructure**

Comprehensive federation key trust layer: key verification, transparency logging, anomaly detection, and distributed cluster coordination.

Production-deployed core notary service. Security-hardened. Tested under load and failure scenarios.

**Core capabilities:**
- Federation key notary with perspective signatures
- Append-only transparency log with Merkle proofs
- Trust policies (deny/allow lists, signature requirements)
- Federation analytics and anomaly detection
- Distributed notary cluster modes (CRDT sync + optional Raft consensus)

---

## Why MXKeys?

Matrix federation relies on server signing keys. Current infrastructure has limitations:

| Problem | MXKeys Solution |
|---------|-----------------|
| Compromised servers publish malicious keys | Trust policies + anomaly detection |
| Key rotation is hard to validate | Transparency log + rotation tracking |
| Federation relies on few perspective servers | Distributed notary clusters |
| No audit trail for key changes | Append-only log with Merkle proofs |
| Verification logic duplicated across homeservers | Centralized trust infrastructure |

**MXKeys provides a federation key trust layer, not just a notary endpoint.**

---

## What MXKeys Does

**Notary Layer:**
- Fetches keys using full Matrix server discovery
- Verifies signatures using canonical JSON
- Caches in memory + PostgreSQL
- Adds perspective signatures
- Exposes standard Matrix key APIs

**Trust Layer:**
- Configurable trust policies (deny/allow, require signatures)
- Trusted notary pinning with public key verification
- Circuit breaker for failing upstreams
- Private IP blocking

**Transparency Layer:**
- Append-only audit log with SHA-256 hash chaining
- Merkle tree for cryptographic proofs of inclusion
- Key rotation tracking and history
- Automatic anomaly detection

**Analytics Layer:**
- Per-server key statistics
- Rotation frequency analysis
- Anomaly detection (rapid rotation, short validity, missing signatures)
- Prometheus metrics export

**Distributed Layer:**
- Multi-node cluster coordination
- CRDT-based state synchronization (eventually consistent)
- Raft consensus (strong consistency)
- Automatic key broadcasting across nodes

---

## Features

**Matrix Server Discovery**
- `.well-known` delegation
- SRV records (`_matrix-fed._tcp`, `_matrix._tcp`)
- Explicit federation ports
- IP literal support

**Signature Verification**
- Canonical JSON per Matrix spec
- Ed25519 signature validation
- Server name validation
- Key length validation

**Caching**
- In-memory cache with TTL
- PostgreSQL persistent storage
- Negative cache for failed lookups
- Automatic cache cleanup

**DoS Protection**
- Per-IP rate limiting
- Request body limits
- Max servers per query limit
- Concurrent fetch limits (semaphore)
- Request deduplication (singleflight)

**Observability**
- Prometheus-compatible metrics
- Structured logging (slog)
- Health/liveness/readiness probes
- Detailed status endpoint

---

## Supported API

```
GET  /_matrix/key/v2/server
GET  /_matrix/key/v2/server/{keyID}
POST /_matrix/key/v2/query
```

**Additional endpoints:**
```
GET  /_mxkeys/health
GET  /_mxkeys/live
GET  /_mxkeys/ready
GET  /_mxkeys/status
GET  /_mxkeys/metrics
GET  /_matrix/federation/v1/version
```

Matrix Server-Server API compatible.

**Example request:**

```bash
curl -X POST https://mxkeys.example.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{
    "server_keys": {
      "matrix.org": {}
    }
  }'
```

**Example response:**

```json
{
  "server_keys": [
    {
      "server_name": "matrix.org",
      "valid_until_ts": 1735689600000,
      "verify_keys": {
        "ed25519:a_ABCDEF": {
          "key": "base64-public-key"
        }
      },
      "signatures": {
        "matrix.org": {
          "ed25519:a_ABCDEF": "base64-server-signature"
        },
        "mxkeys.example.org": {
          "ed25519:mxkeys": "base64-notary-signature"
        }
      }
    }
  ]
}
```

Note: MXKeys adds its own perspective signature (`mxkeys.example.org`) to verified responses.

---

## Architecture

**Request flow:**

```
Client → Resolver → Fetcher → Verifier → Storage → Notary → Response
```

```
┌─────────────────────────────────────────────────────┐
│                      MXKeys                         │
├─────────────────────────────────────────────────────┤
│  Resolver     │ Matrix server discovery             │
│  Fetcher      │ Remote key retrieval + validation   │
│  Notary       │ Perspective signatures              │
│  Storage      │ PostgreSQL + memory cache           │
│  Analytics    │ Federation key analytics            │
│  Transparency │ Append-only audit log               │
│  Cluster      │ Multi-node coordination             │
├─────────────────────────────────────────────────────┤
│               internal/zero packages                │
│  zero/metrics   │ Prometheus exporter (no deps)     │
│  zero/config    │ YAML config + env override        │
│  zero/log       │ slog wrapper                      │
│  zero/canonical │ Matrix canonical JSON             │
│  zero/merkle    │ Merkle tree + proofs              │
│  zero/raft      │ Raft consensus protocol           │
│  zero/router    │ HTTP routing utilities            │
└─────────────────────────────────────────────────────┘
```

**External dependencies (minimal):**

| Dependency | Purpose |
|------------|---------|
| `lib/pq` | PostgreSQL driver |
| `golang.org/x/sync` | singleflight, semaphore |
| `golang.org/x/time` | rate limiter |

All other functionality (metrics, logging, config, canonical JSON, Merkle tree, Raft consensus, routing) is implemented in `internal/zero` with **zero external dependencies**.

**Components:**

| Component | Description |
|-----------|-------------|
| Resolver | Implements Matrix server discovery algorithm |
| Fetcher | Fetches keys from homeservers and perspective servers |
| Notary | Provides perspective signatures and federation endpoints |
| Storage | Caches keys in PostgreSQL with automatic cleanup |
| zero/* | Zero-dependency internal packages |

---

## Zero-Dependency Design

MXKeys minimizes external dependencies.

**Internal packages (`internal/zero/`):**

| Package | Description |
|---------|-------------|
| `zero/metrics` | Prometheus text format exporter |
| `zero/config` | YAML parser with environment variable override |
| `zero/log` | Structured logging via Go slog |
| `zero/canonical` | Matrix canonical JSON encoder |

**External dependencies:**
- `github.com/lib/pq` — PostgreSQL driver
- `golang.org/x/sync` — Singleflight (request deduplication)
- `golang.org/x/time` — Token bucket rate limiter

No frameworks. No ORMs. Minimal attack surface.

---

## Quick Start

**Build:**

```bash
go build ./cmd/mxkeys
```

**Run:**

```bash
./mxkeys
```

MXKeys will automatically create required database tables on first start.

**Example config (`config.yaml`):**

```yaml
server:
  name: mxkeys.example.org
  port: 8448
  bind_address: 127.0.0.1

database:
  url: postgres://mxkeys:mxkeys@localhost/mxkeys?sslmode=disable
  max_connections: 10

keys:
  storage_path: /var/lib/mxkeys/keys
  validity_hours: 24
  cache_ttl_hours: 1
  fetch_timeout_s: 30

logging:
  level: info
  format: text

trusted_servers:
  fallback:
    - matrix.org
```

**Environment variables:**

All config values can be overridden via `MXKEYS_*` environment variables:

```bash
export MXKEYS_SERVER_NAME=mxkeys.example.org
export MXKEYS_DATABASE_URL=postgres://...
export MXKEYS_LOGGING_LEVEL=debug
```

---

## Deployment

MXKeys does not terminate TLS directly.
It is intended to run behind a reverse proxy such as Nginx, Caddy or Envoy.

**Typical setup:**

```
Internet
    │
    ▼
Reverse proxy (TLS termination)
    │
    ▼
MXKeys (127.0.0.1:8448)
    │
    ▼
PostgreSQL
```

**Database setup:**

```sql
CREATE DATABASE mxkeys;
CREATE USER mxkeys WITH PASSWORD 'mxkeys';
GRANT ALL PRIVILEGES ON DATABASE mxkeys TO mxkeys;
```

**Systemd service:**

```ini
[Unit]
Description=MXKeys Matrix Key Notary Server
After=network.target postgresql.service

[Service]
Type=simple
User=mxkeys
ExecStart=/usr/local/bin/mxkeys
WorkingDirectory=/etc/mxkeys
Restart=always

[Install]
WantedBy=multi-user.target
```

---

## Integration

MXKeys can be configured as an additional trusted key server
used by homeservers to verify federation signing keys.

MXKeys can be deployed as a public perspective key server
or as a private trust anchor inside Matrix infrastructure.

**MXCore:**

```yaml
federation:
  trusted_key_servers:
    - mxkeys.example.org
```

**Synapse:**

```yaml
trusted_key_servers:
  - server_name: mxkeys.example.org
```

**Conduit:**

```toml
trusted_servers = ["mxkeys.example.org"]
```

---

## Security

MXKeys performs several verification steps:

- **Canonical JSON verification** — signatures validated per Matrix spec
- **Perspective signature** — MXKeys adds its own signature to verified responses
- **Server name validation** — response must match requested server
- **Key length validation** — Ed25519 keys must be 32 bytes
- **Signature length validation** — signatures must be 64 bytes
- **Response sanity checks** — `valid_until_ts` must be in the future
- **Cache TTL enforcement** — expired keys are not served
- **Rate limiting** — per-IP token bucket
- **Body size limits** — prevents memory exhaustion
- **Concurrent fetch limits** — semaphore-based throttling

MXKeys verifies server keys independently from homeservers.
MXKeys never trusts upstream responses without cryptographic verification.

---

## Metrics

MXKeys exposes Prometheus-compatible metrics at `/_mxkeys/metrics`.

**HTTP metrics:**
- `mxkeys_http_requests_total{method,route,status}`
- `mxkeys_http_request_duration_seconds{method,route}`
- `mxkeys_in_flight_requests`

**Key operations:**
- `mxkeys_key_queries_total{status}`
- `mxkeys_key_fetches_total{status,source}`
- `mxkeys_key_fetch_duration_seconds{status,source}`
- `mxkeys_cache_hits_total{cache_type}`
- `mxkeys_cache_misses_total{cache_type}`

**Protection:**
- `mxkeys_rate_limited_requests_total`
- `mxkeys_request_rejections_total{reason}`
- `mxkeys_upstream_failures_total{reason}`

---

## Use Cases

**For Homeserver Operators:**
- Independent key verification without trusting other homeservers
- Perspective signatures for federation
- Early warning for compromised or misbehaving servers

**For Federation Administrators:**
- Trust anchor for Matrix infrastructure
- Centralized policy enforcement across federation
- Key rotation monitoring and audit

**For Security Teams:**
- Append-only audit trail for compliance
- Anomaly detection for suspicious key behavior
- Cryptographic proofs for forensic analysis

**For Research:**
- Federation topology analysis
- Key rotation patterns across the network
- Trust model research and experimentation

---

## Status

MXKeys core notary service is production-deployed.
Core Matrix key server API endpoints are stable. Test coverage: 183 tests across 19 test files. Zero external dependencies for internal advanced packages.

Release and conformance quality gate:
- Internal release readiness checklist (private)
- See `docs/matrix-v1.16-conformance-matrix.md`
- See `docs/matrix-v1.16-clause-map.md`

---

## Public Documentation

Official public docs in this repository:
- `docs/architecture.md` — system architecture and data flow
- `docs/threat-model.md` — threat model, attack vectors, and controls
- `docs/federation-behavior.md` — deterministic federation behavior contract
- `docs/deployment.md` — deployment and operations runbooks
- `docs/build.md` — build and reproducible build instructions
- `docs/matrix-v1.16-conformance-matrix.md` — conformance status matrix
- `docs/matrix-v1.16-clause-map.md` — clause-level spec mapping
- `docs/release-notes-current.md` — current release notes
- `docs/prometheus-alerts.yaml` — Prometheus alert rules
- `docs/grafana/README.md` and `docs/grafana/*.json` — Grafana dashboards
- `docs/adr/` — architecture decision records
- `docs/release-evidence/` — checksums and SBOM release evidence

---

## Advanced Features

**Federation Trust Policies**
- Server deny/allow lists with wildcards
- Require minimum notary signatures
- Maximum key age enforcement
- Block private IP ranges
- Configurable via `config.yaml`

**Key Transparency Log**
- Append-only audit log
- SHA-256 hash chaining for integrity
- Key rotation tracking
- Anomaly detection
- PostgreSQL storage with retention

**Federation Key Analytics**
- Per-server statistics
- Key rotation metrics
- Anomaly detection (rapid rotation, short validity)
- Top rotators tracking
- Prometheus metrics export

**Distributed Notary Clusters**
- Multi-node coordination
- CRDT-based state synchronization (default)
- Raft consensus mode (strong consistency)
- Eventually consistent key cache
- Automatic node discovery
- TCP-based cluster communication

**Zero-Dependency Merkle Tree**
- Custom implementation in `internal/zero/merkle`
- Cryptographic proofs of inclusion
- Transparency log integration
- Consistency proofs for auditing

**Zero-Dependency Raft Consensus**
- Custom implementation in `internal/zero/raft`
- Leader election
- Log replication
- No external dependencies

**REST API for Enterprise Features**
- `/_mxkeys/transparency/log` — query transparency log
- `/_mxkeys/transparency/verify` — verify hash chain integrity
- `/_mxkeys/transparency/stats` — transparency statistics
- `/_mxkeys/transparency/proof` — Merkle proof for entry
- `/_mxkeys/analytics/summary` — analytics summary
- `/_mxkeys/analytics/servers` — per-server stats
- `/_mxkeys/analytics/anomalies` — detected anomalies
- `/_mxkeys/analytics/rotators` — top key rotators
- `/_mxkeys/cluster/status` — cluster status
- `/_mxkeys/cluster/nodes` — cluster nodes
- `/_mxkeys/policy/status` — trust policy status
- `/_mxkeys/policy/check` — check server against policy

**Grafana Dashboard Templates**
- Overview dashboard (`docs/grafana/mxkeys-overview.json`)
- Federation health dashboard (`docs/grafana/mxkeys-federation.json`)
- Ready-to-import JSON files

---

## License

Apache License 2.0

MXKeys is released under Apache License 2.0.

Commercial use, redistribution and modification are permitted.
Enterprise features and managed deployments may be provided separately.

See `LICENSE` for full text and `SECURITY.md` for vulnerability reporting policy.

---

## Project

MXKeys is part of the Matrix infrastructure ecosystem developed by Matrix.Family.

Copyright (c) 2026 Matrix.Family Inc. - Delaware C-Corp
