#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Created
#
# Enforce the file-size policy from ADR-0010:
#   - target: 250-300 lines per file
#   - warn at 300 lines
#   - fail at 500 lines (hard ceiling)
#
# Scope: Go, TypeScript, and shell source files that are tracked in
# git. Generated files, vendored directories, node_modules, and test
# data are excluded by the git-ls-files path filters.

set -euo pipefail

WARN_AT="${FILE_SIZE_LINT_WARN:-300}"
FAIL_AT="${FILE_SIZE_LINT_FAIL:-500}"

patterns=(
  '*.go'
  '*.ts'
  '*.tsx'
  '*.js'
  '*.mjs'
  '*.sh'
)

exclude_globs=(
  ':(exclude)landing/node_modules/**'
  ':(exclude)**/vendor/**'
  ':(exclude)**/*.pb.go'
  ':(exclude)**/*_generated.go'
  ':(exclude)landing/dist/**'
)

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "file-size-lint: not a git worktree; nothing to check."
  exit 0
fi

mapfile -t files < <(git ls-files -- "${patterns[@]}" "${exclude_globs[@]}")

warn=0
fail=0
for f in "${files[@]}"; do
  # Guard against files deleted on the working tree but still in the
  # index; skip those rather than erroring.
  if [[ ! -f "${f}" ]]; then
    continue
  fi
  lines="$(wc -l < "${f}")"
  if [[ "${lines}" -gt "${FAIL_AT}" ]]; then
    printf '  FAIL  %5d lines  %s\n' "${lines}" "${f}" >&2
    fail=$((fail + 1))
  elif [[ "${lines}" -gt "${WARN_AT}" ]]; then
    printf '  warn  %5d lines  %s\n' "${lines}" "${f}"
    warn=$((warn + 1))
  fi
done

echo
if [[ "${fail}" -gt 0 ]]; then
  echo "file-size-lint: ${fail} file(s) above hard ceiling ${FAIL_AT} (see ADR-0010)." >&2
  exit 1
fi
if [[ "${warn}" -gt 0 ]]; then
  echo "file-size-lint: ${warn} file(s) above target warn threshold ${WARN_AT} (not fatal)."
else
  echo "file-size-lint: all tracked source files are within the ${WARN_AT}-line target."
fi
