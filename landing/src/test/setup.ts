/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import '@testing-library/jest-dom/vitest';
import { afterAll, afterEach, beforeAll } from 'vitest';
import { server } from '@/shared/api/mocks/server';
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import en from '@/shared/i18n/locales/en';

// MSW lifecycle: all unit/integration tests share a single server
// instance, reset handlers between tests so one test cannot leak
// state into another.
beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

// Initialise i18next once so every useTranslation() in the tree
// returns real English strings instead of the raw i18n keys. Real
// landing bootstrap loads locales lazily; here we inline the en
// bundle so tests stay deterministic.
if (!i18n.isInitialized) {
  void i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    resources: { en: { translation: en } },
    interpolation: { escapeValue: false },
  });
}
