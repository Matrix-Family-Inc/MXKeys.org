/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import * as Sentry from '@sentry/react';

import { env } from '../../shared/config/env';

/**
 * initSentry wires @sentry/react to the configured DSN. When no DSN is
 * set the function is a no-op so the landing still runs on operator forks
 * that opt out of telemetry.
 *
 * Idempotent: safe to call multiple times (hot reload, tests). Subsequent
 * calls short-circuit on the Sentry client already being configured.
 */
export function initSentry() {
  const dsn = env.VITE_SENTRY_DSN;
  if (!dsn) return;
  if (Sentry.getClient()) return;

  Sentry.init({
    dsn,
    environment: env.VITE_ENVIRONMENT,
    // Keep the default sample rate low by default: bursty traffic on a
    // marketing page should not saturate the Sentry quota. Operators
    // override via VITE_SENTRY_SAMPLE_RATE upstream if they need more.
    tracesSampleRate: 0.1,
    replaysSessionSampleRate: 0,
    replaysOnErrorSampleRate: 1,
  });
}
