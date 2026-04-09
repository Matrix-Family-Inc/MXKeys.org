Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Maintainer: Brabus
Role: Lead Architect
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Wed Apr 09 2026 UTC
Status: Created

# Release Process and Evidence

## Scope

This document defines the minimum release traceability expected for MXKeys.
Release evidence is produced by CI and release workflows. It is not maintained as versioned markdown snapshots inside `docs/`.

## Required Evidence

Every release candidate should have:

- commit SHA and tag,
- full CI parity result,
- checksums for shipped artifacts,
- machine-readable SBOM,
- release notes,
- security scan output,
- live interoperability evidence when the release policy requires it.

## Recommended Artifact Locations

- workflow artifacts in GitHub Actions,
- release attachments in GitHub Releases,
- immutable external artifact storage if your release policy requires it.

## Minimum Commands

Build and verification:

```bash
./scripts/ci-parity-preflight.sh
```

SBOM generation example:

```bash
cyclonedx-gomod mod -json -output sbom.json
```

Checksums example:

```bash
sha256sum mxkeys
```

## Release Notes

Release notes should describe:

- user-visible changes,
- contract-impacting changes,
- known operational constraints,
- required migration or rollout notes.

## Live Interop Evidence

When live interop is part of the release policy, keep:

- workflow URL,
- target base URL,
- UTC execution timestamp,
- relevant request/response evidence or logs.

## References

- Required check names in branch protection must stay aligned with `.github/workflows/pr-gate.yml`, `.github/workflows/release-live-interop-gate.yml`, and `scripts/verify-github-branch-protection.sh`.
- `docs/build.md`
- `.github/workflows/pr-gate.yml`
- `.github/workflows/release-live-interop-gate.yml`
