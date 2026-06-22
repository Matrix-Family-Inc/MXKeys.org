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

import { useMutation } from '@tanstack/react-query';

import { apiBaseURL } from '@/shared/config/env';
import { fetchServerInfo, type ServerInfo } from '../api/server-info';

/**
 * useServerInfo wraps fetchServerInfo() in a TanStack mutation
 * so the widget can trigger it on the same submit event as the
 * key lookup. Mutation semantics (not useQuery) keep the state
 * aligned with the single-shot form interaction: one submit -
 * one request, no refetch on window focus.
 *
 * Resolves to `null` when the notary does not expose
 * /_mxkeys/server-info (503), so the widget can hide the
 * enrichment panel entirely instead of rendering "feature off".
 */
export function useServerInfo() {
  return useMutation<ServerInfo | null, Error, { serverName: string }>({
    mutationFn: ({ serverName }) =>
      fetchServerInfo({
        baseURL: apiBaseURL(),
        serverName,
      }),
  });
}
