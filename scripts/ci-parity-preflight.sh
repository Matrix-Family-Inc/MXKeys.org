#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Updated
#
# Locally reproduces every check that the PR gate runs on GitHub Actions.
# Keep this script in lock-step with .github/workflows/pr-gate.yml; when a
# new job is added to CI it must be added here too (and referenced by
# verify-github-branch-protection.sh as a required status check).

set -euo pipefail

packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"
package_dirs="$(bash ./scripts/go-package-list.sh dirs | tr '\n' ' ')"

STEP=0
TOTAL=16
step() {
  STEP=$((STEP + 1))
  echo
  echo "==> [${STEP}/${TOTAL}] $*"
}

step "go test (unit)"
go test -count=1 ${packages}

step "go test -race"
go test -race -count=1 ${packages}

step "integration tests"
go test -tags=integration -race -count=1 ./tests/integration/...

step "go vet"
go vet ${packages}

step "gofmt"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  files="$(git ls-files '*.go')"
else
  files="$(GOFLAGS='-buildvcs=false' go list -f '{{range .GoFiles}}{{$.Dir}}/{{.}} {{end}}{{range .TestGoFiles}}{{$.Dir}}/{{.}} {{end}}{{range .XTestGoFiles}}{{$.Dir}}/{{.}} {{end}}' ./...)"
fi
existing_files=()
for file in ${files}; do
  if [[ -f "${file}" ]]; then
    existing_files+=("${file}")
  fi
done
if [[ ${#existing_files[@]} -eq 0 ]]; then
  echo "No Go files found for gofmt check."
else
  unformatted="$(gofmt -l "${existing_files[@]}")"
  if [[ -n "${unformatted}" ]]; then
    echo "Found unformatted files:"
    echo "${unformatted}"
    exit 1
  fi
fi

step "staticcheck"
if [[ ! -x "$(go env GOPATH)/bin/staticcheck" ]]; then
  go install honnef.co/go/tools/cmd/staticcheck@latest
fi
"$(go env GOPATH)/bin/staticcheck" ${packages}

step "errcheck"
if [[ ! -x "$(go env GOPATH)/bin/errcheck" ]]; then
  go install github.com/kisielk/errcheck@latest
fi
"$(go env GOPATH)/bin/errcheck" \
  -ignoretests \
  -exclude scripts/errcheck-excludes.txt \
  ${packages}

step "govulncheck"
if [[ ! -x "$(go env GOPATH)/bin/govulncheck" ]]; then
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi
"$(go env GOPATH)/bin/govulncheck" ${packages}

step "gosec (high)"
if [[ ! -x "$(go env GOPATH)/bin/gosec" ]]; then
  go install github.com/securego/gosec/v2/cmd/gosec@latest
fi
"$(go env GOPATH)/bin/gosec" -severity high ${package_dirs}

step "coverage gate (total + per-package floors)"
bash ./scripts/coverage-gate.sh

step "fuzz quick (seed corpus sweep)"
bash ./scripts/fuzz-quick.sh

step "file-size (ADR-0010)"
bash ./scripts/file-size-lint.sh

step "frontend lint"
if [[ ! -x "$(command -v bun)" ]]; then
  echo "ERROR: bun is required for landing build checks."
  exit 1
fi
(cd landing && bun install --frozen-lockfile && bun run lint)

step "frontend typecheck"
(cd landing && bun run typecheck)

step "frontend test"
(cd landing && bun run test)

step "frontend build"
(cd landing && bun run build)

if [[ "${MXKEYS_LIVE_TEST:-0}" == "1" ]]; then
  if [[ -z "${MXKEYS_LIVE_BASE_URL:-}" ]]; then
    echo "ERROR: MXKEYS_LIVE_BASE_URL must be set when MXKEYS_LIVE_TEST=1 (no hardcoded fallback)."
    exit 1
  fi
  step "live interop against ${MXKEYS_LIVE_BASE_URL}"
  MXKEYS_LIVE_TEST=1 MXKEYS_LIVE_BASE_URL="${MXKEYS_LIVE_BASE_URL}" \
    go test -count=1 ./internal/server -run 'TestLive(FederationQueryStrictness|QueryCompatibility|NotaryFailurePath)' -v
else
  echo
  echo "Live interop skipped (set MXKEYS_LIVE_TEST=1 and MXKEYS_LIVE_BASE_URL to enable)."
fi

echo
echo "CI parity preflight passed."
