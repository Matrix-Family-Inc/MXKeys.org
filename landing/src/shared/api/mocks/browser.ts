/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
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
