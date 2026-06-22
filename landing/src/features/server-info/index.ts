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

// Public API barrel for the server-info feature slice: every
// consumer outside this directory imports from here, not from
// the inner files. Keeps the slice boundary explicit per FSD.

export { fetchServerInfo, serverInfoSchema } from './api/server-info';
export type {
  ServerInfo,
  ServerInfoDns,
  ServerInfoReachability,
  ServerInfoWhois,
} from './api/server-info';
export { useServerInfo } from './model/query';
export { ServerInfoPanel } from './ui/server-info-panel';
