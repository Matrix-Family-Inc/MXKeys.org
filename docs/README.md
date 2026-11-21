Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Sat 29 Aug 2026 16:11:22 UTC
Status: Updated

# Docs Index

MXKeys 1.0.2 is the key notary for a classic Matrix deployment
(homeserver, clients, this service). Cluster runbooks describe the
optional cluster, which stays off in the shipping configuration.
There is no application platform in this repository.

## Core Set

- `matrix-family-standardization.md` — Matrix Family headers, branding, git SSH (port 42224)
- `federation-behavior.md` — normative public API and request/response behavior
- `../ARCHITECTURE.md` — runtime architecture and data flow (top-level)
- `deployment.md` — deployment and operational guide
- `build.md` — build, verification, and reproducibility commands
- `threat-model.md` — security assumptions, risks, and controls
- `matrix-v1.16-conformance-matrix.md` — Matrix v1.16 scope coverage
- `matrix-v1.16-clause-map.md` — detailed clause-to-code mapping

## Runbooks

- `runbook/key-rotation.md` — signing-key rotation procedure
- `runbook/cluster-disaster-recovery.md` — CRDT and Raft recovery paths
- `runbook/schema-migration.md` — PostgreSQL schema change procedure
- `runbook/backup-restore.md` — database and signing-key backup/restore
- `runbook/production-deploy.md` — production rollout procedure
- `runbook/raft-wal-upgrade.md` — WAL format upgrade procedure
- `runbook/release.md` — release cut, tagging, and artifact publication

## Supporting Material

- `adr/` — architecture decision records
- `../../ecosystem-docs/adr/ECO-0011-mxkeys-remains-a-separate-notary-product.md`
  — MXKeys remains a separate notary product; hostname `mxkeys.org` is the
  Matrix `server_name` and is not collapsed into the client brand
- `grafana/` — dashboard assets
- `prometheus-alerts.yaml` — alert definitions
- `monitoring.md` — metrics reference and Prometheus + Grafana setup notes
- `transparency-verification.md` — external STH verification guide

## Usage

Read `federation-behavior.md` for the stable public contract.
Read `../ARCHITECTURE.md` and `deployment.md` for implementation and operations.
Use `build.md` for local verification and CI-parity commands.
Consult `runbook/` for step-by-step operator procedures.

The GitHub-facing overview remains in the repository `README.md`.
