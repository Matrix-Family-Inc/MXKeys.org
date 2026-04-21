/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import { http, HttpResponse } from 'msw';

/**
 * MSW request handlers used in Vitest and in the browser when the
 * landing page runs under MODE=development with mocks enabled.
 *
 * Each handler mirrors the shape of a real MXKeys endpoint so the
 * components under test never observe a synthetic-only field.
 */
export const handlers = [
  /**
   * /_matrix/key/v2/query/{server_name} is the notary's perspective
   * lookup endpoint. The mock returns a deterministic cached-looking
   * response whose server_name matches the request so the Zod schema
   * in verify.ts sees a valid payload.
   */
  http.get('*/_matrix/key/v2/query/:server', ({ params }) => {
    const raw = params.server;
    const server = typeof raw === 'string' ? raw : Array.isArray(raw) ? raw[0] : '';
    return HttpResponse.json({
      server_keys: [
        {
          server_name: server,
          valid_until_ts: Date.now() + 24 * 60 * 60 * 1000,
          verify_keys: {
            'ed25519:auto': {
              key: 'Nzxs2Mh0Fb+Uhv3uTE47iWBoCGY8oSa11BZX9S7W6RE',
            },
          },
          old_verify_keys: {},
        },
      ],
    });
  }),

  /**
   * Keep a simple health stub so tests can assert MSW is wired.
   */
  http.get('*/api/landing/health', () =>
    HttpResponse.json({ status: 'ok', source: 'msw' }),
  ),
];
