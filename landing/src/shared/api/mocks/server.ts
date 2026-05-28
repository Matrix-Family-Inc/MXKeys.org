/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

/**
 * Node-side MSW server factory for Vitest / unit tests. Import this
 * from the test setup module and call server.listen() in beforeAll,
 * server.close() in afterAll, server.resetHandlers() in afterEach.
 */
export const server = setupServer(...handlers);
