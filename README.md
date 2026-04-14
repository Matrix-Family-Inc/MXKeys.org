# MXKeys

![Go](https://img.shields.io/badge/go-1.22+-blue)
![License](https://img.shields.io/badge/license-Apache%202.0-green)
![Matrix](https://img.shields.io/badge/matrix-federation-purple)
![Prometheus](https://img.shields.io/badge/metrics-prometheus-orange)
![Dependencies](https://img.shields.io/badge/go%20deps-3-brightgreen)

**Matrix Federation Key Notary**

MXKeys verifies remote Matrix server keys, adds a perspective signature, caches verified responses in PostgreSQL and memory, and exposes a compact operational surface for monitoring and protected admin-style workflows.

## What It Does

- Resolves Matrix homeservers via `.well-known`, SRV, explicit ports, and IP literals.
- Verifies Ed25519 self-signatures and server identity before accepting key material.
- Adds a perspective signature to verified responses.
- Stores validated responses in PostgreSQL and in-process cache.
- Supports transparency logging, analytics, and authenticated cluster operation.

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
  name: mxkeys.example.org

database:
  url: postgres://mxkeys:password@localhost/mxkeys?sslmode=disable

keys:
  storage_path: /var/lib/mxkeys/keys

trusted_servers:
  fallback:
    - matrix.org
```

`database.url` has no built-in default and must be configured explicitly. For protected operational routes, set `security.enterprise_access_token`. For cluster mode, set `cluster.shared_secret` and, when binding to a wildcard address, `cluster.advertise_address`.
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

Query example:

```bash
curl -X POST https://mxkeys.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys":{"matrix.org":{}}}'
```

Protected operational routes for transparency, analytics, cluster, and policy exist separately and require an enterprise access token. The normative API contract lives in `docs/federation-behavior.md`.

The strongest compatibility promise applies to the Matrix key-notary endpoints. Operational probes are documented and supported, but they do not carry the same strict compatibility scope as the core federation API.

## Integration

Synapse:

```yaml
trusted_key_servers:
  - server_name: mxkeys.org
```

MXCore:

```yaml
federation:
  trusted_key_servers:
    - mxkeys.org
```

## Verification

For a full local verification pass:

```bash
./scripts/ci-parity-preflight.sh
```

This script mirrors the PR gate workflow, including package selection from `scripts/go-package-list.sh`, the targeted integration-with-fixtures subset, frontend checks in `landing/`, and the patched `govulncheck` toolchain used in CI.

## Documentation

- Start here: `docs/README.md`
- Public contract: `docs/federation-behavior.md`
- Architecture: `docs/architecture.md`
- Deployment: `docs/deployment.md`
- Build and verification: `docs/build.md`
- Security: `docs/threat-model.md`

---

## License

Apache License 2.0. See `LICENSE` and `SECURITY.md`.

---

## Landing Page

The landing page at [mxkeys.org](https://mxkeys.org) supports automatic language detection based on browser preferences.

Supported languages: Arabic, Bengali, Chinese (Simplified), Dutch, English, French, German, Hebrew, Hindi, Indonesian, Italian, Japanese, Korean, Polish, Portuguese, Russian, Spanish, Thai, Turkish, Ukrainian, Urdu, Vietnamese.

RTL languages (Arabic, Hebrew, Urdu) are fully supported with automatic layout direction.

---

## Project

MXKeys is part of Matrix Family Inc. infrastructure.

Website: https://matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family

Copyright (c) 2026 Matrix Family Inc. All rights reserved.
