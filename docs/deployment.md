# MXKeys Deployment Guide

## Scope

This document covers:

- runtime prerequisites,
- a minimal systemd deployment,
- reverse proxy requirements,
- backup and recovery essentials,
- monitoring and basic troubleshooting.

For build and verification commands, see `docs/build.md`.

## Runtime Requirements

- PostgreSQL 14+
- TLS termination for public deployments
- explicit `database.url` in configuration
- `security.enterprise_access_token` when protected operational routes are enabled
- `cluster.shared_secret` when cluster mode is enabled

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

## Systemd Service

```ini
[Unit]
Description=MXKeys Matrix Key Notary Server
After=network.target postgresql.service

[Service]
Type=simple
User=mxkeys
Group=mxkeys
ExecStart=/usr/local/bin/mxkeys
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

Example nginx block:

```nginx
server {
    listen 443 ssl http2;
    server_name mxkeys.example.org;

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

If you rely on forwarded headers, configure `security.trust_forwarded_headers` and `security.trusted_proxies` explicitly.

## Scaling

MXKeys supports multiple deployment patterns:

- single-node with PostgreSQL-backed cache,
- multiple stateless HTTP instances sharing PostgreSQL,
- authenticated cluster mode using CRDT or Raft.

Clustered deployments require:

- explicit `advertise_address` when binding to wildcard addresses,
- a shared `cluster.shared_secret`,
- network-level protection for cluster ports.

## Backup and Recovery

Minimum backup set:

- PostgreSQL database,
- `/var/lib/mxkeys/keys/mxkeys_ed25519.key`.

Example commands:

```bash
pg_dump mxkeys > /backup/mxkeys_$(date +%Y%m%d%H%M%S).sql
install -d -m 700 /backup/mxkeys-keys
install -m 600 /var/lib/mxkeys/keys/mxkeys_ed25519.key /backup/mxkeys-keys/mxkeys_ed25519.key
```

Minimum recovery checks:

```bash
curl -fsS https://mxkeys.example.org/_mxkeys/ready
curl -fsS https://mxkeys.example.org/_mxkeys/status
curl -fsS https://mxkeys.example.org/_matrix/key/v2/server | jq '.verify_keys'
```

## Monitoring

Prometheus scrape:

```yaml
scrape_configs:
  - job_name: 'mxkeys'
    static_configs:
      - targets: ['mxkeys.example.org:8448']
    metrics_path: '/_mxkeys/metrics'
```

Alert rules live in `docs/prometheus-alerts.yaml`.
Grafana dashboards live in `docs/grafana/`.

## Troubleshooting

Health checks:

```bash
curl -fsS https://mxkeys.example.org/_mxkeys/health
curl -fsS https://mxkeys.example.org/_mxkeys/ready
curl -fsS https://mxkeys.example.org/_mxkeys/status
```

Logs:

```bash
journalctl -u mxkeys -f
```

Config changes require a service restart. MXKeys does not document or guarantee in-process reload via `SIGHUP`.
