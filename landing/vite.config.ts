/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import path from 'node:path';

import react from '@vitejs/plugin-react';
import { loadEnv, type Plugin } from 'vite';
import { defineConfig } from 'vitest/config';

/**
 * Replace %VITE_SITE_URL% (and similar) placeholders in index.html at
 * build/serve time so operators deploying a forked landing under their
 * own brand don't need to patch the HTML by hand.
 */
function htmlEnvReplace(env: Record<string, string>): Plugin {
  return {
    name: 'html-env-replace',
    transformIndexHtml(html) {
      const siteURL = (env.VITE_SITE_URL || 'https://notary.example.org').replace(/\/+$/, '');
      return html
        .replace(/__MXKEYS_SITE_URL__/g, siteURL)
        .replace(/__MXKEYS_ENVIRONMENT__/g, env.VITE_ENVIRONMENT || 'development');
    },
  };
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  return {
    plugins: [react(), htmlEnvReplace(env)],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
    },
    server: {
      port: parseInt(process.env.PORT || '3005'),
      host: '0.0.0.0',
      allowedHosts: true,
    },
    build: {
      outDir: 'dist',
      sourcemap: mode === 'production',
      chunkSizeWarningLimit: 600,
      rollupOptions: {
        output: {
          // Split heavy third-party groups into their own chunks so the
          // main bundle does not carry Sentry/Tanstack on every page load.
          // Locale chunks come for free from the lazy backend.
          manualChunks: {
            sentry: ['@sentry/react'],
            tanstack: ['@tanstack/react-router', '@tanstack/react-query'],
            i18n: ['i18next', 'react-i18next', 'i18next-resources-to-backend'],
          },
        },
      },
    },
    test: {
      environment: 'happy-dom',
      setupFiles: './src/test/setup.ts',
      css: true,
      // Playwright specs live under e2e/; keep them out of vitest's glob.
      exclude: ['node_modules/**', 'dist/**', 'e2e/**'],
    },
  };
});
