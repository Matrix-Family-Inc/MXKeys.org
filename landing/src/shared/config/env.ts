/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { z } from 'zod';

/**
 * envSchema describes the runtime-visible Vite environment variables the
 * landing page consumes. Vite exposes variables prefixed with VITE_ to the
 * client bundle; any variable here must use that prefix.
 *
 * The schema runs at module load, so malformed configuration fails fast
 * with a precise error instead of producing broken behavior at runtime.
 */
const envSchema = z.object({
  /**
   * Public-facing site URL, used in canonical tags, Open Graph metadata,
   * and any sitemap/robots-style content that needs an absolute URL.
   * Operators deploying their own branded MXKeys notary landing
   * override this at build time via `VITE_SITE_URL=https://notary.example.org
   * bun run build`.
   *
   * The empty default is intentional: runtime code that needs a base
   * URL for API calls must fall back to `window.location.origin` so
   * a visitor on any host (including localhost) talks to the notary
   * their HTML was served from. See `apiBaseURL()` below.
   */
  VITE_SITE_URL: z
    .string()
    .url()
    .optional()
    .or(z.literal('')),

  /**
   * Optional Sentry DSN. When absent, Sentry is not initialized and every
   * tracing/error hook becomes a no-op.
   */
  VITE_SENTRY_DSN: z.string().url().optional().or(z.literal('')),

  /**
   * Environment label surfaced to Sentry and logs. Free-form but typical
   * values are "production", "staging", "development".
   */
  VITE_ENVIRONMENT: z.string().default('development'),
});

export type Env = z.infer<typeof envSchema>;

function loadEnv(): Env {
  const raw = {
    VITE_SITE_URL: import.meta.env.VITE_SITE_URL,
    VITE_SENTRY_DSN: import.meta.env.VITE_SENTRY_DSN,
    VITE_ENVIRONMENT: import.meta.env.VITE_ENVIRONMENT ?? import.meta.env.MODE,
  };
  const parsed = envSchema.safeParse(raw);
  if (!parsed.success) {
    // Log a structured error and fall back to defaults so the site still
    // renders; a strictly-fatal policy is inappropriate for a static
    // marketing page where a typo in VITE_SENTRY_DSN must not take the
    // whole page down.
    console.warn('[env] invalid VITE_* configuration, using defaults', parsed.error.flatten());
    return envSchema.parse({});
  }
  return parsed.data;
}

export const env = loadEnv();

/**
 * siteURL is the absolute origin used for canonical tags, Open Graph
 * metadata, and `link rel="canonical"`. Build-time `VITE_SITE_URL`
 * overrides it; otherwise a build-time fallback is kept so tooling
 * that only inspects the build (e.g. sitemap generators) still has a
 * reasonable value. Runtime API calls MUST NOT use siteURL - they
 * use apiBaseURL() which prefers the live browser origin.
 */
export const siteURL = (env.VITE_SITE_URL ?? 'https://mxkeys.org').replace(/\/+$/, '');

/**
 * apiBaseURL returns the origin the landing must talk to for
 * /_matrix/* and /_mxkeys/* calls. On any real visitor request it
 * is the origin the HTML was served from, so the same landing
 * bundle works on mxkeys.org, a branded operator clone, and any
 * localhost dev server without a build-time environment variable.
 * A build-time `VITE_SITE_URL` still wins so operators that proxy
 * the API through a different hostname than the landing can
 * opt in.
 */
export function apiBaseURL(): string {
  const explicit = env.VITE_SITE_URL;
  if (explicit && explicit.length > 0) return explicit.replace(/\/+$/, '');
  if (typeof window !== 'undefined' && window.location && window.location.origin) {
    return window.location.origin.replace(/\/+$/, '');
  }
  return siteURL;
}
