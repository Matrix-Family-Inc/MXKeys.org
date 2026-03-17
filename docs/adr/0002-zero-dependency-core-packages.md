Project: MXKeys
Company: Matrix.Family Inc. - Delaware C-Corp
Dev: Brabus
Date: Mon Mar 16 2026 UTC
Status: Created
Contact: @support:matrix.family

# ADR-0002: Zero-Dependency Core Packages

## Status

Accepted

## Context

MXKeys aims to minimize supply-chain and runtime risk while keeping deterministic behavior for critical federation trust logic.
Core utility layers (metrics, config helpers, canonical JSON, Merkle primitives, router helpers, Raft implementation) are used across security-sensitive paths.

## Decision

Maintain internal `internal/zero/*` packages for core cross-cutting functionality with minimal external dependencies.

## Consequences

Positive:

- reduced dependency attack surface,
- simpler dependency auditing and update control,
- predictable behavior for critical code paths.

Trade-offs:

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
