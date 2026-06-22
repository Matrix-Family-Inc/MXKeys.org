Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# ADR-0001: Cluster Consensus Modes

## Status

Accepted

## Visibility

Public.

## Context

MXKeys is a single-binary key-notary. Clustering is optional and off
by default (`cluster.enabled: false`). When operators do enable it,
the cluster layer needs to support both low-complexity cache
propagation and stronger replicated coordination for deployments that
explicitly need it.

## Decision

When clustering is enabled, use CRDT synchronization as the default
mode, with Raft as an alternative for deployments that need stronger
consistency semantics.

- default mode (when cluster is enabled): `crdt`
- alternative mode: `raft`
- cluster itself is opt-in via `cluster.enabled`

## Consequences

- CRDT default lowers operational complexity for the (minority of)
  deployments that enable clustering at all.
- Eventual consistency is sufficient for non-transactional cache
  propagation.
- Raft provides stronger consistency when explicitly required.
- CRDT mode may expose temporary divergence between nodes.
- Raft mode increases operational and network coordination complexity.
- Cluster transport requires explicit configuration, authentication,
  and monitoring.

## Mode Comparison

The table below compares the two modes on the axes operators usually
care about. It is a decision aid, not a service-level agreement.

| Property | CRDT (default) | Raft |
|----------|---------------|------|
| Consistency | Eventual (LWW by timestamp) | Strong (quorum commit) |
| Availability | All nodes accept writes during partition | Minority becomes read-only |
| Partition tolerance | Both sides converge after heal | Leader election after timeout |
| State persistence | In-memory only | Write-ahead log + snapshot on disk |
| Data on restart | Lost (requires re-sync from peers) | Preserved (WAL replay + snapshot install) |
| Log compaction | N/A | Snapshot via `Node.CompactLog`, truncates WAL prefix |
| Catch-up for lagging peers | Full re-sync via CRDT merge | `InstallSnapshot` RPC + AppendEntries |

### Operational Implications

- **CRDT**: clock skew between nodes can cause LWW conflicts; NTP synchronization required.
- **Raft**: configure `cluster.raft_state_dir` on durable local storage; WAL and snapshot handling are covered by `docs/runbook/cluster-disaster-recovery.md`.
- Both modes require `cluster.shared_secret` for message authentication (>=32 chars, placeholders rejected). TLS 1.3 is available for cluster transport and should be enabled for production clusters; deployments that leave TLS disabled must keep cluster ports on a private network or VPN.

## Alternatives Considered

- Raft-only cluster architecture. Rejected because it forces quorum
  operations on deployments that only need cache propagation.
- CRDT-only cluster architecture. Rejected because some deployments
  need stronger replicated coordination.
- External consensus dependency. Rejected to keep the cluster core
  self-contained and aligned with ADR-0002.

## References

- `internal/cluster/runtime.go` - cluster-mode selection and lifecycle wiring.
- `internal/zero/raft/` - in-tree Raft implementation used by strong
  consistency mode.
- `internal/config/config.go` - cluster configuration surface.
