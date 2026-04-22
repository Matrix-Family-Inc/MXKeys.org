Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Wed Apr 22 2026 UTC
Status: Created

# Runbook — production deploy (mxkeys.org)

Canonical procedure for pushing a new MXKeys release to the public
`https://mxkeys.org` host. Two artefacts, two commands, identical
naming. Every step is idempotent and carries its own rollback
plan.

## Production layout

Single host, `82.21.114.30`, hostname `MXKeys`. Only ports 80 and
443 face the internet (nginx). `mxkeys` and `postgresql` are bound
to `127.0.0.1`.

| Artefact          | Canonical path                               |
|-------------------|----------------------------------------------|
| Go binary         | `/opt/mxkeys/mxkeys`                         |
| Go config         | `/opt/mxkeys/config.yaml`                    |
| Signing keys      | `/var/lib/mxkeys/keys/`                      |
| Landing build     | `/opt/MXKeys.org/` (nginx `root`)            |
| Landing snapshots | `/opt/MXKeys.org.prev.<timestamp>` (rollback)|
| Archive bin/tgz   | `/opt/deploy-backups/`                       |
| systemd unit      | `/etc/systemd/system/mxkeys.service`         |
| nginx vhost       | `/etc/nginx/sites-enabled/mxkeys.org`        |
| postgres database | `mxkeys` (owner role `mxkeys`)               |
| DB dumps          | `/opt/mxkeys/db_backup_before_v*_*.sql`      |

nginx vhost contract (summary):

- `root /opt/MXKeys.org; index index.html;` serves landing statics.
- `location /` carries an SPA `try_files $uri $uri/ /index.html`.
- `location /_matrix/key/`, `/_matrix/federation/v1/version`,
  `/_mxkeys/` each `proxy_pass http://127.0.0.1:8448;` to the Go
  service.
- `location = /.well-known/matrix/server` returns inline JSON
  `{"m.server": "mxkeys.org:443"}`.

## Prerequisites

On the build host (wherever this runbook is executed from):

- SSH config alias with `root@82.21.114.30` reachable under the
  corporate key (see `matrix-family-info.md`).
- `rsync`, `scp`, `curl`, `sha256sum`, `file`, `bash` on PATH.
- Go toolchain for binary builds; Bun for landing builds.
- Working tree at the tag you intend to deploy (e.g. `git checkout v1.0.0`).

## 1) Deploy the Go binary

```bash
# 1.1 reproducible release build (host-independent output)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o bin/mxkeys-linux-amd64-v${VERSION} \
    ./cmd/mxkeys

# 1.2 push to prod (this script does: scp + sha256 verify + pg_dump
#     + keep rollback bin + systemctl stop/start + health probe)
bash scripts/deploy-mxkeys.sh \
    root@82.21.114.30 \
    bin/mxkeys-linux-amd64-v${VERSION} \
    ${VERSION}
```

The script refuses to return success unless:

- The local binary reports `MXKeys/${VERSION}` via `-version`.
- `sha256sum` matches after `scp`.
- `systemctl is-active mxkeys.service` is `active`.
- Both `http://127.0.0.1:8448/_mxkeys/health` and
  `https://mxkeys.org/_mxkeys/health` report `"version":"${VERSION}"`.

Rollback: stop service, install the backup binary that the script
wrote as `/opt/mxkeys/mxkeys.bak.v<old>.<timestamp>`, start service.
Database rollback is almost never needed because schema migrations
are additive and idempotent; the paired `db_backup_before_v*.sql`
exists for catastrophic cases only.

## 2) Deploy the landing

```bash
# 2.1 fresh dist
cd landing && bun run build && cd ..

# 2.2 push to prod (this script does: snapshot → rsync --delete →
#     chown → HTTP probe with version + GitHub-URL sanity checks)
bash scripts/deploy-landing.sh \
    root@82.21.114.30 \
    ${VERSION}
```

The script refuses to return success unless:

- `landing/dist/index.html` and `landing/dist/assets/index-*.js`
  exist locally.
- `https://mxkeys.org/` returns a non-trivial body.
- The main bundle referenced in that HTML is reachable over HTTPS.
- The main bundle does NOT contain `github.com/matrixfamily/` (stale
  org slug).
- The main bundle does contain the expected `v${VERSION}` marker.

Rollback: the script wrote a snapshot at
`/opt/MXKeys.org.prev.<timestamp>`. Restore with:

```bash
ssh root@82.21.114.30 '
    rm -rf /opt/MXKeys.org &&
    mv /opt/MXKeys.org.prev.<timestamp> /opt/MXKeys.org
'
```

nginx does not need a reload; it stats files per request.

## 3) Verify the release end-to-end

```bash
# Go service version + health
curl -sS https://mxkeys.org/_matrix/federation/v1/version
curl -sS https://mxkeys.org/_mxkeys/health

# Landing markers
curl -sS https://mxkeys.org/ | grep -oE '/assets/index-[^"]+\.js' | head -1

# Byte-exact origin-signature invariant (raw-preserving pipeline)
python3 - <<'PY'
import json, base64, urllib.request
import nacl.signing, nacl.exceptions
r = urllib.request.urlopen('https://mxkeys.org/_matrix/key/v2/query', data=b'{"server_keys":{"matrix.org":{}}}',
                           timeout=10, headers={'Content-Type': 'application/json'})
entry = json.load(r)['server_keys'][0]
signable = {k: v for k, v in entry.items() if k not in ('signatures', 'unsigned')}
canonical = json.dumps(signable, sort_keys=True, separators=(',', ':'), ensure_ascii=False).encode()
srv = entry['server_name']
pad = lambda s: s + '=' * ((4 - len(s) % 4) % 4)
for kid, sig in entry['signatures'][srv].items():
    pub = base64.b64decode(pad(entry['verify_keys'][kid]['key']))
    nacl.signing.VerifyKey(pub).verify(canonical, base64.b64decode(pad(sig)))
    print(f'origin sig {srv}:{kid} verifies')
PY
```

If origin signature verification fails, the raw-preserving
pipeline (`ServerKeysResponse.Raw`, migration 0003) has regressed.
Treat as a release blocker and rollback the Go binary first;
landing does not affect this invariant.

## Housekeeping

After every deploy, the script leaves one timestamped snapshot of
the previous tree per artefact. A quarterly cron or manual sweep
should move snapshots older than 30 days into
`/opt/deploy-backups/` as tarballs:

```bash
ssh root@82.21.114.30 '
    for d in /opt/MXKeys.org.prev.* ; do
        [ -d "$d" ] || continue
        age=$(( ( $(date +%s) - $(stat -c %Y "$d") ) / 86400 ))
        if [ "$age" -gt 30 ]; then
            base=$(basename "$d")
            tar czf "/opt/deploy-backups/${base}.tgz" -C /opt "$base"
            rm -rf "$d"
        fi
    done
'
```

DB dumps (`/opt/mxkeys/db_backup_before_v*.sql`) follow the same
30-day retention in `/opt/deploy-backups/` for completeness.

## Why no CI auto-deploy

Prod is a single host with no auto-deploy pipeline by design.
Every release is a deliberate two-command sequence that the
operator runs after merging to `main` and tagging. Auto-deploy
behind branch protection is on the 1.x roadmap; until it lands
the manual path documented here is the only supported flow.
