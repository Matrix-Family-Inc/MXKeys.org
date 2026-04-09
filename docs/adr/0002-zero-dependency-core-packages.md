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

# ADR-0002: Zero-Dependency Core Packages

## Status

Accepted

## Context

MXKeys aims to minimize supply-chain and runtime risk while keeping deterministic behavior for critical federation trust logic.
Some infrastructure layers are security-sensitive and small enough to maintain in-tree: metrics, config helpers, canonical JSON, Merkle primitives, router helpers, and raft internals.

## Decision

Maintain internal `internal/zero/*` packages for core cross-cutting functionality with minimal external dependencies.

## Consequences

- reduced dependency attack surface,
- simpler dependency auditing and update control,
- predictable behavior for critical code paths.
- higher maintenance burden for internal packages,
- fewer ready-made ecosystem integrations.

## Alternatives Considered

- use large external utility frameworks for observability/config/routing,
- mixed model with selective external replacements for each `zero/*` package.

## References

- `internal/zero/metrics`
- `internal/zero/config`
- `internal/zero/canonical`
- `internal/zero/merkle`
- `internal/zero/raft`
- `internal/zero/router`
