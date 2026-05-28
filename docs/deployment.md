Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# MXKeys Deployment Guide

## Scope

This document covers:

- runtime prerequisites,
- a minimal systemd deployment,
- reverse proxy requirements,
- backup and recovery essentials,
- monitoring and basic troubleshooting,
- cluster transport hardening notes.

For build and verification commands, see `docs/build.md`. Operator
runbooks for specific tasks (key rotation, schema migration, cluster DR)
live under `docs/runbook/`.

## Runtime Requirements

- Go 1.26+ toolchain (build only).
- PostgreSQL 14+.
- TLS termination for public deployments.
- Explicit `database.url`.
- `security.admin_access_token` when the admin-only operational routes are exposed.
- `cluster.shared_secret` when cluster mode is enabled: minimum 32 characters, placeholder values rejected.
- For cluster mode with Raft consensus, `cluster.raft_state_dir` (default `/var/lib/mxkeys/raft`) on a durable filesystem.

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

## Scaling

MXKeys supports multiple deployment patterns:

- Single-node with PostgreSQL-backed cache.
- Multiple stateless HTTP instances sharing PostgreSQL (CRDT cluster optional for cache-warming coordination).
- Authenticated cluster mode using CRDT (eventually consistent) or Raft (strong consistency with persistent WAL).

Clustered deployments require:

- Explicit `cluster.advertise_address` when binding to wildcard addresses.
- A shared `cluster.shared_secret` of 32+ random characters.
- Network-level protection for cluster ports (see "Cluster Transport Hardening" below).

Raft-specific requirements:

- `cluster.raft_state_dir` must live on a durable local filesystem. Do not point this at a tmpfs; a crash loses committed entries otherwise.
- `cluster.raft_sync_on_append=true` (default) enforces fsync-per-batch for power-loss durability. Disabling trades durability for throughput on battery-backed hosts.

## Cluster Transport Hardening

Cluster-to-cluster traffic is authenticated with HMAC-SHA256 over
canonical JSON of the message fields; replay protection caches MAC
signatures for the 5-minute skew window.

Transport-level encryption is **not** built in. Required deployment patterns:

- Dedicated private VLAN or VPN between cluster nodes (strong recommendation).
- WireGuard / Tailscale overlay for multi-datacenter clusters.
- `iptables`/`nftables` rules restricting the cluster port to peer IPs.

Do not expose cluster ports to the public internet.

## Backup and Recovery

Minimum backup set:

- PostgreSQL database.
- Contents of `keys.storage_path` (for `file` provider: `mxkeys_ed25519.key`).
- `cluster.raft_state_dir` contents for Raft cluster members (WAL + snapshot).

Example commands:

```bash
pg_dump mxkeys > /backup/mxkeys_$(date +%Y%m%d%H%M%S).sql
install -d -m 700 /backup/mxkeys-keys
install -m 600 /var/lib/mxkeys/keys/mxkeys_ed25519.key /backup/mxkeys-keys/mxkeys_ed25519.key
tar -cf /backup/mxkeys-raft_$(date +%Y%m%d%H%M%S).tar -C /var/lib/mxkeys raft
```

Minimum recovery checks:

```bash
curl -fsS https://notary.example.org/_mxkeys/ready
curl -fsS https://notary.example.org/_mxkeys/status
curl -fsS https://notary.example.org/_matrix/key/v2/server | jq '.verify_keys'
```

For cluster-specific disaster-recovery see `docs/runbook/cluster-disaster-recovery.md`.

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
