# MXKeys

![Go](https://img.shields.io/badge/go-1.22+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Matrix](https://img.shields.io/badge/matrix-federation-purple)
![Prometheus](https://img.shields.io/badge/metrics-prometheus-orange)
![Zero Dependencies](https://img.shields.io/badge/core-zero%20deps-brightgreen)

**Matrix Federation Trust Infrastructure**

Production-deployed key notary service with perspective signatures, transparency logging, anomaly detection, and distributed cluster coordination.

---

## Why MXKeys?

| Problem | Solution |
|---------|----------|
| Compromised servers publish malicious keys | Trust policies + anomaly detection |
| Key rotation is hard to validate | Transparency log + rotation tracking |
| Federation relies on few perspective servers | Distributed notary clusters |
| No audit trail for key changes | Append-only log with Merkle proofs |

---

## Features

**Core Notary**
- Full Matrix server discovery (well-known, SRV, IP literals)
- Ed25519 signature verification per Matrix spec
- Perspective signatures on all verified responses
- PostgreSQL + in-memory caching

**Trust & Security**
- Configurable deny/allow lists
- Trusted notary pinning
- Circuit breaker for failing upstreams
- Rate limiting, body size limits, concurrent fetch limits

**Transparency & Analytics**
- Append-only audit log with SHA-256 hash chaining
- Merkle tree proofs of inclusion
- Key rotation tracking and anomaly detection
- Prometheus metrics export

**Distributed Mode**
- Multi-node cluster coordination
- CRDT sync (eventually consistent) or Raft consensus (strong consistency)

---

## API

```
GET  /_matrix/key/v2/server
GET  /_matrix/key/v2/server/{keyID}
POST /_matrix/key/v2/query
GET  /_mxkeys/health | /ready | /status | /metrics
GET  /_matrix/federation/v1/version
```

**Example:**

```bash
curl -X POST https://mxkeys.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys":{"matrix.org":{}}}'
```

Response includes both origin signature and MXKeys perspective signature.

---

## Architecture

```
Client → Resolver → Fetcher → Verifier → Storage → Notary → Response
```

| Component | Description |
|-----------|-------------|
| Resolver | Matrix server discovery |
| Fetcher | Remote key retrieval + validation |
| Notary | Perspective signatures |
| Storage | PostgreSQL + memory cache |
| Analytics | Key statistics + anomaly detection |
| Transparency | Append-only audit log |
| Cluster | Multi-node coordination |

**Dependencies (3 total):**
- `github.com/lib/pq` — PostgreSQL driver
- `golang.org/x/sync` — singleflight, semaphore
- `golang.org/x/time` — rate limiter

All other functionality in `internal/zero/` with zero external dependencies.

---

## Quick Start

```bash
go build ./cmd/mxkeys
./mxkeys
```

**Config (`config.yaml`):**

```yaml
server:
  name: mxkeys.example.org
  port: 8448

database:
  url: postgres://mxkeys:mxkeys@localhost/mxkeys?sslmode=disable

keys:
  storage_path: /var/lib/mxkeys/keys
  validity_hours: 24

trusted_servers:
  fallback:
    - matrix.org
```

Environment override: `MXKEYS_SERVER_NAME`, `MXKEYS_DATABASE_URL`, etc.

---

## Integration

**MXCore:**
```yaml
federation:
  trusted_key_servers:
    - mxkeys.org
```

**Synapse:**
```yaml
trusted_key_servers:
  - server_name: mxkeys.org
```

---

## Status

Production-deployed at `mxkeys.org`. Test coverage: 183 tests.

| Test | Result |
|------|--------|
| Server keys endpoint | ed25519:mxkeys, 32-byte key |
| Federation query | matrix.org, mozilla.org, gitter.im |
| Perspective signature | Added to all responses |
| Error handling | M_BAD_JSON, M_INVALID_PARAM |
| Synapse integration | s-a.mxtest.tech, s-b.mxtest.tech |

**Verify:**
```bash
curl -s -X POST https://mxkeys.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys":{"matrix.org":{}}}' | jq '.server_keys[0].signatures | keys'
# ["matrix.org", "mxkeys.org"]
```

---

## Documentation

- `docs/architecture.md` — system design
- `docs/deployment.md` — operations runbooks
- `docs/threat-model.md` — security analysis
- `docs/federation-behavior.md` — behavior contract
- `docs/matrix-v1.16-conformance-matrix.md` — spec conformance

---

## License

Apache License 2.0. See `LICENSE` and `SECURITY.md`.

---

## Project

MXKeys is part of Matrix Family Inc. infrastructure.

Website: https://matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family

Copyright (c) 2026 Matrix Family Inc. All rights reserved.
