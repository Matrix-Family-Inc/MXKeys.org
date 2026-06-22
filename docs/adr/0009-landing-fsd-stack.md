Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# ADR-0009: Landing Stack and Architecture

## Status

Accepted.

## Visibility

Public.

## Context

The `landing/` site is mostly static, but it still needs enforceable
frontend boundaries, lazy i18n, a runtime-configurable site URL, and
space for small interactive features such as mobile navigation and
notary lookup.

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

Additional landing rules:

- use TanStack Router at the app boundary;
- keep UI state in the `mobile-nav` Zustand store;
- validate `VITE_SITE_URL`, `VITE_SENTRY_DSN`, and
  `VITE_ENVIRONMENT` with Zod at module load;
- lazy-load i18n resources by locale;
- add `shared/ui` primitives only with a real consumer.

`VITE_SITE_URL` defaults to `https://notary.example.org` and is
substituted into static assets through the Vite HTML replacement
plugin.

## Consequences

- Widgets, features, and shared are testable in isolation.
- Operators fork the landing by setting `VITE_SITE_URL` and
  `VITE_SENTRY_DSN` in their environment.
- Lazy i18n keeps non-active locales out of the first-paint payload.
- The e2e suite catches "the page renders at all" regressions
  that lint and unit cannot.

## Alternatives Considered

- Flat component layout. Rejected: organization rule requires
  FSD.
- Static-site generator (Astro, Next.js static export).
  Rejected for this pass. The Vite + React pipeline already
  covers the static-marketing requirement.
- Scaffold all declared stack members before the first consumer
  exists. Rejected because unused primitives produced dead code.

## References

- `landing/src/` - MXKeys landing implementation governed by this ADR.
- `landing/eslint.config.mjs` - FSD boundary enforcement.
- `landing/vite.config.ts` - routing, environment replacement, and bundle split
  configuration.
