# MXKeys Deployment Guide

## Prerequisites

- PostgreSQL 14+
- Go 1.22+ (for building)
- TLS certificate (for production)

## Quick Start

### 1. Database Setup

```sql
CREATE USER mxkeys WITH PASSWORD 'your-secure-password';
CREATE DATABASE mxkeys OWNER mxkeys;

-- Schema is created automatically on first run
```

### 2. Build

```bash
git clone https://github.com/matrixfamily/MXKeys.org.git
cd MXKeys.org

go build -trimpath -ldflags="-s -w" -o mxkeys ./cmd/mxkeys
```

### 3. Configure

```bash
cp config.example.yaml /etc/mxkeys/config.yaml
# Edit config.yaml with your settings
```

### 4. Run

```bash
./mxkeys
```

## Production Deployment

### Systemd Service

```ini
# /etc/systemd/system/mxkeys.service
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

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/mxkeys

[Install]
WantedBy=multi-user.target
```

Enable:
```bash
systemctl enable --now mxkeys
```

### Nginx Reverse Proxy

```nginx
server {
    listen 443 ssl http2;
    server_name mxkeys.example.org;

    ssl_certificate /etc/letsencrypt/live/mxkeys.example.org/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mxkeys.example.org/privkey.pem;

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

## Docker Deployment

### Build Image

```bash
docker build -t mxkeys:latest .
```

### Run

```bash
docker run -d \
  --name mxkeys \
  -p 8448:8448 \
  -v /etc/mxkeys:/etc/mxkeys:ro \
  -v /var/lib/mxkeys:/var/lib/mxkeys \
  -e MXKEYS_DATABASE_URL="postgres://mxkeys:pass@host.docker.internal/mxkeys?sslmode=disable" \
  mxkeys:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  mxkeys:
    build: .
    ports:
      - "8448:8448"
    volumes:
      - ./config.yaml:/etc/mxkeys/config.yaml:ro
      - mxkeys-keys:/var/lib/mxkeys/keys
    environment:
      - MXKEYS_DATABASE_URL=postgres://mxkeys:mxkeys@db/mxkeys?sslmode=disable
    depends_on:
      - db
    restart: unless-stopped

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: mxkeys
      POSTGRES_PASSWORD: mxkeys
      POSTGRES_DB: mxkeys
    volumes:
      - mxkeys-db:/var/lib/postgresql/data

volumes:
  mxkeys-keys:
  mxkeys-db:
```

## Horizontal Scaling

MXKeys is stateless (keys stored in PostgreSQL).

```
                    ┌─────────────┐
                    │   Load      │
                    │   Balancer  │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
      ┌────▼────┐     ┌────▼────┐     ┌────▼────┐
      │ MXKeys  │     │ MXKeys  │     │ MXKeys  │
      │   #1    │     │   #2    │     │   #3    │
      └────┬────┘     └────┬────┘     └────┬────┘
           │               │               │
           └───────────────┼───────────────┘
                           │
                    ┌──────▼──────┐
                    │  PostgreSQL │
                    │   Primary   │
                    └─────────────┘
