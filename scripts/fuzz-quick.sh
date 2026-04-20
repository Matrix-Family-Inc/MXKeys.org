#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Created
#
# Runs every declared fuzz target for a short interval (30s by default)
# in CI. Long fuzzing campaigns happen out-of-band; this gate catches
# regressions in the parser hardening layer on every PR.
#
# Each entry in FUZZ_TARGETS is "<package>:<FuzzFunc>". To add a new target,
# append it here; the loop handles the rest.

set -euo pipefail

FUZZTIME="${FUZZTIME:-30s}"

FUZZ_TARGETS=(
  "./internal/zero/canonical:FuzzJSON"
  "./internal/zero/canonical:FuzzMarshalRoundTrip"
  "./internal/server:FuzzValidateServerName"
  "./internal/server:FuzzValidateKeyID"
  "./internal/server:FuzzDecodeStrictJSON"
  "./internal/keys:FuzzParseServerName"
)

failed=0
for target in "${FUZZ_TARGETS[@]}"; do
  pkg="${target%%:*}"
  fn="${target#*:}"
  echo "=== fuzz ${pkg} ${fn} (${FUZZTIME}) ==="
  if ! go test "${pkg}" -run=^$ -fuzz="^${fn}$" -fuzztime="${FUZZTIME}"; then
    echo "FAIL: fuzz target ${pkg} ${fn} reported a failure" >&2
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  echo "fuzz-quick FAILED."
  exit 1
fi

echo "fuzz-quick passed."
