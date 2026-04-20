/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { defineConfig, devices } from '@playwright/test';

/**
 * Minimal Playwright config for the landing smoke pass. CI invokes
 * `bun run e2e` which runs against a locally-served dev build. Tests
 * keep their footprint small: one browser, short timeouts, default HTML
 * reporter so CI logs stay readable.
 */
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  expect: { timeout: 5_000 },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? 'line' : 'html',
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://127.0.0.1:4173',
    trace: 'on-first-retry',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
  ],
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: 'bun run build && bun run preview --port 4173 --host 127.0.0.1',
        url: 'http://127.0.0.1:4173',
        timeout: 120_000,
        reuseExistingServer: !process.env.CI,
      },
});
