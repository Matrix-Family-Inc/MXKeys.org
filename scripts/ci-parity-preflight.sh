#!/usr/bin/env bash
# Project: MXKeys - Matrix Federation Trust Infrastructure
# Company: Matrix.Family Inc. - Delaware C-Corp
# Dev: Brabus
# Date: Mon Mar 16 2026 UTC
# Status: Created
# Contact: @support:matrix.family

set -euo pipefail

echo "[1/8] go test"
go test -count=1 ./...

echo "[2/8] integration tests"
go test -tags=integration -count=1 ./tests/integration/...

echo "[3/8] race tests"
go test -race -count=1 ./...

echo "[4/8] go vet"
go vet ./...

echo "[5/8] gofmt check"
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  files="$(git ls-files '*.go')"
else
  files="$(GOFLAGS='-buildvcs=false' go list -f '{{range .GoFiles}}{{$.Dir}}/{{.}} {{end}}{{range .TestGoFiles}}{{$.Dir}}/{{.}} {{end}}{{range .XTestGoFiles}}{{$.Dir}}/{{.}} {{end}}' ./...)"
fi
unformatted="$(gofmt -l ${files})"
if [[ -n "${unformatted}" ]]; then
  echo "Found unformatted files:"
  echo "${unformatted}"
  exit 1
fi

echo "[6/8] govulncheck"
if [[ ! -x "$(go env GOPATH)/bin/govulncheck" ]]; then
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi
GOTOOLCHAIN=go1.26.1 "$(go env GOPATH)/bin/govulncheck" ./...

echo "[7/8] gosec (high)"
if [[ ! -x "$(go env GOPATH)/bin/gosec" ]]; then
  go install github.com/securego/gosec/v2/cmd/gosec@latest
fi
"$(go env GOPATH)/bin/gosec" -severity high ./...

if [[ "${MXKEYS_LIVE_TEST:-0}" == "1" ]]; then
  base_url="${MXKEYS_LIVE_BASE_URL:-https://mxkeys.org}"
  echo "[8/8] live interop against ${base_url}"
  MXKEYS_LIVE_TEST=1 MXKEYS_LIVE_BASE_URL="${base_url}" \
    go test -count=1 ./internal/server -run 'TestLive(FederationQueryStrictness|QueryCompatibility|NotaryFailurePath)' -v
else
  echo "[8/8] live interop skipped (set MXKEYS_LIVE_TEST=1 to enable)"
fi

echo "CI parity preflight passed."
