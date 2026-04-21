#!/usr/bin/env bash
# Project: MXKeys
# Company: Matrix Family Inc. (https://matrix.family)
# Maintainer: Brabus
# Contact: dev@matrix.family
# Date: Mon Apr 20 2026 UTC
# Status: Created
#
# Builds reproducible release artifacts for mxkeys + mxkeys-verify.
#
# Invariants:
#   - deterministic output: same commit produces byte-identical
#     binaries on any machine that has the same Go toolchain version.
#   - CGO disabled so binaries are pure static.
#   - GOFLAGS are pinned so no external environment can change what we
#     ship.
#   - ldflags -s -w strip the symbol table and DWARF info; that is the
#     conventional release posture and makes the binary smaller.
#   - -trimpath removes local filesystem prefixes so different build
#     hosts still produce identical bytes.
#   - Per-artifact checksum + SBOM (cyclonedx JSON) written alongside
#     the binary.
#
# Produces, for each (goos, goarch) target:
#   dist/mxkeys-${version}-${goos}-${goarch}
#   dist/mxkeys-${version}-${goos}-${goarch}.sha256
#   dist/mxkeys-${version}-${goos}-${goarch}.sbom.json
#
# And a consolidated dist/SHA256SUMS file.

set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
TARGETS="${TARGETS:-linux/amd64 linux/arm64}"
DIST="${DIST:-dist}"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git log -1 --pretty=%ct 2>/dev/null || echo 0)}"
export SOURCE_DATE_EPOCH

mkdir -p "${DIST}"
rm -f "${DIST}/SHA256SUMS"

have_cyclonedx=1
if ! command -v cyclonedx-gomod >/dev/null 2>&1; then
  have_cyclonedx=0
  echo "NOTE: cyclonedx-gomod not on PATH; SBOMs will be skipped."
  echo "      install with: go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"
fi

for target in ${TARGETS}; do
  goos="${target%/*}"
  goarch="${target#*/}"
  for bin in mxkeys mxkeys-verify mxkeys-walctl; do
    out="${DIST}/${bin}-${VERSION}-${goos}-${goarch}"
    echo "==> ${out}"
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" \
      go build \
        -trimpath \
        -ldflags="-s -w -X mxkeys/internal/version.Version=${VERSION}" \
        -buildvcs=false \
        -o "${out}" \
        "./cmd/${bin}"
    (cd "${DIST}" && sha256sum "$(basename "${out}")" >> SHA256SUMS)
    sha256sum "${out}" | awk '{print $1}' > "${out}.sha256"

    if [[ "${have_cyclonedx}" -eq 1 ]]; then
      cyclonedx-gomod mod -json -output-version 1.5 -output "${out}.sbom.json" . >/dev/null
    fi
  done
done

echo
echo "Done. Release artifacts:"
ls -la "${DIST}" | sed 's/^/  /'
echo
echo "Consolidated hashes: ${DIST}/SHA256SUMS"
