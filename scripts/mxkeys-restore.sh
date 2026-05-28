#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Created
#
# Restore a MXKeys node from a tarball produced by mxkeys-backup.sh.
#
# WARNING: this script is destructive. It replaces the contents of the
# target database (via the SQL dump inside the backup) and overwrites
# the keys + raft directories. Always operate on a stopped service.

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: mxkeys-restore.sh --input TARBALL [options]

Options:
  --input PATH           mxkeys-backup-*.tar.gz produced by mxkeys-backup.sh (required)
  --database-url URL     PostgreSQL URL (default: $MXKEYS_DATABASE_URL)
  --keys-dir DIR         Signing-key directory (default: /var/lib/mxkeys/keys)
  --raft-dir DIR         Raft state directory (default: /var/lib/mxkeys/raft)
  --skip-database        Do not restore the SQL dump
  --skip-keys            Do not overwrite keys dir
  --skip-raft            Do not overwrite raft dir
  --help                 Show this help
EOF
}

INPUT=""
DATABASE_URL="${MXKEYS_DATABASE_URL:-}"
KEYS_DIR="/var/lib/mxkeys/keys"
RAFT_DIR="/var/lib/mxkeys/raft"
SKIP_DB=0
SKIP_KEYS=0
SKIP_RAFT=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --input)         INPUT="$2"; shift 2 ;;
    --database-url)  DATABASE_URL="$2"; shift 2 ;;
    --keys-dir)      KEYS_DIR="$2"; shift 2 ;;
    --raft-dir)      RAFT_DIR="$2"; shift 2 ;;
    --skip-database) SKIP_DB=1; shift ;;
    --skip-keys)     SKIP_KEYS=1; shift ;;
    --skip-raft)     SKIP_RAFT=1; shift ;;
    --help|-h)       usage; exit 0 ;;
    *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -z "${INPUT}" || ! -f "${INPUT}" ]]; then
  echo "ERROR: --input must point to an existing backup tarball" >&2
  exit 2
fi
if [[ "${SKIP_DB}" -eq 0 && -z "${DATABASE_URL}" ]]; then
  echo "ERROR: --database-url or MXKEYS_DATABASE_URL is required (or pass --skip-database)" >&2
  exit 2
fi

workdir="$(mktemp -d -t mxkeys-restore-XXXXXX)"
trap 'rm -rf "${workdir}"' EXIT

echo "=> extract ${INPUT}"
tar -xzf "${INPUT}" -C "${workdir}"

# The tarball extracts as mxkeys-backup-<timestamp>/; find its single
# top-level entry rather than assuming the exact name.
mapfile -t roots < <(find "${workdir}" -mindepth 1 -maxdepth 1 -type d)
if [[ ${#roots[@]} -ne 1 ]]; then
  echo "ERROR: unexpected tarball layout (expected exactly one top-level directory)" >&2
  exit 3
fi
root="${roots[0]}"

echo "=> manifest:"
cat "${root}/MANIFEST.txt" || true

if [[ "${SKIP_DB}" -eq 0 ]]; then
  if [[ ! -f "${root}/database.sql" ]]; then
    echo "ERROR: backup has no database.sql" >&2
    exit 3
  fi
  if ! command -v psql >/dev/null 2>&1; then
    echo "ERROR: psql not found on PATH" >&2
    exit 2
  fi
  echo "=> restoring database"
  psql --quiet --set=ON_ERROR_STOP=1 "${DATABASE_URL}" < "${root}/database.sql"
fi

if [[ "${SKIP_KEYS}" -eq 0 ]]; then
  if [[ ! -d "${root}/keys" ]]; then
    echo "ERROR: backup has no keys directory" >&2
    exit 3
  fi
  echo "=> restoring keys into ${KEYS_DIR}"
  mkdir -p "$(dirname "${KEYS_DIR}")"
  rm -rf "${KEYS_DIR}"
  cp -a "${root}/keys" "${KEYS_DIR}"
  chmod -R u+rwX,go-rwx "${KEYS_DIR}" || true
fi

if [[ "${SKIP_RAFT}" -eq 0 ]]; then
  if [[ -d "${root}/raft" ]]; then
    echo "=> restoring raft state into ${RAFT_DIR}"
    mkdir -p "$(dirname "${RAFT_DIR}")"
    rm -rf "${RAFT_DIR}"
    cp -a "${root}/raft" "${RAFT_DIR}"
    chmod -R u+rwX,go-rwx "${RAFT_DIR}" || true
  else
    echo "=> backup has no raft state, nothing to restore"
  fi
fi

echo "Restore complete. Start the service and verify /_mxkeys/readyz returns 200."
