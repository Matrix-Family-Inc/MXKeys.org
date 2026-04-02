#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Owner: Matrix Family Inc.
# Maintainer: Brabus
# Role: Lead Architect
# Contact: dev@matrix.family
# Support: support@matrix.family
# Matrix: @support:matrix.family
# Date: Mon Mar 16 2026 UTC
# Status: Created

set -euo pipefail

repo="${GITHUB_REPOSITORY:-}"
branch="${GITHUB_BRANCH:-main}"
token="${GITHUB_TOKEN:-${GH_TOKEN:-}}"

if [[ -z "${repo}" ]]; then
  echo "ERROR: set GITHUB_REPOSITORY=owner/repo"
  exit 2
fi

if [[ -z "${token}" ]]; then
  echo "ERROR: set GITHUB_TOKEN or GH_TOKEN"
  exit 2
fi

api="https://api.github.com/repos/${repo}/branches/${branch}/protection"
resp="$(curl -fsSL \
  -H "Authorization: Bearer ${token}" \
  -H "Accept: application/vnd.github+json" \
  "${api}")"

required_checks=(
  "unit"
  "integration-with-fixtures"
  "race"
  "vet"
  "lint"
  "live-federation-strictness"
  "live-query-compatibility"
)

contexts="$(jq -r '.required_status_checks.contexts[]?' <<<"${resp}")"

missing=0
for check in "${required_checks[@]}"; do
  if ! grep -Fxq "${check}" <<<"${contexts}"; then
    echo "MISSING required check: ${check}"
    missing=1
  fi
done

strict="$(jq -r '.required_status_checks.strict // false' <<<"${resp}")"
if [[ "${strict}" != "true" ]]; then
  echo "MISSING strict up-to-date requirement (required_status_checks.strict=true)"
  missing=1
fi

if [[ "${missing}" -ne 0 ]]; then
  exit 1
fi

echo "Branch protection verification passed for ${repo}:${branch}"
