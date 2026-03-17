Project: MXKeys
Company: Matrix.Family Inc. - Delaware C-Corp
Dev: Brabus
Date: Mon Mar 16 2026 UTC
Status: Created
Contact: @support:matrix.family

# ADR-0001: Cluster Consensus Modes

## Status

Accepted

## Context

MXKeys supports distributed notary operation where nodes share key-related state.
The system requires:

- deterministic behavior for critical flows,
- practical operability for multi-node deployments,
- resilience under partial node/network failures.

Two models exist in code:

- CRDT-based state synchronization,
- Raft-based strong-consistency consensus.

## Decision

Use CRDT synchronization as the default cluster mode, with optional Raft mode for deployments that require stronger consistency semantics.

Decision details:

- default mode: `crdt`,
- alternative mode: `raft`,
- cluster feature is opt-in via configuration (`cluster.enabled`).

## Consequences

Positive:

- CRDT default lowers operational complexity for most deployments.
- Eventual consistency is sufficient for non-transactional cache propagation.
- Optional Raft provides stronger consistency when explicitly required.

Trade-offs:

- CRDT mode may expose temporary divergence between nodes.
- Raft mode increases operational and network coordination complexity.

Operational implications:

- mode selection must be explicit in production configuration,
- observability must include cluster sync/health metrics,
- incident procedures should account for mode-specific failure behavior.

## Alternatives Considered

- Raft-only cluster architecture.
- CRDT-only cluster architecture.
- External consensus dependency.

## References

- `internal/cluster/cluster.go`
- `internal/zero/raft/raft.go`
- `internal/config/config.go`
