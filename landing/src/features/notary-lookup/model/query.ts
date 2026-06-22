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
import { verifyServer, type QueryResponse } from '../api/verify';

/**
 * useVerifyServer wraps verifyServer() in a TanStack mutation so the
 * widget can render loading / error / result states idiomatically.
 * The notary base URL comes from the Zod-validated env.
 */
export function useVerifyServer() {
  return useMutation<QueryResponse, Error, { serverName: string }>({
    mutationFn: ({ serverName }) =>
      verifyServer({
        baseURL: apiBaseURL(),
        serverName,
      }),
  });
}
