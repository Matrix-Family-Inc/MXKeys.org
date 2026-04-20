Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Docs Index

## Core Set

- `federation-behavior.md` — normative public API and request/response behavior
- `architecture.md` — runtime architecture and data flow
- `deployment.md` — deployment and operational guide
- `build.md` — build, verification, and reproducibility commands
- `threat-model.md` — security assumptions, risks, and controls
- `matrix-v1.16-conformance-matrix.md` — Matrix v1.16 scope coverage

## Runbooks

- `runbook/key-rotation.md` — signing-key rotation procedure
- `runbook/cluster-disaster-recovery.md` — CRDT and Raft recovery paths
- `runbook/schema-migration.md` — PostgreSQL schema change procedure

## Supporting Material

- `adr/` — architecture decision records
- `grafana/` — dashboard assets
- `prometheus-alerts.yaml` — alert definitions
- `release-process.md` — release traceability and evidence policy
- `transparency-verification.md` — external STH verification guide
- `deployment/monitoring.md` — Prometheus + Grafana setup notes

## Usage

Read `federation-behavior.md` for the stable public contract.
Read `architecture.md` and `deployment.md` for implementation and operations.
Use `build.md` for local verification and CI-parity commands.
Consult `runbook/` for step-by-step operator procedures.

The GitHub-facing overview remains in the repository `README.md`.
