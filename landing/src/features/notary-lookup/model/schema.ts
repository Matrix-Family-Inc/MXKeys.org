/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

import { z } from 'zod';

/**
 * Runtime-validated shape for the notary-lookup form.
 *
 * UX contract: the visitor types a bare hostname (`matrix.org`).
 * Port is optional; the notary performs Matrix discovery
 * (`.well-known/matrix/server`, SRV, 8448 fallback) on its own
 * and the form must not scold the visitor for omitting one.
 *
 * Error codes are deliberately machine-friendly identifiers
 * (`empty`, `too_long`, `bad_shape`, `bad_port`). The React
 * form resolver maps them to localised strings via
 * `t('lookup.validation.<code>')` so the same schema drives
 * both English and Russian builds.
 */
const SCHEME_PREFIX = /^https?:\/\//i;
const HOST_AND_OPTIONAL_PORT = /^(?:\[[0-9a-f:]+\]|[a-z0-9.\-_]+)(?::[0-9]{1,5})?$/i;

export const notaryLookupSchema = z.object({
  server_name: z
    .string()
    .transform((raw) => {
      let v = raw.trim().replace(SCHEME_PREFIX, '');
      if (v.endsWith('/')) v = v.slice(0, -1);
      const colon = v.lastIndexOf(':');
      const isIPv6Literal = v.startsWith('[');
      if (!isIPv6Literal && colon === -1) return v.toLowerCase();
      if (!isIPv6Literal) {
        return v.slice(0, colon).toLowerCase() + v.slice(colon);
      }
      return v;
    })
    .pipe(
      z
        .string()
        .min(1, 'empty')
        .max(253, 'too_long')
        .regex(HOST_AND_OPTIONAL_PORT, 'bad_shape')
        .refine((v) => {
          const colon = v.lastIndexOf(':');
          const hasPort = !v.startsWith('[') && colon !== -1;
          if (!hasPort) return true;
          const port = Number(v.slice(colon + 1));
          return Number.isInteger(port) && port >= 1 && port <= 65535;
        }, 'bad_port'),
    ),
});

export type NotaryLookupInput = z.infer<typeof notaryLookupSchema>;

/** Map of schema error codes -> i18n keys under `lookup.validation.*`. */
export const NOTARY_LOOKUP_ERROR_I18N: Record<string, string> = {
  empty: 'lookup.validation.empty',
  too_long: 'lookup.validation.tooLong',
  bad_shape: 'lookup.validation.badShape',
  bad_port: 'lookup.validation.badPort',
};
