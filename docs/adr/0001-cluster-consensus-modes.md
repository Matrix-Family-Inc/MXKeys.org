Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

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

| Property | CRDT (default) | Raft |
|----------|---------------|------|
| Consistency | Eventual (LWW by timestamp) | Strong (quorum commit) |
| Availability | All nodes accept writes during partition | Minority becomes read-only |
| Partition tolerance | Both sides converge after heal | Leader election after timeout |
| State persistence | In-memory only | Write-ahead log + snapshot on disk |
| Data on restart | Lost (requires re-sync from peers) | Preserved (WAL replay + snapshot install) |
| Log compaction | N/A | Snapshot via `Node.CompactLog`, truncates WAL prefix |
| Catch-up for lagging peers | Full re-sync via CRDT merge | `InstallSnapshot` RPC + AppendEntries |
| Production ready | Yes | Yes |
| Authentication | HMAC-SHA256 over canonical JSON | HMAC-SHA256 over canonical JSON |
| Transport encryption | None (plaintext TCP) | None (plaintext TCP) |

### Operational Implications

- **CRDT**: clock skew between nodes can cause LWW conflicts; NTP synchronization required.
- **Raft**: configure `cluster.raft_state_dir` to a local directory with 0700 permissions (e.g. `/var/lib/mxkeys/raft`). The WAL (`raft.wal`) and snapshot file (`raft.snapshot`) live there. Each record is length-prefixed and CRC32-protected; a torn tail after a crash is truncated to the last well-formed record on replay. `cluster.raft_sync_on_append=true` fsyncs every append for strict power-loss durability.
- Both modes require `cluster.shared_secret` for message authentication (>=32 chars, placeholders rejected). Transport-level encryption (TLS) is not implemented; deploy behind a secure network boundary or VPN.

## Alternatives Considered

- Raft-only cluster architecture.
- CRDT-only cluster architecture.
- External consensus dependency.

## References

- `internal/cluster/runtime.go`
- `internal/cluster/network.go`
- `internal/zero/raft/`
- `internal/config/config.go`
