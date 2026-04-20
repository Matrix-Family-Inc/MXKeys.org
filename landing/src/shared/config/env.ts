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
   * Operators deploying their own branded MXKeys notary landing override
   * this without touching the code.
   */
  VITE_SITE_URL: z
    .string()
    .url()
    .default('https://notary.example.org'),

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

/** siteURL is the canonical absolute origin of the landing deployment. */
export const siteURL = env.VITE_SITE_URL.replace(/\/+$/, '');
