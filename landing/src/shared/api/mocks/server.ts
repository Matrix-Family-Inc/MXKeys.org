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

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

/**
 * Node-side MSW server factory for Vitest / unit tests. Import this
 * from the test setup module and call server.listen() in beforeAll,
 * server.close() in afterAll, server.resetHandlers() in afterEach.
 */
export const server = setupServer(...handlers);
