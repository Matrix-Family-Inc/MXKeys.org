#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Wed Apr 22 2026 UTC
# Status: Created
#
# Idempotent prod mxkeys binary swap for mxkeys.service on
# 82.21.114.30 (single-host setup).
#
# What it does, in order:
#   1. Verify a release binary exists locally, is linux/amd64, and
#      reports a version string via `-version`.
#   2. scp the binary to /opt/mxkeys/mxkeys.new on the remote.
#   3. sha256 parity check (local vs remote) to catch transport
#      corruption.
#   4. Remote pre-swap: pg_dump mxkeys → /opt/mxkeys/
#      db_backup_before_v<version>_<timestamp>.sql; keep the
#      current binary as /opt/mxkeys/mxkeys.bak.v<old-version>.<timestamp>.
#   5. Remote swap: systemctl stop mxkeys, install -m 755 new →
#      mxkeys, systemctl start mxkeys, sleep 3 for migration log.
#   6. Health + version probe on both the local :8448 and the
#      public https://mxkeys.org/_mxkeys/health surface.
#   7. Refuse to return success unless the reported version matches
#      the binary name / expected value.
#
# The pre-swap pg_dump IS the rollback plan. On failure restore
# by: stop service, copy backed-up binary over mxkeys, start.
# Database rollback is almost never needed because schema
# migrations are additive and idempotent, but keep the dump for
# catastrophic cases.
#
# Usage (from repo root on the build host):
#
#     bash scripts/deploy-mxkeys.sh <remote-host> <local-binary> <expected-version>
#
# Example:
#
#     bash scripts/deploy-mxkeys.sh root@82.21.114.30 \
#          bin/mxkeys-linux-amd64-v1.0.0 1.0.0

set -euo pipefail

REMOTE="${1:-}"
LOCAL_BIN="${2:-}"
EXPECTED_VERSION="${3:-}"

if [[ -z "$REMOTE" || -z "$LOCAL_BIN" || -z "$EXPECTED_VERSION" ]]; then
  echo "usage: $0 <remote-host> <local-binary> <expected-version>" >&2
  exit 2
fi

REMOTE_PATH="/opt/mxkeys/mxkeys"

# Step 1 ─ local sanity.
if [[ ! -x "$LOCAL_BIN" ]]; then
  echo "fatal: $LOCAL_BIN not executable" >&2
  exit 1
fi
FILE_INFO="$(file -b "$LOCAL_BIN")"
if ! grep -q 'ELF 64-bit LSB executable, x86-64' <<<"$FILE_INFO"; then
  echo "fatal: $LOCAL_BIN is not linux/amd64 ELF: $FILE_INFO" >&2
  exit 1
fi
LOCAL_VERSION="$("$LOCAL_BIN" -version | head -1)"
if ! grep -q "MXKeys/${EXPECTED_VERSION}" <<<"$LOCAL_VERSION"; then
  echo "fatal: local binary reports '$LOCAL_VERSION', expected 'MXKeys/${EXPECTED_VERSION}'" >&2
  exit 1
fi
echo "[1/7] local binary ok: $LOCAL_VERSION ($(du -h "$LOCAL_BIN" | awk '{print $1}'))"

# Step 2 ─ upload to staging path.
echo "[2/7] upload → ${REMOTE}:${REMOTE_PATH}.new"
scp -o BatchMode=yes "$LOCAL_BIN" "${REMOTE}:${REMOTE_PATH}.new"

# Step 3 ─ sha256 parity.
LOCAL_SHA="$(sha256sum "$LOCAL_BIN" | awk '{print $1}')"
REMOTE_SHA="$(ssh -o BatchMode=yes "$REMOTE" "sha256sum '${REMOTE_PATH}.new' | awk '{print \$1}'")"
if [[ "$LOCAL_SHA" != "$REMOTE_SHA" ]]; then
  echo "fatal: sha256 mismatch after scp (local=$LOCAL_SHA remote=$REMOTE_SHA)" >&2
  exit 1
fi
echo "[3/7] sha256 parity ok"

# Step 4 ─ pre-swap backups on remote (DB + current binary).
echo "[4/7] pg_dump + backup current binary"
ssh -o BatchMode=yes "$REMOTE" "
  set -e
  TS=\$(date +%Y%m%d_%H%M%S)
  BACKUP_SQL=/opt/mxkeys/db_backup_before_v${EXPECTED_VERSION}_\${TS}.sql
  sudo -u postgres pg_dump mxkeys > \"\$BACKUP_SQL\"
  ls -la \"\$BACKUP_SQL\"

  if [ -x '$REMOTE_PATH' ]; then
    OLD_VERSION=\$('$REMOTE_PATH' -version 2>/dev/null | awk -F/ '{print \$2}')
    [ -z \"\$OLD_VERSION\" ] && OLD_VERSION=unknown
    BACKUP_BIN=/opt/mxkeys/mxkeys.bak.v\${OLD_VERSION}.\${TS}
    cp -p '$REMOTE_PATH' \"\$BACKUP_BIN\"
    echo '  rollback binary: '\"\$BACKUP_BIN\"
  fi
"

# Step 5 ─ swap.
echo "[5/7] stop + swap + start"
ssh -o BatchMode=yes "$REMOTE" "
  set -e
  systemctl stop mxkeys.service
  install -m 755 -o root -g root '${REMOTE_PATH}.new' '${REMOTE_PATH}'
  rm -f '${REMOTE_PATH}.new'
  systemctl start mxkeys.service
"
sleep 3

# Step 6 ─ probes.
echo "[6/7] health + version probes"
ssh -o BatchMode=yes "$REMOTE" "systemctl is-active mxkeys.service"
LOCAL_PROBE="$(ssh -o BatchMode=yes "$REMOTE" 'curl -sS --max-time 5 http://127.0.0.1:8448/_mxkeys/health')"
PUBLIC_PROBE="$(curl -sS --max-time 5 https://mxkeys.org/_mxkeys/health)"
echo "  local  (127.0.0.1:8448): $LOCAL_PROBE"
echo "  public (https://…      ): $PUBLIC_PROBE"

# Step 7 ─ version strictness.
if ! grep -q "\"version\":\"${EXPECTED_VERSION}\"" <<<"$PUBLIC_PROBE"; then
  echo "fatal: public /_mxkeys/health does not report ${EXPECTED_VERSION}" >&2
  exit 6
fi
echo "[7/7] version check ok: ${EXPECTED_VERSION}"

echo
echo "mxkeys deploy OK"
echo "  binary : ${REMOTE_PATH} (sha256 ${LOCAL_SHA})"
echo "  version: ${EXPECTED_VERSION}"
echo
echo "rollback (if needed):"
echo "  ssh ${REMOTE} 'systemctl stop mxkeys \\"
echo "                && install -m 755 /opt/mxkeys/mxkeys.bak.v<old>.<ts> /opt/mxkeys/mxkeys \\"
echo "                && systemctl start mxkeys'"
