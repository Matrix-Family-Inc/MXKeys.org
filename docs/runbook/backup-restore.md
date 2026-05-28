Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Runbook: Backup and Restore

Procedure for backing up and restoring a single MXKeys node.

## Backup Contents

A backup tarball contains:

- `database.sql` : `pg_dump --clean --if-exists` of the notary
  database (server-key cache, transparency log, migrations table).
- `keys/` : signing-key directory as on disk, including the
  `MXKENC01` envelope when at-rest encryption is on.
- `raft/` : Raft state directory (WAL and snapshot) for clusters
  running in Raft mode.
- `MANIFEST.txt` : backup timestamp, binary version, sha256 for
  every file in the archive.

The tarball is deterministic (`tar --sort=name --mtime=@0`), so
two runs against the same on-disk state produce identical bytes.

## Backup Procedure

### Prerequisites

- `pg_dump` on PATH with read access to the notary database.
- Write access to the output directory (default
  `/var/backups/mxkeys`).
- Free disk roughly `keys_dir + raft_dir + db_size * 1.5`.

### Command

    scripts/mxkeys-backup.sh \
      --database-url postgres://mxkeys:password@localhost/mxkeys \
      --keys-dir /var/lib/mxkeys/keys \
      --raft-dir /var/lib/mxkeys/raft \
      --output-dir /var/backups/mxkeys

Steps the script performs:

1. `pg_dump` with `--clean --if-exists` (idempotent restore).
2. `cp -a` the keys directory (permissions preserved).
3. `cp -a` the Raft state directory when present.
4. Write `MANIFEST.txt` with per-file sha256.
5. Write `mxkeys-backup-<UTC>.tar.gz` at 0600 using atomic rename.

The service does not need to be stopped. `pg_dump` runs in a
read-only transaction. The key and Raft directories are
append-only from the service's view, so a live copy captures a
point-in-time snapshot; a file that races with a concurrent write
is caught on the next cycle.

### Scheduling

Example systemd timer:

    [Unit]
    Description=MXKeys daily backup

    [Timer]
    OnCalendar=daily
    Persistent=true
    RandomizedDelaySec=10m

    [Install]
    WantedBy=timers.target

Retention is out of scope for the script. Use `find ... -mtime +N
-delete` or the object-storage lifecycle policy.

### Off-site Copy

Ship each tarball to at least one off-host location (object
storage with encryption at rest, tape, peer replica). The
MANIFEST sha256 list verifies integrity without decompressing.

## Restore Procedure

Restore is destructive. Stop the service first, otherwise the
running process keeps serving with the old keys and the `psql`
replay may deadlock or corrupt state.

### Prerequisites

- Service stopped (`systemctl stop mxkeys` or the orchestrator
  equivalent).
- `psql` on PATH with create/drop rights on the database.
- Trusted backup tarball.

### Command

    scripts/mxkeys-restore.sh \
      --input /var/backups/mxkeys/mxkeys-backup-20260420T120000Z.tar.gz \
      --database-url postgres://mxkeys:password@localhost/mxkeys \
      --keys-dir /var/lib/mxkeys/keys \
      --raft-dir /var/lib/mxkeys/raft

Steps the script performs:

1. Extract the tarball to a temp directory.
2. Print `MANIFEST.txt` so the operator can confirm source version.
3. `psql --set=ON_ERROR_STOP=1` replay of `database.sql`.
4. Replace `keys_dir` atomically (`rm -rf` + `cp -a`).
5. Replace `raft_dir` atomically.
6. Restore 0600 / 0700 permissions on the copied trees.

After the script exits, start the service and verify:

- `/_mxkeys/readyz` returns 200 within the first readiness window.
- Startup log shows `Schema migrations applied count=0`. A
  non-zero count during restore indicates schema drift and should
  be investigated.
- Startup log shows `Signing key at-rest encryption enabled` (or
  `... disabled`) matching the operator's configuration.

## Validation

A restore is considered successful only when all three hold:

- `/_mxkeys/readyz` returns 200.
- A self-round-trip succeeds: fetch `/_matrix/key/v2/server` and
  verify the signature offline with `mxkeys-verify`.
- `mxkeys-verify --url https://<host>:8448` reports a trust level
  matching the configured policy (typically `origin_trust`).

If any check fails, stop the service and do not restart until the
cause is understood. A half-restored node that keeps signing is
worse than a longer outage.

## Disaster Recovery

For complete loss (all nodes gone), restore a single node from
the most recent off-site tarball. The restored node is the new
source of truth; rebuild other nodes by joining the standard
cluster flow (see `docs/runbook/cluster-disaster-recovery.md`).

If the signing key is unrecoverable (all backups lost), follow
`docs/runbook/key-rotation.md`. Relying homeservers re-fetch the
new key through the normal Matrix federation path.
