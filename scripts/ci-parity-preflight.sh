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

packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"
package_dirs="$(bash ./scripts/go-package-list.sh dirs | tr '\n' ' ')"

echo "[1/11] go test"
go test -count=1 ${packages}

echo "[2/11] integration tests"
go test -tags=integration -race -count=1 ./tests/integration/...

echo "[3/11] race tests"
go test -race -count=1 ${packages}

echo "[4/11] go vet"
go vet ${packages}

echo "[5/11] gofmt check"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  files="$(git ls-files '*.go')"
else
  files="$(GOFLAGS='-buildvcs=false' go list -f '{{range .GoFiles}}{{$.Dir}}/{{.}} {{end}}{{range .TestGoFiles}}{{$.Dir}}/{{.}} {{end}}{{range .XTestGoFiles}}{{$.Dir}}/{{.}} {{end}}' ./...)"
fi

existing_files=()
unformatted=""
for file in ${files}; do
  if [[ -f "${file}" ]]; then
    existing_files+=("${file}")
  fi
done

if [[ ${#existing_files[@]} -eq 0 ]]; then
  echo "No Go files found for gofmt check."
else
  unformatted="$(gofmt -l "${existing_files[@]}")"
fi

if [[ -n "${unformatted}" ]]; then
  echo "Found unformatted files:"
  echo "${unformatted}"
  exit 1
fi

echo "[6/11] govulncheck"
if [[ ! -x "$(go env GOPATH)/bin/govulncheck" ]]; then
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi
GOTOOLCHAIN=go1.26.2 "$(go env GOPATH)/bin/govulncheck" ${packages}

echo "[7/11] gosec (high)"
if [[ ! -x "$(go env GOPATH)/bin/gosec" ]]; then
  go install github.com/securego/gosec/v2/cmd/gosec@latest
fi
"$(go env GOPATH)/bin/gosec" -severity high ${package_dirs}

echo "[8/11] frontend lint"
if [[ ! -x "$(command -v bun)" ]]; then
  echo "ERROR: bun is required for landing build checks."
  exit 1
fi
(
  cd landing
  bun install --frozen-lockfile
  bun run lint
)

echo "[9/11] frontend test"
(
  cd landing
  bun run test
)

echo "[10/11] frontend build"
(
  cd landing
  bun run build
)

if [[ "${MXKEYS_LIVE_TEST:-0}" == "1" ]]; then
  base_url="${MXKEYS_LIVE_BASE_URL:-https://mxkeys.org}"
  echo "[11/11] live interop against ${base_url}"
  MXKEYS_LIVE_TEST=1 MXKEYS_LIVE_BASE_URL="${base_url}" \
    go test -count=1 ./internal/server -run 'TestLive(FederationQueryStrictness|QueryCompatibility|NotaryFailurePath)' -v
else
  echo "[11/11] live interop skipped (set MXKEYS_LIVE_TEST=1 to enable)"
fi

echo "CI parity preflight passed."
