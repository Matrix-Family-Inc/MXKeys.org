/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { z } from 'zod';

/**
 * Runtime-validated shape for the "look up this Matrix server's keys"
 * form. Exported as both the schema (for react-hook-form resolvers
 * and MSW handlers) and the inferred TypeScript type (for consumers
 * that need static typing).
 *
 * The server_name grammar follows Matrix spec MSC1711: either
 * `<hostname>` or `<hostname>:<port>`, where `<hostname>` is a
 * subset of ASCII, digits, and `-._`, and `<port>` is 1-65535.
 * We keep the check permissive; the backend performs the
 * authoritative validation via internal/server/validation.go.
 */
export const notaryLookupSchema = z.object({
  server_name: z
    .string()
    .min(1, 'Server name is required')
    .max(253, 'Server name is unreasonably long')
    .regex(
      /^[A-Za-z0-9.\-_]+(:[0-9]{1,5})?$/,
      'Must look like example.org or example.org:8448',
    ),
});

export type NotaryLookupInput = z.infer<typeof notaryLookupSchema>;
