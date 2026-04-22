/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import * as Sentry from '@sentry/react';
import type { ReactNode } from 'react';

/**
 * AppErrorBoundary reports any React render-path exception to Sentry (when
 * configured) and shows a minimal fallback. The landing is static enough
 * that recovering is usually a reload; the fallback therefore nudges the
 * visitor rather than trying to resume rendering.
 */
export function AppErrorBoundary({ children }: { children: ReactNode }) {
  return (
    <Sentry.ErrorBoundary
      fallback={({ resetError }) => (
        <div
          role="alert"
          className="min-h-screen flex items-center justify-center p-6 text-center"
        >
          <div className="max-w-md space-y-4">
            <h1 className="text-2xl font-semibold">Something went wrong</h1>
            <p className="text-text-secondary">
              The page failed to render. Refresh to try again; if it keeps
              happening, the error has been reported.
            </p>
            <button
              type="button"
              onClick={() => {
                resetError();
                if (typeof window !== 'undefined') {
                  window.location.reload();
                }
              }}
              className="inline-flex h-10 items-center justify-center rounded-md border border-border px-4 text-sm font-medium hover:bg-bg-hover"
            >
              Reload
            </button>
          </div>
        </div>
      )}
    >
      {children}
    </Sentry.ErrorBoundary>
  );
}
