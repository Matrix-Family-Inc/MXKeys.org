Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Fri Apr 24 2026 UTC
Status: Updated

# ADR-0002: Zero-Dependency Core Packages

## Status

Accepted

## Visibility

Public.

## Context

MXKeys aims to minimize supply-chain and runtime risk while keeping deterministic behavior for critical federation trust logic.
Some infrastructure layers are security-sensitive and small enough to maintain in-tree: metrics, config helpers, canonical JSON, Merkle primitives, and raft internals. The module targets Go 1.26. Routing relies on stdlib `http.ServeMux` method/path patterns introduced in Go 1.22; no custom router is required.

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

- `internal/zero/canonical` - canonical JSON implementation for signed
  federation payloads.
- `internal/zero/config` - small config helpers without a framework dependency.
- `internal/zero/log` - structured logging wrapper.
- `internal/zero/merkle` - Merkle primitives for transparency proofs.
- `internal/zero/metrics` - in-tree metrics primitives.
- `internal/zero/raft` - in-tree consensus implementation.

## Alternatives

None recorded at authoring time. Any future revision that modifies this decision must list the rejected options explicitly.
