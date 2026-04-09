# Building MXKeys

## Scope

This document covers:

- local development builds,
- reproducible production builds,
- verification commands used by CI parity,
- frontend build/test commands for `landing/`.

For deployment and operations, see `docs/deployment.md`.

## Requirements

- Go 1.22+
- PostgreSQL 14+ for runtime
- Bun 1.x for `landing/`

## Required Configuration

- `database.url` must be set explicitly.
- `cluster.shared_secret` is required when cluster mode is enabled.
- `security.enterprise_access_token` is required only when protected operational routes are intended to be exposed.

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
go test -count=1 ./internal/server ./internal/keys ./internal/cluster ./internal/zero/canonical ./internal/zero/merkle ./internal/zero/raft
go test -tags=integration ./tests/integration/...
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

Full parity with CI:

```bash
./scripts/ci-parity-preflight.sh
```

## Security Scanning

```bash
packages="$(bash ./scripts/go-package-list.sh | tr '\n' ' ')"
package_dirs="$(bash ./scripts/go-package-list.sh dirs | tr '\n' ' ')"

go install golang.org/x/vuln/cmd/govulncheck@latest
GOTOOLCHAIN=go1.26.2 govulncheck ${packages}

go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec -severity high ${package_dirs}
```

`GOTOOLCHAIN=go1.26.2` is used only for `govulncheck`, mirroring CI exactly. The project module remains pinned by `go.mod`; the patched toolchain is an isolated scanner requirement rather than the general local build requirement.

## SBOM

Example CycloneDX generation:

```bash
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
cyclonedx-gomod mod -json -output sbom.json
```

Release traceability expectations are defined in `docs/release-process.md`.
