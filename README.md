# MXKeys

![Go](https://img.shields.io/badge/go-1.26+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Matrix](https://img.shields.io/badge/matrix-federation-purple)
![Prometheus](https://img.shields.io/badge/metrics-prometheus-orange)
![Dependencies](https://img.shields.io/badge/go%20deps-3-brightgreen)

**Matrix Federation Key Notary**

MXKeys verifies remote Matrix server keys, adds a perspective signature,
and caches verified responses in PostgreSQL and memory. The default
deployment is a single Go binary plus PostgreSQL; optional modules
(transparency log, analytics, trust policy, clustering, server-info
enrichment) are off by default and opt in via config.

## What It Does (core)

- Resolves Matrix homeservers via `.well-known`, SRV, explicit ports, and IP literals.
- Verifies Ed25519 self-signatures and server identity before accepting key material.
- Adds a perspective signature to verified responses.
- Stores validated responses in PostgreSQL and in-process cache.
- Exposes health/liveness/readiness probes and Prometheus metrics.

Optional modules (opt-in, off by default): transparency log, analytics,
trust policy, clustering (CRDT or Raft), server-info enrichment. See
[Core vs optional modules](#core-vs-optional-modules) for the full table.

## Why It Exists

- Federation trust is hard to audit when key changes happen silently.
- Upstream failures and inconsistent keys need explicit handling, not blind trust.
- Operators need a small, inspectable service rather than a large dependency stack.

## Quick Start

1. Copy the example configuration and set `database.url`.
2. Build the binary.
3. Start the service.

```bash
cp config.example.yaml config.yaml
go build -o mxkeys ./cmd/mxkeys
./mxkeys
```

Minimum config shape:

```yaml
server:
  # Set to the public server name of your deployment (must match TLS certificate).
  name: notary.example.org

database:
  url: postgres://mxkeys:password@localhost/mxkeys?sslmode=disable

keys:
  storage_path: /var/lib/mxkeys/keys

trusted_servers:
  fallback:
    - matrix.org
```

`server.name` must be configured explicitly to the public hostname of your deployment. `database.url` has no built-in default and must be configured explicitly. To gate the admin-only operational routes with a bearer token, set `security.admin_access_token`. For cluster mode, set `cluster.shared_secret` and, when binding to a wildcard address, `cluster.advertise_address`.
If `security.trust_forwarded_headers` is enabled, configure the full trusted proxy chain in `security.trusted_proxies` and ensure those proxies overwrite forwarded headers instead of passing client-supplied values through unchanged.

## Public API

```text
GET  /_matrix/key/v2/server
GET  /_matrix/key/v2/server/{keyID}
POST /_matrix/key/v2/query
GET  /_matrix/federation/v1/version

GET  /_mxkeys/health
GET  /_mxkeys/live
GET  /_mxkeys/ready
GET  /_mxkeys/status
GET  /_mxkeys/metrics
```

Query example (replace `notary.example.org` with your deployed hostname):

```bash
curl -X POST https://notary.example.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys":{"matrix.org":{}}}'
```

Admin-only operational routes for transparency inspection, analytics,
cluster status, and trust-policy checks sit behind a bearer token
(`security.admin_access_token`) and are not part of the stable
federation contract. They are local ops/debug surfaces, not a product
tier. The normative API contract lives in `docs/federation-behavior.md`.

The strongest compatibility promise applies to the Matrix key-notary endpoints. Operational probes are documented and supported, but they do not carry the same strict compatibility scope as the core federation API.

## Core vs optional modules

The default deployment ships only the Matrix key-notary endpoints, the
health/live/ready probes, and the gated `/_mxkeys/status` +
`/_mxkeys/metrics` pair. Everything else is opt-in.

| Module | Config flag | Default | Purpose |
|---|---|---|---|
| Key notary (Matrix v2) | always on | on | `/_matrix/key/v2/*`, `/_matrix/federation/v1/version` |
| Health/live/ready probes | always on | on | Kubernetes / orchestration probes |
| Status + metrics | `security.admin_access_token` (optional gate) | served, token-gated if set | `/_mxkeys/status`, `/_mxkeys/metrics` |
| Admin routes (transparency inspection, analytics, circuits, cluster/policy status) | `security.admin_access_token` | unset — routes not registered | Local ops/debug surface |
| Trust policy | `trust_policy.enabled` | `false` | Allow/deny list, TLS/notary-signature requirements |
| Transparency log | `transparency.enabled` | `false` | Append-only log with Merkle proofs |
| Cluster (CRDT or Raft) | `cluster.enabled` | `false` | Multi-node replication |
| Server-info enrichment | `server_info.enabled` | `false` | DNS/reachability/WHOIS lookup endpoint |
| At-rest key encryption | `keys.encryption.passphrase_env` | unset — plaintext 0600 | AES-256-GCM envelope for the signing key |

If none of the optional modules are enabled, the runtime surface is the
core federation API plus three probes; nothing else is reachable.

## Integration

Replace `notary.example.org` with the public hostname of your MXKeys deployment.

Synapse:

```yaml
trusted_key_servers:
  - server_name: notary.example.org
```

Dendrite:

```yaml
federation_api:
  key_perspectives:
    - server_name: notary.example.org
```

MXCore:

```yaml
federation:
  trusted_key_servers:
    - notary.example.org
```

## Verification

For a full local verification pass:

```bash
./scripts/ci-parity-preflight.sh
```

This script mirrors the PR gate workflow: unit + race + tagged integration tests, vet, gofmt, govulncheck, gosec, coverage gate, staticcheck, errcheck, a 30s-per-target fuzz pass, and the landing lint/test/build/typecheck cycle.

## Documentation

- Start here: `docs/README.md`
- Public contract: `docs/federation-behavior.md`
- Architecture: `ARCHITECTURE.md`
- Deployment: `docs/deployment.md`
- Build and verification: `docs/build.md`
- Security: `docs/threat-model.md`
- Runbooks: `docs/runbook/` (key rotation, cluster DR, schema migration)
- ADRs: `docs/adr/`

---

## License

Apache License 2.0. See `LICENSE` and `SECURITY.md`.

---

## Landing Page

The `landing/` tree is an operator-forkable marketing page for a deployed
notary. Feature-Sliced Design with Zustand for UI state, Zod for env
validation, TanStack Router/Query ready for future pages, lazy-loaded i18n
(22 locales, ~7-12 KB each), and Sentry opt-in via `VITE_SENTRY_DSN`.
Set `VITE_SITE_URL` to rebrand without file edits.

Supported languages: Arabic, Bengali, Chinese (Simplified), Dutch, English,
French, German, Hebrew, Hindi, Indonesian, Italian, Japanese, Korean,
Polish, Portuguese, Russian, Spanish, Thai, Turkish, Ukrainian, Urdu,
Vietnamese. RTL languages (Arabic, Hebrew, Urdu) have automatic layout
direction.

See `docs/adr/0009-landing-fsd-stack.md` for stack rationale.

---

## Project

MXKeys is an independent Matrix federation key-notary server maintained by
Matrix Family Inc. It works with any Matrix homeserver (Synapse, Dendrite,
Conduit, MXCore). Operators deploy their own branded notary without
coupling to any specific homeserver or ecosystem.

Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family

Copyright (c) 2026 Matrix Family Inc. All rights reserved.
