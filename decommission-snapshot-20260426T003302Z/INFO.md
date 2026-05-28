<!--
Project: MXKeys
Company: Matrix Family Inc.
Maintainer: Brabus
Date: 2026-04-26 00:34:02 UTC
Status: Created
Contact: dev@matrix.family
-->

# MXKeys raft testbed cluster decommission snapshot

This directory captures the operational state of the three raft
smoke-test members `mxkeys-node-a`, `mxkeys-node-b`, `mxkeys-node-c`
immediately before they were stopped and removed from `bra`.

The standalone notary `mxkeys-node-main` (port `:8449`, server name
`mxkeys.org.mxtest.tech`) is unaffected by this decommission and
continues to serve the public testbed surface.

## Snapshot context

| Field            | Value                                          |
| ---------------- | ---------------------------------------------- |
| Captured at      | 2026-04-26 00:34:02 UTC                  |
| Host             | bra                                          |
| Snapshot stamp   | 20260426T003302Z                                         |
| Cluster binary   | `/opt/matrix-family/test_servers/mxkeys-cluster/bin/mxkeys` |
| Binary sha256    | `02969b6b151926f06047bb11edccc219e66a1d85f5d467d4d57606b729897199`                                  |

## Per-node uptime

| Node          | Started (local TZ)              | Backend port |
| ------------- | ------------------------------- | ------------ |
| mxkeys-node-a | Sat 2026-04-25 18:16:36 MSK                        | 127.0.0.1:8450 |
| mxkeys-node-b | Sat 2026-04-25 18:16:36 MSK                        | 127.0.0.1:8451 |
| mxkeys-node-c | Sat 2026-04-25 18:16:36 MSK                        | 127.0.0.1:8452 |

All three nodes formed a single raft group with shared secret stored
at `/opt/matrix-family/test_servers/mxkeys-cluster/SECRET`
(captured here as `SECRET-cluster-shared-secret.txt`, mode 0600).

## Captured artefacts (per node)

- `config.yaml` - exact runtime config the unit was started with.
- `unit.service` - copy of the systemd unit file.
- `systemctl-show.txt` - full `systemctl show` snapshot.
- `journal.log` - `journalctl -u <unit>` over the entire uptime.
- `state.tar.zst` - workspace state directory (raft log, snapshots,
  signing keys storage).
- `pgdump.pgc` - `pg_dump -Fc` of the per-node PostgreSQL database
  (`mxkeys_node_a`, `mxkeys_node_b`, `mxkeys_node_c`).
- `key-v2-server.json` / `key-v2-server.http` - last public
  `/_matrix/key/v2/server` response while running.
- `federation-version.json` / `federation-version.http` - last
  `/_matrix/federation/v1/version` response while running.

## Restore procedure

If the cluster ever has to come back on this or another host:

1. Recreate the three PostgreSQL databases owned by role `mxkeys`:
   `mxkeys_node_a`, `mxkeys_node_b`, `mxkeys_node_c`.
2. `pg_restore --clean --if-exists -d mxkeys_node_X node-X/pgdump.pgc`
   for each node.
3. Recreate workspace directories
   `/opt/matrix-family/test_servers/mxkeys-cluster/node-X/` and
   extract `state.tar.zst` into them.
4. Drop `unit.service` into `/etc/systemd/system/`,
   `daemon-reload`, `enable --now`.
5. Verify the raft group reconverges with quorum (`/_mxkeys/cluster`
   admin endpoint or follower logs).

## Rationale

The raft smoke-test cluster has served its purpose. Going forward the
testbed slots `node-a/b/c.mxtest.tech` are repurposed as independent
MXCore lab nodes (asymmetric scenarios: mixed versions, canary
upgrades, alternate storage backends). MXKeys public testbed surface
remains on `mxkeys.org.mxtest.tech` via the standalone notary.
