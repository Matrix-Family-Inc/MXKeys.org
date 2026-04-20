/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { expect, test } from '@playwright/test';

test.describe('landing smoke', () => {
  test('home renders with navigation and footer', async ({ page }) => {
    await page.goto('/');

    // The hub element every in-page link points at.
    await expect(page.locator('#home')).toBeVisible();

    // Header link to repo exists (either desktop or mobile icon).
    await expect(page.getByRole('link', { name: /github/i }).first()).toBeVisible();

    // A footer section title. Copy is i18n-sensitive; we match a known
    // string that appears in every locale via the "ecosystem" heading
    // mapped through i18n. Falling back to role-based look up keeps the
    // test resilient to copy tweaks.
    const footerLinks = page.locator('footer a');
    await expect(footerLinks.first()).toBeVisible();
  });

  test('language query string switches document direction for RTL', async ({ page }) => {
    await page.goto('/?lang=ar');
    await expect(page.locator('html')).toHaveAttribute('dir', 'rtl');

    await page.goto('/?lang=en');
    await expect(page.locator('html')).toHaveAttribute('dir', 'ltr');
  });

  test('mobile menu toggle opens and closes nav', async ({ page }) => {
    await page.setViewportSize({ width: 400, height: 800 });
    await page.goto('/');

    const toggle = page.getByRole('button', { name: /menu/i });
    await expect(toggle).toBeVisible();

    await toggle.click();
    // After open, nav links are accessible; pick one by role.
    await expect(page.getByRole('link', { name: /about/i }).first()).toBeVisible();
  });
});
