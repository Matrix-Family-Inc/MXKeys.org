/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

import { z } from 'zod';

/**
 * Zod schema for GET /_mxkeys/server-info?name=<host>. Mirrors
 * the Go-side ServerInfoResponse in internal/server/
 * serverinfo_types.go. Every sub-section is optional because the
 * backend always returns 200 with whatever succeeded within the
 * request budget, and the widget renders whichever fields come
 * back without demanding a specific combination.
 */
export const serverInfoSrvTargetSchema = z.object({
  target: z.string(),
  port: z.number().int().nonnegative(),
  priority: z.number().int().nonnegative(),
  weight: z.number().int().nonnegative(),
});

export const serverInfoDnsSchema = z.object({
  well_known_server: z.string().optional(),
  srv: z.array(serverInfoSrvTargetSchema).optional(),
  resolved_host: z.string().optional(),
  resolved_port: z.number().int().nonnegative().optional(),
  a: z.array(z.string()).optional(),
  aaaa: z.array(z.string()).optional(),
});

export const serverInfoReachabilitySchema = z.object({
  federation_port: z.number().int().nonnegative(),
  reachable: z.boolean(),
  tls_version: z.string().optional(),
  tls_sni_match: z.boolean().optional(),
  rtt_ms: z.number().nonnegative().optional(),
  error: z.string().optional(),
});

export const serverInfoWhoisSchema = z.object({
  registrar: z.string().optional(),
  registered: z.string().optional(),
  expires: z.string().optional(),
  updated: z.string().optional(),
  nameservers: z.array(z.string()).optional(),
});

export const serverInfoSchema = z.object({
  server_name: z.string(),
  fetched_at: z.string(),
  dns: serverInfoDnsSchema.optional(),
  reachability: serverInfoReachabilitySchema.optional(),
  whois: serverInfoWhoisSchema.optional(),
  errors: z.record(z.string(), z.string()).optional(),
});

export type ServerInfo = z.infer<typeof serverInfoSchema>;
export type ServerInfoReachability = z.infer<typeof serverInfoReachabilitySchema>;
export type ServerInfoDns = z.infer<typeof serverInfoDnsSchema>;
export type ServerInfoWhois = z.infer<typeof serverInfoWhoisSchema>;

export interface FetchServerInfoArgs {
  /** Base URL of the notary, e.g. "https://mxkeys.org". */
  baseURL: string;
  /** Matrix server_name being queried. */
  serverName: string;
  /** Optional abort signal tied to the caller's request lifetime. */
  signal?: AbortSignal;
}

/**
 * Calls GET /_mxkeys/server-info. Returns the parsed response
 * when the notary responds 2xx with a valid shape; resolves to
 * `null` when the endpoint is not enabled on this notary (HTTP
 * 503); rejects on transport errors and unexpected statuses so
 * the caller can distinguish "feature off" from "real failure".
 */
export async function fetchServerInfo(args: FetchServerInfoArgs): Promise<ServerInfo | null> {
  const base = args.baseURL.replace(/\/+$/, '');
  const endpoint = `${base}/_mxkeys/server-info?name=${encodeURIComponent(args.serverName)}`;
  const res = await fetch(endpoint, {
    method: 'GET',
    headers: { Accept: 'application/json' },
    signal: args.signal,
  });
  if (res.status === 503) {
    return null;
  }
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    throw new Error(`server-info responded with ${res.status}${text ? `: ${text}` : ''}`);
  }
  const json = (await res.json()) as unknown;
  return serverInfoSchema.parse(json);
}
