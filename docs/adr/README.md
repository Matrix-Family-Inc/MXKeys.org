Project: MXKeys
Company: Matrix Family Inc.
Maintainer: Brabus
Contact: dev@matrix.family
Date: 2026-04-25
Status: Updated

# Architecture Decision Records (ADR)

This directory stores Architecture Decision Records for MXKeys.

## ADR Goals

- Capture technical decisions with rationale.
- Keep decision history immutable and reviewable.
- Make trade-offs explicit for future maintainers.

## File Naming

Use zero-padded sequence with short slug:

- `0001-decision-title.md`
- `0002-next-decision.md`

## ADR Lifecycle

- `Proposed`: draft under review.
- `Accepted`: approved and active.
- `Superseded`: replaced by a newer ADR.
- `Deprecated`: no longer recommended.

## Required Sections

Each ADR should include:

- Title
- Status
- Context
- Decision
- Consequences
- Alternatives considered
- References

## Current ADRs

- `0001-cluster-consensus-modes.md`
- `0002-zero-dependency-core-packages.md`
- `0003-postgresql-as-persistent-cache.md`
- `0004-strict-json-and-matrix-error-contract.md`
- `0005-core-api-stability-scope.md`
- `0006-file-header-standard.md` (ecosystem policy mapping; see `../../../ecosystem-docs/adr/ECO-0005-file-header-standard.md`)
- `0007-signing-key-provider.md`
- `0008-schema-migrations.md`
- `0009-landing-fsd-stack.md` (ecosystem frontend policy mapping; see `../../../ecosystem-docs/adr/ECO-0007-frontend-fsd-stack.md`)
- `0010-file-size-policy.md` (ecosystem policy mapping; see `../../../ecosystem-docs/adr/ECO-0006-file-size-policy.md`)

Ecosystem decisions live in `../../../ecosystem-docs/adr/`. MXKeys ADRs should
reference ecosystem ADRs instead of duplicating cross-project policy.
