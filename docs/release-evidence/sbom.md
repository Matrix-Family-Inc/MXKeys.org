Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# SBOM (Software Bill of Materials)

## Go Modules

Source: `go.mod`

| Module | Version |
|--------|---------|
| `github.com/lib/pq` | v1.10.9 |
| `github.com/likexian/whois` | v1.15.7 |
| `golang.org/x/sync` | v0.10.0 |
| `golang.org/x/time` | v0.9.0 |
| `golang.org/x/net` | v0.48.0 (indirect) |

## Landing Dependencies

Source: `landing/package.json`

### Runtime

| Package | Purpose |
|---------|---------|
| `react` | UI framework |
| `@tanstack/react-router` | Routing |
| `@tanstack/react-query` | Server state |
| `zustand` | Client state |
| `zod` | Validation |
| `react-hook-form` | Form state |
| `i18next` | Internationalization |
| `@sentry/react` | Error tracking |
| `tailwind-merge` | CSS utilities |
| `class-variance-authority` | Component variants |

### Build

| Package | Purpose |
|---------|---------|
| `vite` | Build tool |
| `typescript` | Type checking |
| `tailwindcss` | Styling |
| `vitest` | Testing |
| `playwright` | E2E testing |
| `storybook` | Component development |
| `msw` | API mocking |

## Machine-Readable Export

```bash
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
cyclonedx-gomod mod -json -output sbom.json
```