```

Requirements:
- Shared PostgreSQL database
- Sticky sessions not required
- Each instance generates its own signing key (or share via config)

## Backup

### Database

```bash
pg_dump mxkeys > mxkeys_backup_$(date +%Y%m%d).sql
```

### Signing Key

```bash
cp /var/lib/mxkeys/keys/mxkeys_ed25519.key /backup/
```

## Key Rotation Runbook

1. Create key backup and verify permissions:
```bash
install -d -m 700 /backup/mxkeys-keys
install -m 600 /var/lib/mxkeys/keys/mxkeys_ed25519.key /backup/mxkeys-keys/mxkeys_ed25519.key.$(date +%Y%m%d%H%M%S)
```
2. Stop service:
```bash
systemctl stop mxkeys
```
3. Rotate key by moving current key out of active path:
```bash
mv /var/lib/mxkeys/keys/mxkeys_ed25519.key /var/lib/mxkeys/keys/mxkeys_ed25519.key.rotated.$(date +%Y%m%d%H%M%S)
chmod 600 /var/lib/mxkeys/keys/mxkeys_ed25519.key.rotated.*
```
4. Start service and let MXKeys generate a new key:
```bash
systemctl start mxkeys
```
5. Validate post-rotation behavior:
```bash
curl -fsS https://mxkeys.example.org/_matrix/key/v2/server | jq '.verify_keys'
```
6. Record artifact: old key fingerprint, new key fingerprint, timestamp UTC, operator.

## Key Compromise Response Runbook

1. Immediate containment:
```bash
systemctl stop mxkeys
chmod 000 /var/lib/mxkeys/keys/mxkeys_ed25519.key || true
```
2. Preserve evidence copy (read-only):
```bash
install -d -m 700 /forensics/mxkeys
cp /var/lib/mxkeys/keys/mxkeys_ed25519.key /forensics/mxkeys/compromised_ed25519.key
chmod 400 /forensics/mxkeys/compromised_ed25519.key
```
3. Generate clean key by removing compromised key and starting service:
```bash
rm -f /var/lib/mxkeys/keys/mxkeys_ed25519.key
systemctl start mxkeys
```
4. Verify new key is active:
```bash
curl -fsS https://mxkeys.example.org/_matrix/key/v2/server | jq '.verify_keys'
```
5. Invalidate old trust in dependent systems (trusted key caches / pinning).
6. Record incident artifact: compromise window, new key id/fingerprint, mitigation timestamp UTC.

## Key Backup/Restore Drill

1. Backup key:
```bash
install -d -m 700 /backup/mxkeys-drill
install -m 600 /var/lib/mxkeys/keys/mxkeys_ed25519.key /backup/mxkeys-drill/mxkeys_ed25519.key
```
2. Simulate key loss and restore:
```bash
systemctl stop mxkeys
rm -f /var/lib/mxkeys/keys/mxkeys_ed25519.key
install -m 600 /backup/mxkeys-drill/mxkeys_ed25519.key /var/lib/mxkeys/keys/mxkeys_ed25519.key
systemctl start mxkeys
```
3. Validate restored key consistency:
```bash
curl -fsS https://mxkeys.example.org/_matrix/key/v2/server | jq '.verify_keys'
```
4. Required artifact fields: backup path, restore timestamp UTC, before/after key fingerprint, operator.

## Database Backup/Restore Drill

1. Create backup:
```bash
pg_dump mxkeys > /backup/mxkeys_db_drill_$(date +%Y%m%d%H%M%S).sql
```
2. Restore into isolated verification database:
```bash
createdb mxkeys_restore_drill
psql mxkeys_restore_drill < /backup/mxkeys_db_drill_YYYYMMDDHHMMSS.sql
```
3. Validate restored data:
```bash
psql mxkeys_restore_drill -c "\dt"
psql mxkeys_restore_drill -c "SELECT COUNT(*) FROM server_keys;"
```
4. Required artifact fields: backup file path, restore target DB, row-count verification output, timestamp UTC, operator.

## Disaster Recovery Runbook

1. Restore database from latest verified backup.
2. Restore signing key from latest verified key backup.
3. Validate key identity:
```bash
curl -fsS https://mxkeys.example.org/_matrix/key/v2/server | jq '.verify_keys'
```
4. Validate service readiness:
```bash
curl -fsS https://mxkeys.example.org/_mxkeys/ready
curl -fsS https://mxkeys.example.org/_mxkeys/status
```
5. Validate federation path:
```bash
curl -fsS -X POST https://mxkeys.example.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys":{"matrix.org":{}}}'
```
6. Required artifact fields: incident id, restore sources, key fingerprint, readiness output, federation query output, timestamp UTC.

## Monitoring

See [prometheus-alerts.yaml](prometheus-alerts.yaml) for alerting rules.

Scrape config:
```yaml
scrape_configs:
  - job_name: 'mxkeys'
    static_configs:
      - targets: ['mxkeys.example.org:8448']
    metrics_path: '/_mxkeys/metrics'
```

## Troubleshooting

### Check Health

```bash
curl https://mxkeys.example.org/_mxkeys/health
curl https://mxkeys.example.org/_mxkeys/ready
curl https://mxkeys.example.org/_mxkeys/status
```

### View Logs

```bash
journalctl -u mxkeys -f
```

### Reload Config

```bash
kill -HUP $(pgrep mxkeys)
```

### Test Query

```bash
curl -X POST https://mxkeys.example.org/_matrix/key/v2/query \
  -H "Content-Type: application/json" \
  -d '{"server_keys": {"matrix.org": {}}}'
```
