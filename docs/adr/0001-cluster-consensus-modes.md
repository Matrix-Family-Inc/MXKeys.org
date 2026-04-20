Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
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

## Service Level Agreement by Mode

| Property | CRDT (default) | Raft (experimental) |
|----------|---------------|-------------------|
| Consistency | Eventual (LWW by timestamp) | Strong (quorum commit) |
| Availability | All nodes accept writes during partition | Minority becomes read-only |
| Partition tolerance | Both sides converge after heal | Leader election after timeout |
| State persistence | In-memory only | In-memory only (no WAL) |
| Data on restart | Lost (requires re-sync from peers) | Lost (requires full cluster restart) |
| Production ready | Yes | No — experimental, no persistent log or snapshots |
| Authentication | HMAC-SHA256 shared secret | HMAC-SHA256 shared secret |
| Transport encryption | None (plaintext TCP) | None (plaintext TCP) |

### Operational Implications

- **CRDT**: clock skew between nodes can cause LWW conflicts; NTP synchronization required.
- **Raft**: without persistent log, a restarted node loses committed entries and must rejoin as empty follower. No snapshot/InstallSnapshot mechanism exists.
- Both modes require `cluster.shared_secret` for message authentication. Transport-level encryption (TLS) is not implemented; deploy behind a secure network boundary or VPN.

## Alternatives Considered

- Raft-only cluster architecture.
- CRDT-only cluster architecture.
- External consensus dependency.

## References

- `internal/cluster/runtime.go`
- `internal/cluster/network.go`
- `internal/zero/raft/`
- `internal/config/config.go`
