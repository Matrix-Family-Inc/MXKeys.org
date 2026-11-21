# MXKeys

Matrix federation key notary. Current release: `1.0.2`.

MXKeys fetches another server's signing keys, checks the signature, adds its own, and stores the result. A homeserver can then ask MXKeys instead of trusting that fetch itself. One Go process and PostgreSQL. No application platform.

## Run

```bash
cp config.example.yaml config.yaml
# set server.name and database.url
go build -o mxkeys ./cmd/mxkeys
./mxkeys
```

```yaml
server:
  name: notary.example.org

database:
  url: postgres://mxkeys:password@localhost/mxkeys?sslmode=disable

keys:
  storage_path: /var/lib/mxkeys/keys
```

`server.name` is the public hostname of this notary. `database.url` has no default.

Check a running notary:

```bash
curl https://mxkeys.org/_matrix/key/v2/server
```

## What it serves

```text
GET  /_matrix/key/v2/server
GET  /_matrix/key/v2/server/{keyID}
POST /_matrix/key/v2/query
GET  /_matrix/federation/v1/version
GET  /_mxkeys/health
GET  /_mxkeys/live
GET  /_mxkeys/ready
GET  /_mxkeys/status
GET  /_mxkeys/metrics
```

```bash
curl -X POST https://notary.example.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys":{"matrix.org":{}}}'
```

The key API is the stable contract. Probes and metrics are for the operator. The request rules are in `docs/federation-behavior.md`.

## Point a homeserver at it

Synapse:

```yaml
trusted_key_servers:
  - server_name: notary.example.org
```

Dendrite:

```yaml
federation_api:
  key_perspectives:
    - server_name: notary.example.org
```

Matrix.Family:

```yaml
federation:
  trusted_key_servers:
    - notary.example.org
```

Replace `notary.example.org` with the hostname from `server.name`.

## Cluster

Off unless `cluster.enabled` is true. Two modes, both in this tree:

- `crdt` (default when clustering is on) shares key updates between nodes.
- `raft` commits those updates to a write-ahead log under `cluster.raft_state_dir` and survives a restart.

Nodes share `cluster.shared_secret` (at least 32 characters). Do not publish the cluster port. See `docs/deployment.md` and `docs/adr/0001-cluster-consensus-modes.md`. Recovery is in `docs/runbook/cluster-disaster-recovery.md`. A v2 Raft log is upgraded with `docs/runbook/raft-wal-upgrade.md`.

## Documents

The ordered index is `docs/README.md`. Start with
`docs/federation-behavior.md` for the HTTP API, `docs/deployment.md`
to install, and `ARCHITECTURE.md` for the query path.

## Source

```bash
git clone git@github.com:Matrix-Family-Inc/MXKeys.org.git
```

https://github.com/Matrix-Family-Inc/MXKeys.org

## License

Apache License 2.0. See `LICENSE` and `SECURITY.md`.

Company: Matrix.Family Inc. (https://matrix.family)
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
