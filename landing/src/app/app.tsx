/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { RouterProvider } from '@tanstack/react-router';

import { AppErrorBoundary } from './providers/error-boundary';
import { QueryProvider } from './providers/query';
import { router } from './router/router';

/**
 * App is the top-level component. Order matters: ErrorBoundary wraps
 * everything so render errors anywhere inside are caught and reported;
 * QueryProvider is inside the boundary so data-fetching crashes still
 * surface to Sentry; RouterProvider sits at the bottom because it owns
 * page composition.
 */
export function App() {
  return (
    <AppErrorBoundary>
      <QueryProvider>
        <RouterProvider router={router} />
      </QueryProvider>
    </AppErrorBoundary>
  );
}
