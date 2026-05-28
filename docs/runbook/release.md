Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Runbook: Release

Procedure for building, signing, and publishing a release.
Executed by a human release manager with access to the repository
and the signing key.

## Release Properties

- Reproducible: same commit plus same Go toolchain version produces
  byte-identical binaries. Operators can rebuild locally and
  cross-check sha256 against the published `SHA256SUMS`.
- Attested: every binary ships with a CycloneDX SBOM when
  `cyclonedx-gomod` is installed at build time.
- Signed: the release tag and `SHA256SUMS` are signed with the
  release manager's GPG key. Use ed25519 or RSA-4096; RSA-1024 is
  not acceptable.

## Pre-flight

Run the CI parity preflight locally before tagging. A release
that fails any gate does not ship.

    bash scripts/ci-parity-preflight.sh

This exercises every job in `.github/workflows/pr-gate.yml`: unit,
race, integration, vet, gofmt, staticcheck, errcheck, govulncheck,
gosec, coverage, fuzz-quick, file-size, frontend lint, typecheck,
test, build.

Also confirm:

- `go.mod` declares the Go version documented in the README.
- `CHANGELOG.md` has an entry for the target version matching the
  git tag to be created.
- `docs/` matches shipped behaviour (no stale runbook references,
  ADR numbering contiguous).

## Build

At the commit to ship:

    VERSION=v1.0.0 TARGETS="linux/amd64 linux/arm64" \
      bash scripts/build-release.sh

Output under `dist/`:

    mxkeys-${version}-${os}-${arch}
    mxkeys-${version}-${os}-${arch}.sha256
    mxkeys-${version}-${os}-${arch}.sbom.json   (when cyclonedx-gomod is present)
    mxkeys-verify-${version}-${os}-${arch}
    mxkeys-verify-${version}-${os}-${arch}.sha256
    mxkeys-verify-${version}-${os}-${arch}.sbom.json
    SHA256SUMS

Build flags set by the script:

- `CGO_ENABLED=0`: static binary.
- `-trimpath`: strip local filesystem prefix.
- `-ldflags "-s -w -X mxkeys/internal/version.Version=${VERSION}"`:
  drop symbol table and DWARF, embed the version string.
- `SOURCE_DATE_EPOCH` from `git log -1`: reproducible timestamp.

Reproducibility cross-check:

    # On a second host with the same Go toolchain version:
    git checkout v1.0.0
    VERSION=v1.0.0 bash scripts/build-release.sh
    sha256sum --check dist/SHA256SUMS

Any mismatch is a release blocker and must be investigated before
publishing.

## Sign

Sign the checksum file. Upload the detached signature alongside
the binaries.

    gpg --armor --detach-sign dist/SHA256SUMS
    # produces dist/SHA256SUMS.asc

Tag the release commit with a signed annotated tag:

    git tag -s v1.0.0 -m "mxkeys v1.0.0"
    git push origin v1.0.0

Operator-side verification:

    gpg --verify SHA256SUMS.asc SHA256SUMS
    sha256sum --check SHA256SUMS

## Publish

Upload the binaries, `.sha256` files, SBOM JSONs, `SHA256SUMS`,
and `SHA256SUMS.asc` to the GitHub release page (or equivalent
channel). Release notes include:

- Summary of user-visible changes, copied from `CHANGELOG.md`.
- Security-relevant notes: fixed CVEs, advisory links.
- Upgrade instructions when config or schema shape changed.
- Pointer to `SECURITY.md`.

## Supply-Chain Transparency

Every release carries:

- CycloneDX SBOM JSON files from the build.
- Link to the exact git commit.
- Link to the PR-gate CI run that passed on that commit.

Operators who need SLSA-level provenance can rebuild from source
and cross-check sha256. Reproducibility is what makes this check
meaningful.

## Post-Release

- Bump `internal/version.Version` on `main` to the next
  development value when any in-process consumer reads it. The
  `-X` flag is the source of truth for shipped binaries, so the
  in-source default is informational.
- Update external documentation referencing the previous version.
- Close the tracking issue for the release.
