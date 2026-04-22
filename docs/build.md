Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Building MXKeys

## Scope

This document covers:

- local development builds,
- reproducible production builds,
- verification commands used by CI parity,
- frontend build/test commands for `landing/`,
- extended CI-gate commands (coverage, staticcheck, errcheck, fuzz).

For deployment and operations, see `docs/deployment.md`.

## Requirements

- Go 1.26+
- PostgreSQL 14+ for runtime
- Bun 1.3+ for `landing/`

## Required Configuration

- `database.url` must be set explicitly.
- `cluster.shared_secret` is required when cluster mode is enabled (>=32 chars, placeholder values rejected at startup).
- `cluster.raft_state_dir` is required when `cluster.consensus_mode=raft`.
- `security.admin_access_token` is required only when the admin-only operational routes are intended to be exposed.

## Build Commands

Development build:

```bash
go build -o mxkeys ./cmd/mxkeys
```

Production build:

```bash
CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags="-s -w" \
  -buildvcs=false \
  -o mxkeys \
  ./cmd/mxkeys
```

Landing build:

```bash
cd landing
bun install --frozen-lockfile
bun run build
```

## Reproducible Build

```bash
export SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct)

CGO_ENABLED=0 go build \
  -trimpath \
  -ldflags="-s -w -buildid=" \
  -o mxkeys \
  ./cmd/mxkeys

sha256sum mxkeys
```

## Verification

Backend checks:

```bash
packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"
package_dirs="$(bash ./scripts/go-package-list.sh dirs | tr '\n' ' ')"

go test -count=1 ${packages}
go test -count=1 ./internal/server ./internal/keys ./internal/cluster \
    ./internal/zero/canonical ./internal/zero/merkle ./internal/zero/raft
go test -tags=integration -race ./tests/integration/...
go test -race -count=1 ${packages}
go vet ${packages}
```

Frontend checks:

```bash
cd landing
bun run lint
bun run test
bun run typecheck
bun run build
```

Full parity with CI (runs every gate):

```bash
./scripts/ci-parity-preflight.sh
```

## Extended CI Gates

Coverage gate (thresholds tracked in the script; per-package floors +
total floor):

```bash
./scripts/coverage-gate.sh
```

Static analysis (linters beyond `go vet`):

```bash
packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"

go install honnef.co/go/tools/cmd/staticcheck@latest
"$(go env GOPATH)/bin/staticcheck" ${packages}

go install github.com/kisielk/errcheck@latest
"$(go env GOPATH)/bin/errcheck" -ignoretests \
    -exclude=./scripts/errcheck-excludes.txt ${packages}
```

Fuzz pass (30s per target by default, tunable via `FUZZTIME`):

```bash
./scripts/fuzz-quick.sh
FUZZTIME=5m ./scripts/fuzz-quick.sh   # local deep pass
```

Targeted long fuzz for a single parser:

```bash
go test -run=^$ -fuzz=^FuzzJSON$ -fuzztime=10m ./internal/zero/canonical
```

## Security Scanning

```bash
packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"
package_dirs="$(bash ./scripts/go-package-list.sh dirs | tr '\n' ' ')"

go install golang.org/x/vuln/cmd/govulncheck@latest
"$(go env GOPATH)/bin/govulncheck" ${packages}

go install github.com/securego/gosec/v2/cmd/gosec@latest
"$(go env GOPATH)/bin/gosec" -severity high ${package_dirs}
```

The `GOTOOLCHAIN` pin that previous docs mentioned was a temporary
workaround when the module targeted Go 1.22; it has been removed now
that `go.mod` tracks 1.26 directly.

## Landing E2E

```bash
cd landing
bun run e2e:install           # chromium for the first run
bun run e2e                   # runs playwright.config.ts + e2e/*.spec.ts
```

Tests spin up `vite preview` on 127.0.0.1:4173 by default. Override with
`E2E_BASE_URL` to run against a deployed host.

## SBOM

Example CycloneDX generation:

```bash
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
cyclonedx-gomod mod -json -output sbom.json
```

Release traceability expectations are defined in `docs/release-process.md`.
