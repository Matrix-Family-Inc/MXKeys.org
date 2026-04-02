Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Maintainer: Brabus
Role: Lead Architect
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon Mar 16 2026 UTC
Status: Created

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
