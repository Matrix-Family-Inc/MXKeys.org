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

# ADR-0001: Cluster Consensus Modes

## Status

Accepted

## Context

MXKeys supports distributed notary operation where nodes share key-related state.
The cluster layer must support both low-complexity cache propagation and stronger replicated coordination for deployments that explicitly need it.

## Decision

Use CRDT synchronization as the default cluster mode, with optional Raft mode for deployments that require stronger consistency semantics.

- default mode: `crdt`
- optional mode: `raft`
- cluster mode is opt-in via `cluster.enabled`

## Consequences

- CRDT default lowers operational complexity for most deployments.
- Eventual consistency is sufficient for non-transactional cache propagation.
- Optional Raft provides stronger consistency when explicitly required.
- CRDT mode may expose temporary divergence between nodes.
- Raft mode increases operational and network coordination complexity.
- cluster transport requires explicit configuration, authentication, and monitoring.

## Alternatives Considered

- Raft-only cluster architecture.
- CRDT-only cluster architecture.
- External consensus dependency.

## References

- `internal/cluster/runtime.go`
- `internal/cluster/network.go`
- `internal/zero/raft/`
- `internal/config/config.go`
