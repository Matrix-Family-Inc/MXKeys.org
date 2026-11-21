Project: MXKeys (mxkeys.org)
Company: Matrix.Family Inc. (https://matrix.family)
Owner: Matrix.Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# MXKeys Deployment Guide

## Scope

The usual deployment is one `mxkeys` process and PostgreSQL.
Cluster mode is in the binary and stays off unless `cluster.enabled`
is set. This service is a classic Matrix key notary. It does not host
an application platform.

This document covers:

- runtime prerequisites,
- a minimal systemd deployment,
- reverse proxy requirements,
- backup and recovery essentials,
- monitoring and basic troubleshooting,
- the optional cluster.

For build and verification commands, see `docs/build.md`. Operator
runbooks for key rotation, schema migration, backup, and cluster
recovery live under `docs/runbook/`.

## Runtime Requirements

- Go 1.26+ toolchain (build only).
- PostgreSQL 14+.
- TLS termination for public deployments.
- Explicit `database.url`.
- `security.admin_access_token` when the admin-only operational routes are exposed.
- When `cluster.enabled` is true: `cluster.shared_secret` of at least 32 characters, and `cluster.raft_state_dir` when `cluster.consensus_mode` is `raft`.

## Config Resolution

`mxkeys` accepts an explicit `-config /path/to/config.yaml` flag. When
absent, it searches `./config.yaml` then `/etc/mxkeys/config.yaml`.
Environment variables prefixed `MXKEYS_` always override file values.
`-version` prints the build identifier and exits.

## Minimal Deployment

1. Create the database:

```sql
CREATE USER mxkeys WITH PASSWORD 'replace-with-strong-password';
CREATE DATABASE mxkeys OWNER mxkeys;
```

2. Install configuration:

```bash
cp config.example.yaml /etc/mxkeys/config.yaml
```

3. Build and install the binary:

```bash
go build -trimpath -ldflags="-s -w" -o mxkeys ./cmd/mxkeys
install -m 755 mxkeys /usr/local/bin/mxkeys
```

4. Schema migrations run automatically on first start; no separate step.

## Systemd Service

```ini
[Unit]
Description=MXKeys Matrix Key Notary Server
After=network.target postgresql.service

[Service]
Type=simple
User=mxkeys
Group=mxkeys
ExecStart=/usr/local/bin/mxkeys -config /etc/mxkeys/config.yaml
WorkingDirectory=/etc/mxkeys
Restart=always
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/mxkeys

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
systemctl enable --now mxkeys
```

## Reverse Proxy

When MXKeys is behind a proxy, forward:

- `Host`
- `X-Real-IP`
- `X-Forwarded-For`
- `X-Request-ID`

Example nginx block (replace `notary.example.org` with your hostname):

```nginx
server {
    listen 443 ssl http2;
    server_name notary.example.org;

    location / {
        proxy_pass http://127.0.0.1:8448;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Request-ID $request_id;
        proxy_read_timeout 60s;
    }
}
```

If you rely on forwarded headers, configure `security.trust_forwarded_headers` and `security.trusted_proxies` explicitly. The proxy chain MUST strip/overwrite incoming forwarded headers rather than passing client-supplied values through unchanged.

## Signing Key Management

MXKeys generates an ed25519 signing key on first start under
`keys.storage_path`. The key provider is pluggable (see ADR-0007):

| Provider | Use case | Config |
|---|---|---|
| `file` (default) | Single-node, stable filesystem | `keys.storage_path` (directory enforced 0700, file 0600) |
| `env` | Ephemeral orchestrators (Kubernetes, systemd-credential) | Base64 seed/full key in env variable |
| `kms` | External KMS (stub today; pluggable contract) | Endpoint + key id |

For rotation procedure see `docs/runbook/key-rotation.md`.

## Cluster

Leave `cluster.enabled` false for one node. To replicate key updates:

- `consensus_mode: crdt` shares updates in memory. A full restart of every node drops what was not already written to PostgreSQL on each node.
- `consensus_mode: raft` commits updates to `cluster.raft_state_dir` (default `/var/lib/mxkeys/raft`). Keep that directory on a real disk. `raft_sync_on_append` defaults to true.

Every node needs the same `cluster.shared_secret`. A wildcard bind address needs `cluster.advertise_address`. Keep the cluster port off the public internet. TLS for that port is `cluster.tls`. The mode choice is ADR-0001. Recovery is `docs/runbook/cluster-disaster-recovery.md`.

## Backup and Recovery

Minimum backup set:

- PostgreSQL database.
- Contents of `keys.storage_path` (for `file` provider: `mxkeys_ed25519.key`).
- `cluster.raft_state_dir` when that node runs Raft.

Example commands:

```bash
pg_dump mxkeys > /backup/mxkeys_$(date +%Y%m%d%H%M%S).sql
install -d -m 700 /backup/mxkeys-keys
install -m 600 /var/lib/mxkeys/keys/mxkeys_ed25519.key /backup/mxkeys-keys/mxkeys_ed25519.key
```

Minimum recovery checks:

```bash
curl -fsS https://notary.example.org/_mxkeys/ready
curl -fsS https://notary.example.org/_mxkeys/status
curl -fsS https://notary.example.org/_matrix/key/v2/server | jq '.verify_keys'
```

The step-by-step procedure is `docs/runbook/backup-restore.md`.

## Monitoring

Prometheus scrape:

```yaml
scrape_configs:
  - job_name: 'mxkeys'
    static_configs:
      - targets: ['notary.example.org:8448']
    metrics_path: '/_mxkeys/metrics'
```

Alert rules live in `docs/prometheus-alerts.yaml`.
Grafana dashboards live in `docs/grafana/`.

## Troubleshooting

Health checks:

```bash
curl -fsS https://notary.example.org/_mxkeys/health
curl -fsS https://notary.example.org/_mxkeys/ready
curl -fsS https://notary.example.org/_mxkeys/status
```

Logs:

```bash
journalctl -u mxkeys -f
```

Config changes require a service restart. MXKeys does not document or guarantee in-process reload via `SIGHUP`.
