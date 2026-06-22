/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

import { setupWorker } from 'msw/browser';
import { handlers } from './handlers';

/**
 * Browser-side MSW worker factory. Intentionally NOT started at module
 * load: callers opt in via `worker.start()` from the dev entry point
 * or a Storybook decorator, so production builds never ship the
 * service-worker registration.
 */
export const worker = setupWorker(...handlers);
