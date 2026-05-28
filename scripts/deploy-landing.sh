#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Wed Apr 22 2026 UTC
# Status: Created
#
# Idempotent prod-landing deploy for https://mxkeys.org/.
#
# What it does, in order:
#   1. Validate that a local landing `dist/` exists and looks sane.
#   2. Snapshot the currently-served tree to
#      `/opt/MXKeys.org.prev.<timestamp>` so a rollback is a plain
#      `mv`.
#   3. rsync `dist/` into `/opt/MXKeys.org/` with `--delete` so
#      stale hashed bundles are removed (Vite asset hashes rotate
#      per content).
#   4. chown -R to the canonical landing owner so permissions stay
#      consistent between rsync-from-root and rsync-from-CI-user.
#   5. Probe `https://mxkeys.org/` for presence of the expected
#      release string.
#
# nginx reload is NOT required: the file `root /opt/MXKeys.org;`
# is static, nginx re-stats files on every request. Cache-Control
# is `public, immutable` on hashed assets; hashed filenames change
# on every build so clients never see a stale bundle.
#
# Usage (from repo root on the build host):
#
#     bash scripts/deploy-landing.sh <remote-host> [expected-version]
#
# Example:
#
#     bash scripts/deploy-landing.sh root@82.21.114.30 1.0.0
#
# Exits non-zero on any step failure. Safe to re-run after a
# partial failure: the snapshot it creates is timestamped per
# invocation and never overwrites.

set -euo pipefail

REMOTE="${1:-}"
EXPECTED_VERSION="${2:-}"

if [[ -z "$REMOTE" ]]; then
  echo "usage: $0 <remote-host> [expected-version]" >&2
  exit 2
fi

LANDING_DIST="landing/dist"
REMOTE_ROOT="/opt/MXKeys.org"
REMOTE_OWNER="1000:1000"
HEALTH_URL="https://mxkeys.org/"

# Step 1 ─ local sanity.
if [[ ! -d "$LANDING_DIST" ]]; then
  echo "fatal: $LANDING_DIST not found. run 'bun run build' in landing/ first." >&2
  exit 1
fi
if [[ ! -f "$LANDING_DIST/index.html" ]]; then
  echo "fatal: $LANDING_DIST/index.html missing." >&2
  exit 1
fi
if ! ls "$LANDING_DIST"/assets/index-*.js >/dev/null 2>&1; then
  echo "fatal: $LANDING_DIST/assets/index-*.js missing (no main bundle)." >&2
  exit 1
fi

echo "[1/5] local dist ok: $(du -sh "$LANDING_DIST" | awk '{print $1}')"

# Step 2 ─ snapshot current tree on remote.
STAMP="$(date -u +%Y%m%d_%H%M%S)"
SNAPSHOT="/opt/MXKeys.org.prev.${STAMP}"
echo "[2/5] snapshot current tree → ${SNAPSHOT}"
ssh -o BatchMode=yes "$REMOTE" "
  set -e
  if [ -d '$REMOTE_ROOT' ]; then
    cp -a '$REMOTE_ROOT' '$SNAPSHOT'
    echo '  snapshot size:' \$(du -sh '$SNAPSHOT' | awk '{print \$1}')
  else
    echo '  no existing $REMOTE_ROOT; skipping snapshot'
  fi
"

# Step 3 ─ rsync with --delete.
echo "[3/5] rsync ${LANDING_DIST}/ → ${REMOTE}:${REMOTE_ROOT}/"
rsync -az --delete \
      -e 'ssh -o BatchMode=yes' \
      "${LANDING_DIST}/" \
      "${REMOTE}:${REMOTE_ROOT}/"

# Step 4 ─ unify ownership.
echo "[4/5] chown ${REMOTE_ROOT} → ${REMOTE_OWNER}"
ssh -o BatchMode=yes "$REMOTE" "chown -R '$REMOTE_OWNER' '$REMOTE_ROOT'"

# Step 5 ─ http probe.
echo "[5/5] verify ${HEALTH_URL}"
BODY="$(curl -sS --max-time 10 "$HEALTH_URL")"
SIZE="$(printf %s "$BODY" | wc -c)"
if [[ "$SIZE" -lt 100 ]]; then
  echo "fatal: ${HEALTH_URL} returned ${SIZE} bytes (likely nginx 5xx page)" >&2
  exit 3
fi

# Main bundle hash changed on every build; confirm at least that the
# new bundle is reachable and does not carry the old organisation
# slug.
MAIN_JS="$(printf %s "$BODY" | grep -oE '/assets/index-[A-Za-z0-9_-]+\.js' | head -1 || true)"
if [[ -z "$MAIN_JS" ]]; then
  echo "fatal: could not locate /assets/index-*.js reference in served index.html" >&2
  exit 3
fi
JS_URL="https://mxkeys.org${MAIN_JS}"
JS_BODY="$(curl -sS --max-time 15 "$JS_URL")"
if [[ "$(printf %s "$JS_BODY" | wc -c)" -lt 10000 ]]; then
  echo "fatal: ${JS_URL} looks empty / error page" >&2
  exit 3
fi
if printf %s "$JS_BODY" | grep -q 'github\.com/matrixfamily/'; then
  echo "fatal: new bundle still contains stale matrixfamily/ GitHub URL" >&2
  exit 4
fi
if [[ -n "$EXPECTED_VERSION" ]]; then
  if ! printf %s "$JS_BODY" | grep -q "v${EXPECTED_VERSION}\|V${EXPECTED_VERSION}\|${EXPECTED_VERSION}"; then
    echo "fatal: expected version ${EXPECTED_VERSION} not present in served bundle" >&2
    exit 5
  fi
  echo "  ok: v${EXPECTED_VERSION} marker present in served bundle"
fi

echo
echo "landing deploy OK"
echo "  snapshot: ${SNAPSHOT}"
echo "  remote :  ${REMOTE}:${REMOTE_ROOT}/"
echo "  probe  :  ${HEALTH_URL} ($(printf %s "$BODY" | wc -c) bytes index.html)"
echo
echo "rollback (if needed):"
echo "  ssh ${REMOTE} \"rm -rf ${REMOTE_ROOT} && mv ${SNAPSHOT} ${REMOTE_ROOT}\""
