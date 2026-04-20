#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Created
#
# Coverage gate for CI. Runs the unit suite with -coverprofile and fails when:
#   * total coverage drops below COVERAGE_TOTAL_MIN percent
#   * any package with an explicit floor in COVERAGE_PACKAGE_FLOORS drops
#     below its declared minimum
#
# Thresholds are intentionally tracked in this script (not in YAML) so that
# commits bumping the floors are auditable and reviewable like any other
# code change.

set -euo pipefail

# Tunable thresholds. Raise floors after real improvements land; never
# lower silently: drops must be explicit in the diff.
COVERAGE_TOTAL_MIN="${COVERAGE_TOTAL_MIN:-50}"

# Per-package floors. Format: "<import path>=<min percent>". Packages not
# listed here are subject only to the total-coverage floor.
declare -a COVERAGE_PACKAGE_FLOORS=(
  "mxkeys/internal/config=80"
  "mxkeys/internal/zero/config=85"
  "mxkeys/internal/zero/log=80"
  "mxkeys/internal/zero/merkle=65"
  "mxkeys/internal/zero/raft=60"
  "mxkeys/internal/keys/keyprovider=65"
  "mxkeys/internal/cluster=60"
  "mxkeys/internal/server=55"
  # Raised as part of the Phase 9 hardening pass. Floors reflect the
  # actual current coverage minus a small cushion so ordinary refactors
  # do not trip the gate. Raise deliberately when new tests land.
  "mxkeys/internal/keys=40"
  "mxkeys/internal/zero/canonical=40"
  "mxkeys/internal/zero/metrics=35"
  "mxkeys/internal/storage/migrations=30"
)

profile="$(mktemp -t mxkeys-cov.XXXXXX.out)"
test_output="$(mktemp -t mxkeys-cov-out.XXXXXX.txt)"
trap 'rm -f "${profile}" "${test_output}"' EXIT

packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"

echo "[1/3] go test -coverprofile"
# Tee test output so per-package "coverage: X.X% of statements" lines can
# be parsed for the gate; go test already emits these so there is no need
# for a second pass.
go test -count=1 -coverprofile="${profile}" -covermode=atomic ${packages} | tee "${test_output}"

echo "[2/3] parse total"
total_line="$(go tool cover -func="${profile}" | tail -1)"
total_pct="$(echo "${total_line}" | awk '{print $NF}' | sed 's/%//')"
echo "Total coverage: ${total_pct}% (floor ${COVERAGE_TOTAL_MIN}%)"

awk -v have="${total_pct}" -v want="${COVERAGE_TOTAL_MIN}" 'BEGIN { exit (have + 0 < want + 0) ? 1 : 0 }' || {
  echo "FAIL: total coverage ${total_pct}% below floor ${COVERAGE_TOTAL_MIN}%" >&2
  exit 1
}

echo "[3/3] per-package floors"
# go test output format per line:
#   ok    mxkeys/internal/foo    0.050s    coverage: 67.2% of statements
# Extract the percent from the matching line; this is the package's true
# weighted coverage (as reported by the Go toolchain).
failed=0
for pair in "${COVERAGE_PACKAGE_FLOORS[@]}"; do
  pkg="${pair%=*}"
  floor="${pair#*=}"
  pkg_pct="$(awk -v p="${pkg}" '
    {
      # Find a line that mentions the exact package path followed by a
      # tab/space and ends with coverage info.
      if ($0 ~ ("[ \t]" p "[ \t]") && $0 ~ /coverage:/) {
        for (i = 1; i <= NF; i++) {
          if ($i == "coverage:") {
            sub(/%/, "", $(i+1))
            print $(i+1)
            exit
          }
        }
      }
    }
  ' "${test_output}")"
  if [[ -z "${pkg_pct}" ]]; then
    echo "WARN: no coverage data for ${pkg}; skipping floor check"
    continue
  fi
  echo "${pkg}: ${pkg_pct}% (floor ${floor}%)"
  awk -v have="${pkg_pct}" -v want="${floor}" 'BEGIN { exit (have + 0 < want + 0) ? 1 : 0 }' || {
    echo "  FAIL: ${pkg} coverage ${pkg_pct}% below floor ${floor}%" >&2
    failed=1
  }
done

if [[ "${failed}" -ne 0 ]]; then
  echo "Coverage gate FAILED."
  exit 1
fi

echo "Coverage gate passed."
