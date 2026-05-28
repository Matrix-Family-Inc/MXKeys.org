/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
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
