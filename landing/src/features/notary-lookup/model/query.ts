/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { useMutation } from '@tanstack/react-query';

import { env } from '@/shared/config/env';
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
        baseURL: env.VITE_SITE_URL,
        serverName,
      }),
  });
}
