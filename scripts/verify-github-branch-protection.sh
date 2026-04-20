#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Mar 16 2026 UTC
# Status: Created

set -euo pipefail

repo="${GITHUB_REPOSITORY:-}"
branch="${GITHUB_BRANCH:-main}"
token="${GITHUB_TOKEN:-${GH_TOKEN:-}}"
profile="${GITHUB_CHECK_PROFILE:-pr}"

# Keep these required check names aligned with .github/workflows/pr-gate.yml
# and .github/workflows/release-live-interop-gate.yml.

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

case "${profile}" in
  pr)
    required_checks=(
      "unit"
      "integration-with-fixtures"
      "integration-tagged"
      "race"
      "vet"
      "lint"
      "frontend-quality"
      "security-vuln"
      "security-sast"
    )
    ;;
  release)
    required_checks=(
      "live-federation-strictness"
      "live-query-compatibility"
      "live-notary-interop"
    )
    ;;
  *)
    echo "ERROR: unsupported GITHUB_CHECK_PROFILE=${profile} (expected pr or release)"
    exit 2
    ;;
esac

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

echo "Branch protection verification passed for ${repo}:${branch} (${profile})"
