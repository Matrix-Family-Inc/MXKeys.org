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
 * Strict Zod schema for the Matrix v1.16 /key/v2/server response shape.
 * The notary's /_matrix/key/v2/query endpoint returns this wrapped in a
 * server_keys array, so we validate each element.
 *
 * Keeping the schema here means the widget never trusts the backend
 * blindly; a server that returns garbage surfaces as a parse error in
 * the caller's onError path.
 */
export const verifyKeySchema = z.object({
  key: z.string(),
});

export const serverKeysSchema = z.object({
  server_name: z.string(),
  valid_until_ts: z.number().nonnegative(),
  verify_keys: z.record(z.string(), verifyKeySchema),
  old_verify_keys: z.record(z.string(), verifyKeySchema).optional(),
});

export const queryResponseSchema = z.object({
  server_keys: z.array(serverKeysSchema),
});

export type ServerKeys = z.infer<typeof serverKeysSchema>;
export type QueryResponse = z.infer<typeof queryResponseSchema>;

export interface VerifyArgs {
  /** Base URL of the notary, e.g. "https://notary.example.org". */
  baseURL: string;
  /** Matrix server_name being queried, e.g. "matrix.org". */
  serverName: string;
  /** Optional abort signal tied to the caller's request lifetime. */
  signal?: AbortSignal;
}

/**
 * Calls the notary's POST /_matrix/key/v2/query endpoint with the
 * single-server shape {"server_keys": {"<name>": {}}}. Returns the
 * parsed response; a non-2xx status or a shape that fails Zod
 * validation rejects with a descriptive Error.
 *
 * The POST-batch form is the route MXKeys registers; the deprecated
 * GET /_matrix/key/v2/query/{server_name} form is intentionally not
 * implemented. Using POST here keeps the landing on the supported
 * public API surface.
 */
export async function verifyServer(args: VerifyArgs): Promise<QueryResponse> {
  const endpoint = `${args.baseURL.replace(/\/+$/, '')}/_matrix/key/v2/query`;
  const body = JSON.stringify({
    server_keys: { [args.serverName]: {} },
  });
  const res = await fetch(endpoint, {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body,
    signal: args.signal,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`notary responded with ${res.status}${text ? `: ${text}` : ''}`);
  }
  const json = (await res.json()) as unknown;
  return queryResponseSchema.parse(json);
}
