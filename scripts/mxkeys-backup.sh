#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Created
#
# Offline backup script for a single MXKeys notary node.
#
# Produces a timestamped tarball that contains:
#   - a pg_dump of the notary database,
#   - the signing-key file (plaintext or encrypted, whichever the
#     operator configured),
#   - the Raft state directory (WAL + snapshot), if present,
#   - a manifest with checksums and the running service version.
#
# Designed to be called from systemd timer, cron, or k8s CronJob.
# Invariants:
#   - fails fast on any missing input rather than producing a partial
#     backup,
#   - writes the output atomically (temp file + rename) so the backup
#     directory never contains a half-written tarball,
#   - restricts the output file permissions to 0600.
#
# Restore: see mxkeys-restore.sh and docs/runbook/backup-restore.md.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: mxkeys-backup.sh [options]

Options:
  --database-url URL     PostgreSQL URL (default: $MXKEYS_DATABASE_URL)
  --keys-dir DIR         Signing-key directory (default: /var/lib/mxkeys/keys)
  --raft-dir DIR         Raft state directory (default: /var/lib/mxkeys/raft, skipped if missing)
  --output-dir DIR       Where to place backup-*.tar.gz (default: /var/backups/mxkeys)
  --label LABEL          Optional label appended to the filename
  --version-binary BIN   Path to the mxkeys binary for version capture (default: mxkeys)
  --help                 Show this help

Environment fallbacks:
  MXKEYS_DATABASE_URL    used when --database-url not provided
  MXKEYS_BACKUP_OUTPUT   used when --output-dir not provided
EOF
}

DATABASE_URL="${MXKEYS_DATABASE_URL:-}"
KEYS_DIR="/var/lib/mxkeys/keys"
RAFT_DIR="/var/lib/mxkeys/raft"
OUTPUT_DIR="${MXKEYS_BACKUP_OUTPUT:-/var/backups/mxkeys}"
LABEL=""
VERSION_BIN="mxkeys"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --database-url) DATABASE_URL="$2"; shift 2 ;;
    --keys-dir)     KEYS_DIR="$2"; shift 2 ;;
    --raft-dir)     RAFT_DIR="$2"; shift 2 ;;
    --output-dir)   OUTPUT_DIR="$2"; shift 2 ;;
    --label)        LABEL="$2"; shift 2 ;;
    --version-binary) VERSION_BIN="$2"; shift 2 ;;
    --help|-h)      usage; exit 0 ;;
    *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "${DATABASE_URL}" ]]; then
  echo "ERROR: --database-url or MXKEYS_DATABASE_URL is required" >&2
  exit 2
fi
if [[ ! -d "${KEYS_DIR}" ]]; then
  echo "ERROR: keys dir not found: ${KEYS_DIR}" >&2
  exit 2
fi
if ! command -v pg_dump >/dev/null 2>&1; then
  echo "ERROR: pg_dump not found on PATH" >&2
  exit 2
fi

mkdir -p "${OUTPUT_DIR}"
chmod 0700 "${OUTPUT_DIR}" || true

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
if [[ -n "${LABEL}" ]]; then
  basename="mxkeys-backup-${timestamp}-${LABEL}"
else
  basename="mxkeys-backup-${timestamp}"
fi

workdir="$(mktemp -d -t mxkeys-backup-XXXXXX)"
trap 'rm -rf "${workdir}"' EXIT

staging="${workdir}/${basename}"
mkdir -p "${staging}"

echo "=> pg_dump"
pg_dump --no-owner --no-privileges --clean --if-exists \
  --file="${staging}/database.sql" \
  "${DATABASE_URL}"

echo "=> copy keys dir"
cp -a "${KEYS_DIR}" "${staging}/keys"

if [[ -d "${RAFT_DIR}" ]]; then
  echo "=> copy raft state"
  cp -a "${RAFT_DIR}" "${staging}/raft"
else
  echo "=> no raft state dir at ${RAFT_DIR}, skipping"
fi

echo "=> manifest"
{
  echo "timestamp=${timestamp}"
  if command -v "${VERSION_BIN}" >/dev/null 2>&1; then
    echo "mxkeys_version=$(${VERSION_BIN} -version 2>/dev/null || true)"
  fi
  echo
  echo "checksums (sha256):"
  (cd "${staging}" && find . -type f -print0 | xargs -0 sha256sum)
} > "${staging}/MANIFEST.txt"

echo "=> tar.gz"
tarball="${OUTPUT_DIR}/${basename}.tar.gz"
tmptar="${tarball}.tmp"
(cd "${workdir}" && tar --sort=name --owner=0 --group=0 --mtime=@0 -czf "${tmptar}" "${basename}")
chmod 0600 "${tmptar}"
mv "${tmptar}" "${tarball}"

echo "Backup complete: ${tarball}"
