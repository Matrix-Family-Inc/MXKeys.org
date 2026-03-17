Project: MXKeys
Company: Matrix.Family Inc. - Delaware C-Corp
Dev: Brabus
Date: Mon Mar 16 2026 UTC
Status: Created
Contact: @support:matrix.family

# SBOM (Dependency Inventory)

## Go Modules

Source: `go list -m all`

- `mxkeys`
- `github.com/lib/pq v1.10.9`
- `golang.org/x/sync v0.10.0`
- `golang.org/x/time v0.9.0`

## Landing Dependencies

Source: `landing/package.json`

Runtime:

- `i18next`
- `lucide-react`
- `react`
- `react-dom`
- `react-i18next`

Build/dev:

- `@tailwindcss/postcss`
- `@types/react`
- `@types/react-dom`
- `@vitejs/plugin-react`
- `autoprefixer`
- `postcss`
- `tailwindcss`
- `typescript`
- `vite`

## Notes

This SBOM is maintained as a deterministic dependency inventory in markdown.
If CycloneDX/SPDX export is required for release tooling, generate an additional machine-readable SBOM artifact in CI.
