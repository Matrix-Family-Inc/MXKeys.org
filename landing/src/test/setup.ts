/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { server } from '@/shared/api/mocks/server';

// MSW lifecycle: all unit/integration tests share a single server
// instance, reset handlers between tests so one test cannot leak
// state into another.
beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
