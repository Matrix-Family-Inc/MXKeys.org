Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Tue Apr 22 2026 UTC
Status: Updated

# Release Evidence

This directory stores reproducible release evidence for tagged releases.

## Files

- `checksums.md` — SHA256 checksums for release artifacts
- `sbom.md` — software bill of materials (dependency inventory)

## Generation

Checksums are generated during the release build:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o mxkeys ./cmd/mxkeys
sha256sum mxkeys
```

SBOM is generated via:

```bash
go list -m all
```

For CycloneDX format see `docs/build.md`.
