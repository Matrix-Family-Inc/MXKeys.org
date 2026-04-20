Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# ADR-0009: Landing Stack and Architecture

## Status

Accepted

## Context

The `landing/` marketing site began as a flat `src/components/` tree
with a thin `App.tsx`, eager-loaded 22 i18n bundles, a hard-coded
`https://mxkeys.org` site URL, and no state-management or routing
libraries. The declared operator stack asked for a richer setup
(React 19 + Tailwind + Zustand + TanStack Router/Query + Zod +
CVA + Storybook + Playwright + MSW + Sentry + FSD).

Landing-specific considerations:

- The site is effectively static: no forms, no API calls today, no
  session state, no multi-route navigation. Declared stack members
  designed for rich apps (React Hook Form, MSW, Storybook, TanStack
  Query) have no immediate use case.
- Operators forking the landing for their own deployment need the
  site URL to be configurable without patching HTML.
- The first-paint bundle was bloated by eager i18n imports.

## Decision

Adopt Feature-Sliced Design with the following slice layout:

```text
src/app       App + providers (Query, Sentry, ErrorBoundary) + Router
src/pages     <page>/ui + <page>/index.ts barrel
src/widgets   one folder per widget (header, hero, about, ...)
              each with ui/<name>.tsx + barrel
src/features  narrow interactive pieces (mobile-nav Zustand store)
src/entities  reserved (currently unused)
src/shared    ui kit, config, lib, i18n, api mocks
```

Import direction enforced by `eslint-plugin-boundaries`:

```text
app > pages > widgets > features > entities > shared
```

Adopted stack members with an immediate use case:

| Library | Usage |
|---|---|
| `@tanstack/react-router` | Root route + index route; absorbs future /docs, /status |
| `@tanstack/react-query` | Provider wired with conservative defaults; first consumer will be the live-query demo widget |
| `zustand` | `useMobileNav` store: single boolean + two mutators |
| `zod` | `envSchema` validates `VITE_SITE_URL`, `VITE_SENTRY_DSN`, `VITE_ENVIRONMENT` at module load |
| `class-variance-authority` + `clsx` + `tailwind-merge` | Shared UI kit variants (Button, Container, ExternalLink, Logo) |
| `@sentry/react` | Conditional init + `AppErrorBoundary` |
| `i18next-resources-to-backend` | Lazy-load per-locale JSON via `import()` |
| `@playwright/test` | Smoke e2e (home render, RTL toggle, mobile nav) |
| `eslint-plugin-boundaries` | FSD layer enforcement |

Adopted stack members intentionally deferred:

| Library | Deferral rationale |
|---|---|
| Storybook | ~200 LoC config for ~8 components without an active design-review audience. Revisit when variant iteration becomes a workflow bottleneck. |
| MSW | No client-facing API calls today. Revisit alongside the planned live-query demo widget. |
| React Hook Form | No forms. |
| `@sentry/vite-plugin` source-map upload | Installed but not wired to a release workflow; requires operator-scoped `SENTRY_AUTH_TOKEN`. |

Environment-driven site URL:

- `VITE_SITE_URL` (default `https://notary.example.org`) is validated
  by the Zod schema at load and applied to canonical meta tags,
  Open Graph, robots.txt, sitemap.xml via a custom Vite plugin
  (`htmlEnvReplace`) that substitutes `__MXKEYS_SITE_URL__` and
  `__MXKEYS_ENVIRONMENT__` placeholders during build/serve.

Bundle split via `manualChunks`: the main bundle drops from ~552 KB
to ~301 KB; TanStack Router + Query, Sentry, and i18next land in
dedicated chunks that cache independently.

## Consequences

- Widgets, features, and shared are independently testable and
  reorganizable without cross-cutting fallout.
- Operators deploying a branded fork set `VITE_SITE_URL` and
  `VITE_SENTRY_DSN` in their environment; no file edits required.
- Lazy i18n cuts initial JS payload by ~90% for the 21 non-active
  locales.
- Storybook / MSW / RHF debts are explicit and time-limited rather
  than quietly ignored.
- The e2e suite gives CI a smoke net for the "the page renders at
  all" regression that lint + unit cannot catch.

## Alternatives Considered

- Stay with the flat component layout: rejected; operator stack
  rule mandates FSD, and the current size already benefits.
- Rewrite as a static site generator (Astro, Next.js static export):
  rejected as out-of-scope for this pass; the React + Vite pipeline
  already meets the static-marketing requirement.
- Ship the full declared stack (Storybook, MSW, RHF) immediately:
  rejected to avoid carrying config dead-weight; documented as
  deferrals with clear re-entry criteria.

## References

- `landing/src/`
- `landing/eslint.config.mjs`
- `landing/vite.config.ts`
- `landing/playwright.config.ts`
