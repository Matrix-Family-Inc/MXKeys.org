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
   * POST /_matrix/key/v2/query is the notary's perspective lookup
   * endpoint. The request body is `{"server_keys": {"<name>": {}}}`;
   * the mock echoes each requested name back as a cached-looking
   * response so the Zod schema in verify.ts sees a valid payload.
   */
  http.post('*/_matrix/key/v2/query', async ({ request }) => {
    const body = (await request.json().catch(() => ({}))) as {
      server_keys?: Record<string, unknown>;
    };
    const names = Object.keys(body.server_keys ?? {});
    return HttpResponse.json({
      server_keys: names.map((name) => ({
        server_name: name,
        valid_until_ts: Date.now() + 24 * 60 * 60 * 1000,
        verify_keys: {
          'ed25519:auto': {
            key: 'Nzxs2Mh0Fb+Uhv3uTE47iWBoCGY8oSa11BZX9S7W6RE',
          },
        },
        old_verify_keys: {},
      })),
    });
  }),

  /**
   * /_mxkeys/server-info is the optional enrichment endpoint. The
   * mock returns a compact deterministic payload (DNS +
   * reachability + whois) so the widget test and Storybook see a
   * non-trivial ServerInfoPanel render without depending on real
   * external services.
   */
  http.get('*/_mxkeys/server-info', ({ request }) => {
    const url = new URL(request.url);
    const name = url.searchParams.get('name') ?? '';
    return HttpResponse.json({
      server_name: name,
      fetched_at: new Date().toISOString(),
      dns: {
        well_known_server: `${name}:8448`,
        srv: [{ target: `matrix.${name}`, port: 8448, priority: 10, weight: 0 }],
        resolved_host: `matrix.${name}`,
        resolved_port: 8448,
        a: ['203.0.113.42'],
        aaaa: [],
      },
      reachability: {
        federation_port: 8448,
        reachable: true,
        tls_version: 'TLS 1.3',
        tls_sni_match: true,
        rtt_ms: 42,
      },
      whois: {
        registrar: 'Example Registrar',
        registered: '2020-01-01',
        expires: '2030-01-01',
        nameservers: ['ns1.example.com', 'ns2.example.com'],
      },
    });
  }),

  /**
   * Keep a simple health stub so tests can assert MSW is wired.
   */
  http.get('*/api/landing/health', () =>
    HttpResponse.json({ status: 'ok', source: 'msw' }),
  ),
];
