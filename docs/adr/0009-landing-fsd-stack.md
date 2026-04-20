Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# ADR-0009: Landing Stack and Architecture

## Status

Accepted.

## Context

The `landing/` site started as a flat `src/components/` tree with
a thin `App.tsx`, 22 eager-loaded i18n bundles, a hard-coded
`https://mxkeys.org` site URL, and no state-management or routing
libraries. The organization stack prescribes React 19, Tailwind,
Zustand, TanStack Router/Query, Zod, CVA, Storybook, Playwright,
MSW, Sentry, and Feature-Sliced Design.

Site-specific constraints:

- The site is mostly static: one route, no session, no API calls
  in the shipping build.
- Operator forks need a runtime-configurable site URL.
- Eager i18n inflated first paint.

## Decision

Adopt Feature-Sliced Design with this layout:

```text
src/app       App, providers (Query, Sentry, ErrorBoundary), Router
src/pages     <page>/ui + <page>/index.ts barrel
src/widgets   one folder per widget (header, hero, about, ...)
              each with ui/<name>.tsx + barrel
src/features  narrow interactive pieces (mobile-nav Zustand store,
              notary-lookup form)
src/entities  reserved
src/shared    ui kit, config, i18n, api mocks
```

Layer direction enforced by `eslint-plugin-boundaries`:

```text
app > pages > widgets > features > entities > shared
```

Stack members in active use:

| Library | Usage |
|---|---|
| `@tanstack/react-router` | Root route + index route |
| `@tanstack/react-query` | Provider configured at the app boundary |
| `zustand` | `useMobileNav` store |
| `zod` | `envSchema` validates `VITE_SITE_URL`, `VITE_SENTRY_DSN`, `VITE_ENVIRONMENT` at module load; form schemas in `features/*/model` |
| `react-hook-form` + `@hookform/resolvers/zod` | `features/notary-lookup` form |
| `@sentry/react` | Conditional init and `AppErrorBoundary` |
| `i18next-resources-to-backend` | Lazy per-locale JSON via `import()` |
| `msw` | Node handler setup in `src/shared/api/mocks/server.ts` used by Vitest; browser handler wired in `browser.ts` for local dev |
| `storybook` (10.x) | `.storybook/main.ts` + `preview.ts`; stories for `Logo` and `NotaryLookupForm` |
| `@playwright/test` | Smoke e2e (home render, RTL toggle, mobile nav) |
| `eslint-plugin-boundaries` | FSD layer enforcement |

Shared UI primitives are added per consumer, not scaffolded ahead
of demand. Current contents of `src/shared/ui`:

- `Logo`: consumed by widgets.
- `TextField`: consumed by `NotaryLookupForm`.

An earlier scaffold (`Button`, `Container`, `ExternalLink`) was
removed because no widget imported it and the variants did not
match real widget needs. The next primitive is introduced
alongside the first real consumer.

Environment-driven site URL:

- `VITE_SITE_URL` (default `https://notary.example.org`) is
  validated by the Zod schema at module load and substituted into
  `index.html`, `robots.txt`, `sitemap.xml` via the
  `htmlEnvReplace` Vite plugin on `__MXKEYS_SITE_URL__` and
  `__MXKEYS_ENVIRONMENT__` placeholders.

Bundle split via `manualChunks`: Sentry, TanStack (Router +
Query), and i18next ship in separate chunks that cache
independently. Main bundle is around 273 KB (gzip ~82 KB).

## Consequences

- Widgets, features, and shared are testable in isolation.
- Operators fork the landing by setting `VITE_SITE_URL` and
  `VITE_SENTRY_DSN` in their environment.
- Lazy i18n cuts the non-active locales (21 of 22) out of the
  first-paint payload.
- The e2e suite catches "the page renders at all" regressions
  that lint and unit cannot.

## Alternatives Considered

- Flat component layout. Rejected: organization rule requires
  FSD.
- Static-site generator (Astro, Next.js static export).
  Rejected for this pass. The Vite + React pipeline already
  covers the static-marketing requirement.
- Scaffold all declared stack members before the first consumer
  exists. Rejected for `Button` / `Container` / `ExternalLink`
  when that experiment produced dead code.

## References

- `landing/src/`
- `landing/.storybook/`
- `landing/eslint.config.mjs`
- `landing/vite.config.ts`
- `landing/playwright.config.ts`
