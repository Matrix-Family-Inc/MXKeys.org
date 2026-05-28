#!/bin/bash
# mxkeys-verify-exporter.sh
# Prometheus node_exporter textfile collector for mxkeys-verify
#
# Usage:
#   Place in /etc/cron.d/mxkeys-verify or run via systemd timer.
#   Output goes to node_exporter textfile directory.
#
# Requires:
#   - mxkeys-verify binary in PATH
#   - node_exporter --collector.textfile.directory=/var/lib/node_exporter/textfile
#
# Metrics exported:
#   mxkeys_sth_verify_success      1 if all checks passed, 0 otherwise
#   mxkeys_sth_tree_size           current tree size
#   mxkeys_sth_trust_level         achieved trust level (1-3)
#   mxkeys_sth_verify_duration_ms  verification duration in milliseconds

set -euo pipefail

MXKEYS_URL="${MXKEYS_URL:-https://mxkeys.example.org}"
STH_DIR="${STH_DIR:-/var/lib/mxkeys/sth-history}"
TEXTFILE_DIR="${TEXTFILE_DIR:-/var/lib/node_exporter/textfile}"
EXPECTED_FP="${MXKEYS_EXPECTED_FINGERPRINT:-}"

LATEST="$STH_DIR/sth-latest.json"
PROM_FILE="$TEXTFILE_DIR/mxkeys_verify.prom"
TMP_FILE="${PROM_FILE}.tmp"

mkdir -p "$STH_DIR" "$TEXTFILE_DIR"

PREV_FLAG=""
if [ -f "$LATEST" ]; then
    PREV_FLAG="-prev $LATEST"
fi

FP_FLAG=""
if [ -n "$EXPECTED_FP" ]; then
    FP_FLAG="-expected-fingerprint $EXPECTED_FP"
fi

START_MS=$(date +%s%3N)
OUTPUT=$(mxkeys-verify -url "$MXKEYS_URL" $PREV_FLAG -out "$LATEST" $FP_FLAG -json 2>&1) || true
END_MS=$(date +%s%3N)
DURATION=$((END_MS - START_MS))

OK=$(echo "$OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('ok', False))" 2>/dev/null || echo "False")
TREE_SIZE=$(echo "$OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tree_size', 0))" 2>/dev/null || echo "0")
TRUST_LEVEL=$(echo "$OUTPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('trust_level', 0))" 2>/dev/null || echo "0")

SUCCESS=0
if [ "$OK" = "True" ]; then
    SUCCESS=1
fi

cat > "$TMP_FILE" <<METRICS
# HELP mxkeys_sth_verify_success Whether the last STH verification succeeded
# TYPE mxkeys_sth_verify_success gauge
mxkeys_sth_verify_success $SUCCESS
# HELP mxkeys_sth_tree_size Current transparency log tree size
# TYPE mxkeys_sth_tree_size gauge
mxkeys_sth_tree_size $TREE_SIZE
# HELP mxkeys_sth_trust_level Achieved trust level (1=transport, 2=self-consistency, 3=origin)
# TYPE mxkeys_sth_trust_level gauge
mxkeys_sth_trust_level $TRUST_LEVEL
# HELP mxkeys_sth_verify_duration_ms Verification duration in milliseconds
# TYPE mxkeys_sth_verify_duration_ms gauge
mxkeys_sth_verify_duration_ms $DURATION
METRICS

mv "$TMP_FILE" "$PROM_FILE"
