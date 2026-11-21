Project: MXKeys (mxkeys.org)
Company: Matrix.Family Inc. (https://matrix.family)
Owner: Matrix.Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Sat 21 Nov 2026 01:50:00 UTC
Status: Updated

# Docs

MXKeys 1.0.2 is a Matrix key notary. Read in this order. The usual
deployment is one Go process and PostgreSQL. The cluster is in the
binary and stays off until `cluster.enabled` is set.

## 1. Product

- `../README.md` — what it is, how to start it, where the source lives
- `federation-behavior.md` — HTTP API
- `../ARCHITECTURE.md` — path of one key query through the process
- `deployment.md` — install, systemd, reverse proxy, backup, cluster switch
- `build.md` — build and tests
- `../SECURITY.md` — how to report a vulnerability

## 2. Operate one node

- `runbook/backup-restore.md` — database and signing key
- `runbook/key-rotation.md` — replace the notary signing key
- `runbook/schema-migration.md` — PostgreSQL schema changes
- `runbook/release.md` — tag and publish a release

## 3. Cluster, only if enabled

- `adr/0001-cluster-consensus-modes.md` — why `crdt` and `raft` both exist
- `runbook/cluster-disaster-recovery.md` — recovery
- `runbook/raft-wal-upgrade.md` — Raft log v2 to v3

## 4. Evidence

- `threat-model.md` — what the service protects
- `matrix-v1.16-conformance-matrix.md` — status of each in-scope Matrix rule
- `matrix-v1.16-clause-map.md` — those rules mapped to code and tests
- `monitoring.md` — metrics
- `prometheus-alerts.yaml` — alert rules
- `grafana/README.md` — how to import `grafana/mxkeys-overview.json` and `grafana/mxkeys-federation.json`
- `transparency-verification.md` — check a signed tree head when the log is on

## 5. Decisions

`adr/README.md` lists the decisions about this notary.
