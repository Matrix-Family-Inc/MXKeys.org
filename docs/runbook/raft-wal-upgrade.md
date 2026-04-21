Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# Runbook: Raft WAL v2 to v3 Upgrade

The Raft write-ahead log format was bumped from v2 to v3. The v3
record layout adds an HMAC-SHA256 tag alongside the existing
CRC32C, letting the node detect intentional tampering of the
on-disk file and not just bit rot.

A running v3 node refuses to open a v2 file with
`ErrWALLegacyFormat`. Operators have two paths off v2:

1. **Offline upgrade** with `mxkeys-walctl` (preserves every
   committed entry).
2. **Wipe and snapshot replay** (simpler but requires a recent
   snapshot on disk or a working peer to catch up from).

## Prerequisites

- Service stopped on the node being upgraded
  (`systemctl stop mxkeys` or equivalent).
- The cluster shared secret available in an environment variable
  (the same value configured as `cluster.shared_secret`).
- `mxkeys-walctl` binary on the host. It is shipped alongside
  `mxkeys` and `mxkeys-verify` in release tarballs.

## Offline Upgrade Procedure

```bash
export MXKEYS_CLUSTER_SECRET='<the-same-value-as-cluster.shared_secret>'

mxkeys-walctl upgrade \
  --dir /var/lib/mxkeys/raft \
  --secret-env MXKEYS_CLUSTER_SECRET
```

Steps the tool performs:

1. Reads `/var/lib/mxkeys/raft/raft.wal` as v2.
2. Re-authenticates every record under the shared secret and
   writes the v3 output to `/var/lib/mxkeys/raft/raft.wal.upgrade`.
3. `fsync` the v3 file.
4. Atomically renames the original to
   `/var/lib/mxkeys/raft/raft.wal.v2-backup` (unless
   `--backup=false`).
5. Atomically renames the v3 file to `raft.wal`.

Because each step writes to a fresh path and the only mutation
to `raft.wal` is a rename, a crash leaves either the original
v2 file or the v3 file intact; there is no state in which
`raft.wal` is half-written.

After the rename, start the service:

```bash
systemctl start mxkeys
journalctl -u mxkeys -n 50 -f
```

Look for `raft: loaded N entries from WAL` in the log. That
number must match the record count reported by the upgrade
tool.

## Wipe and Replay Procedure

When the v2 WAL is not needed (snapshot is recent, or other
cluster peers can catch us up):

```bash
systemctl stop mxkeys
rm /var/lib/mxkeys/raft/raft.wal
systemctl start mxkeys
```

On start the node:

1. Loads the most recent `raft.snapshot` from
   `/var/lib/mxkeys/raft/`.
2. Accepts `AppendEntries` from the leader for any entries
   newer than the snapshot's `LastIncludedIndex`.

If there is no snapshot on disk and the cluster is otherwise
available, the leader will eventually drive an
`InstallSnapshot` transfer to bring the node back up.

## Rollback

If the upgraded node refuses to start or the cluster sees
unexpected behaviour, revert to the backup:

```bash
systemctl stop mxkeys
mv /var/lib/mxkeys/raft/raft.wal /var/lib/mxkeys/raft/raft.wal.v3-broken
mv /var/lib/mxkeys/raft/raft.wal.v2-backup /var/lib/mxkeys/raft/raft.wal
systemctl start mxkeys
```

The node is then back on v2 semantics and must be upgraded via
the wipe-and-replay path once the issue is resolved.

## Verification

After upgrade, the v3 file's magic prefix is `MXKS_WAL_v3\x00`.
Operators can sanity-check with `head -c 12 raft.wal | xxd`:

```text
4d 58 4b 53 5f 57 41 4c 5f 76 33 00  MXKS_WAL_v3.
```
