Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# Runbook: Cluster Disaster Recovery

## Scope

Recovery procedures for the two supported cluster modes:

- `crdt` (eventually consistent, LWW by timestamp)
- `raft` (quorum-committed, WAL + snapshot on disk)

For cluster-mode selection rationale see ADR-0001.

## Assumptions

- You run at least 3 nodes in Raft mode (quorum = 2). Smaller
  clusters are documented but not recommended for production.
- `cluster.raft_state_dir` (default `/var/lib/mxkeys/raft`) is on a
  durable local filesystem, not tmpfs.
- You have periodic backups of each node's raft state dir plus the
  PostgreSQL database.

## Failure Scenarios

### CRDT: single node unreachable

CRDT tolerates partial outages natively. All surviving nodes keep
serving traffic; the failed node rejoins and merges via the LWW
sync loop when it returns.

1. Confirm the outage scope:

   ```bash
   curl -fsS https://peer1.example.org/_mxkeys/status | jq .cluster
   ```

2. Restart the failed node. State is replayed from peers through
   the sync loop.

### CRDT: complete cluster loss

Because CRDT state is in-memory, a full cluster restart loses any
replicated-but-not-yet-re-fetched keys. The PostgreSQL cache is the
durable source of truth in this mode.

1. Start every node from the restored PostgreSQL dump.
2. Let `syncLoop` warm up; federation traffic re-populates the
   in-memory CRDT state within seconds.

### Raft: minority failure (1 of 3)

Cluster stays available; the leader continues accepting writes
from the surviving quorum.

1. On the failed node, restore the last good backup of
   `cluster.raft_state_dir`:

   ```bash
   systemctl stop mxkeys
   install -d -m 700 /var/lib/mxkeys/raft
   tar -xf /backup/mxkeys-raft_20260420T120000Z.tar -C /var/lib/mxkeys
   systemctl start mxkeys
   ```

2. The restarted node replays its WAL against the persisted
   snapshot, then catches up from the leader via AppendEntries (or
   InstallSnapshot if it lags past the leader's compaction
   boundary).

### Raft: majority failure (2 of 3)

Raft is safety-first: writes halt when quorum is lost. Recovery
requires operator intervention.

1. Identify the node with the highest durable state:

   ```bash
   for n in peer1 peer2 peer3; do
     ssh "${n}" 'sudo stat -c "%Y %s" /var/lib/mxkeys/raft/raft.wal'
   done
   ```

   Pick the node whose WAL is largest and most recently modified.

2. Promote that node to a single-member cluster by editing
   `cluster.seeds` to contain only itself, then start:

   ```bash
   # On the chosen node:
   systemctl stop mxkeys
   sed -i 's|seeds: .*|seeds: []|' /etc/mxkeys/config.yaml
   systemctl start mxkeys
   ```

3. Verify it elects itself leader and serves traffic:

   ```bash
   curl -fsS https://peer1.example.org/_mxkeys/status | jq '.cluster.state'
   ```

4. Re-seed the other members:

   ```bash
   # On each remaining node, restore raft state from the promoted node:
   systemctl stop mxkeys
   rm -rf /var/lib/mxkeys/raft
   scp -r peer1:/var/lib/mxkeys/raft /var/lib/mxkeys/raft
   chown -R mxkeys:mxkeys /var/lib/mxkeys/raft
   chmod 700 /var/lib/mxkeys/raft
   # Restore seeds to the full peer list:
   sed -i 's|seeds: \[\]|seeds: [peer1:7946, peer2:7946, peer3:7946]|' /etc/mxkeys/config.yaml
   systemctl start mxkeys
   ```

5. Each joining node receives `InstallSnapshot` from the leader
   to fast-forward past the compacted prefix, then AppendEntries
   for the tail.

### Raft: WAL corruption

`ErrWALCorrupt` on startup means the WAL tail is torn (typical on a
mid-write power loss) or a record CRC failed. Recovery behavior:

1. The node logs `Raft WAL tail corrupt; truncating to last
   well-formed record` and starts with the durable prefix. No
   operator action is required for this case.

2. If `ErrSnapshotCorrupt` also fires, restore the snapshot from a
   sibling node:

   ```bash
   systemctl stop mxkeys
   scp peer2:/var/lib/mxkeys/raft/raft.snapshot \
       /var/lib/mxkeys/raft/raft.snapshot
   chmod 600 /var/lib/mxkeys/raft/raft.snapshot
   systemctl start mxkeys
   ```

### Secret compromise

If `cluster.shared_secret` is exposed:

1. Generate a new secret (>=32 random characters):

   ```bash
   head -c 48 /dev/urandom | base64 -w0
   ```

2. Roll the secret to every node simultaneously. The cluster
   transport rejects messages MAC'd with the old secret; a rolling
   update leaves partitioned nodes during the rollout, which is
   acceptable for short windows but must not exceed
   `maxMessageSkew` (5 minutes) without coordination.

## Post-Recovery Verification

```bash
# Every node reports the same leader:
for n in peer1 peer2 peer3; do
  curl -fsS "https://${n}.example.org/_mxkeys/status" \
    | jq '{node: .cluster.node_id, state: .cluster.state, leader: .cluster.leader}'
done

# A sample federation query succeeds from every node:
for n in peer1 peer2 peer3; do
  curl -fsS -X POST "https://${n}.example.org/_matrix/key/v2/query" \
    -H 'Content-Type: application/json' \
    -d '{"server_keys":{"matrix.org":{}}}' | jq '.server_keys | length'
done
```

## Follow-ups (not yet runbooked)

- Automated disaster-recovery drills.
- Operator tooling for atomic `shared_secret` rotation across all
  cluster peers.
