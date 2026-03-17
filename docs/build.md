# Building MXKeys

## Requirements

- Go 1.22+
- PostgreSQL 14+ (for running)

## Development Build

```bash
go build -o mxkeys ./cmd/mxkeys
```

## Production Build

```bash
CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -buildvcs=false \
    -o mxkeys \
    ./cmd/mxkeys
```

Flags:
- `CGO_ENABLED=0` — Static binary, no C dependencies
- `-trimpath` — Remove build paths for reproducibility
- `-ldflags="-s -w"` — Strip debug info, reduce size

## Cross Compilation

### Linux AMD64
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -buildvcs=false \
    -o mxkeys-linux-amd64 ./cmd/mxkeys
```

### Linux ARM64
```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -buildvcs=false \
    -o mxkeys-linux-arm64 ./cmd/mxkeys
```

### macOS AMD64
```bash
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -buildvcs=false \
    -o mxkeys-darwin-amd64 ./cmd/mxkeys
```

### macOS ARM64 (Apple Silicon)
```bash
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -buildvcs=false \
    -o mxkeys-darwin-arm64 ./cmd/mxkeys
```

## Reproducible Builds

For verifiable builds:

```bash
# Set build timestamp
export SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct)

# Build with deterministic flags
CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -buildid=" \
    -o mxkeys \
    ./cmd/mxkeys

# Verify
sha256sum mxkeys
```

## Docker Build

```bash
docker build -t mxkeys:latest .
```

Multi-platform:
```bash
docker buildx build \
    --platform linux/amd64,linux/arm64 \
    -t ghcr.io/matrixfamily/mxkeys:latest \
    --push .
```

## Binary Size

Typical sizes (stripped):

| Platform | Size |
|----------|------|
| Linux AMD64 | ~7.5 MB |
| Linux ARM64 | ~7.3 MB |
| macOS AMD64 | ~7.8 MB |
| macOS ARM64 | ~7.6 MB |

## Dependencies

External dependencies (minimal):

```
github.com/lib/pq v1.10.9         # PostgreSQL driver
golang.org/x/sync v0.10.0         # Singleflight
golang.org/x/time v0.9.0          # Rate limiter
```

All other functionality implemented in `internal/zero/` packages.

## Testing

```bash
# Unit tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

## Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Security Scanning

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Scan for vulnerabilities
govulncheck ./...
```

## SBOM Generation

```bash
# Install cyclonedx-gomod
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest

# Generate SBOM
cyclonedx-gomod mod -json -output sbom.json
```
