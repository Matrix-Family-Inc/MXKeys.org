/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import React from 'react';
import ReactDOM from 'react-dom/client';

import { App } from './app/app';
import { initSentry } from './app/providers/sentry';
import { initI18n } from './shared/i18n';
import './index.css';

/**
 * Bootstrap sequence:
 *  1. Init Sentry (no-op when no DSN) so any error from the steps below
 *     is captured.
 *  2. Init i18n with the lazy-loaded resource backend. Awaited so the
 *     first React render already has translations loaded; prevents the
 *     flash-of-untranslated-content on slow networks.
 *  3. Mount the React tree.
 *
 * Failures during init fall back to a minimal, untranslated page rather
 * than a blank screen: the static HTML already has meaningful content
 * via the index.html shell.
 */
async function bootstrap() {
  initSentry();
  try {
    await initI18n();
  } catch (err) {
    console.error('[bootstrap] i18n init failed, falling back to untranslated render', err);
  }

  const rootElement = document.getElementById('root');
  if (!rootElement) {
    console.error('[bootstrap] #root element not found');
    return;
  }
  ReactDOM.createRoot(rootElement).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>,
  );
}

void bootstrap();
