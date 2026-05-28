Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Tue Apr 22 2026 UTC
Status: Updated

# Checksums

Release artifact checksums are published in GitHub Releases.

## Verification

Download the release and verify:

```bash
sha256sum mxkeys-linux-amd64
```

Compare against the checksum published in the release notes.

## Reproducible Build

To reproduce the exact binary:

```bash
export SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct)
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o mxkeys ./cmd/mxkeys
sha256sum mxkeys
```

See `docs/build.md` for full reproducibility instructions.
