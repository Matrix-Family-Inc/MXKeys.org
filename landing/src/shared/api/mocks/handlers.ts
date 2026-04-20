/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { http, HttpResponse } from 'msw';

/**
 * MSW request handlers for local development and tests.
 *
 * The landing page does not currently call any APIs. This file exists
 * as the agreed scaffold: future features (status widget, live
 * verifier, subscribe form, etc.) register their handlers here so
 * that tests and Storybook can run against a deterministic mock
 * layer without modifying consumer code.
 *
 * Organize by feature when adding real handlers:
 *   /api/verify         -> features/verifier
 *   /api/subscribe      -> features/subscribe
 *   /_mxkeys/status     -> widgets/status-panel
 *
 * See https://mswjs.io/ for the handler API reference.
 */
export const handlers = [
  // Placeholder that returns a deterministic shape so tests can assert
  // "MSW is wired" without depending on real backend endpoints.
  http.get('/api/landing/health', () =>
    HttpResponse.json({ status: 'ok', source: 'msw' }),
  ),
];
