/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import resourcesToBackend from 'i18next-resources-to-backend';

import { applyLanguageEffects, detectInitialLanguage } from './detect';
import { defaultLanguage, parseSupportedLanguage, type SupportedLanguage } from './supported';

/**
 * Lazy-load the per-language bundle. Vite turns the dynamic import pattern
 * below into one chunk per locale; only the active language is downloaded
 * on first paint, dropping initial JS payload by ~90% (previously all 22
 * locales were bundled eagerly).
 */
const backend = resourcesToBackend(async (language: string) => {
  const module = (await import(`./locales/${language}.ts`)) as { default: unknown };
  return module.default as Record<string, unknown>;
});

function normalize(language: string | null | undefined): SupportedLanguage {
  return parseSupportedLanguage(language) ?? defaultLanguage;
}

/**
 * initI18n wires i18next with the lazy backend and returns the ready
 * instance. Must be awaited before rendering so the first paint has the
 * correct translations (no FOUC).
 */
export async function initI18n() {
  const initial = detectInitialLanguage();

  await i18n
    .use(backend)
    .use(initReactI18next)
    .init({
      lng: initial,
      fallbackLng: defaultLanguage,
      ns: 'translation',
      defaultNS: 'translation',
      interpolation: { escapeValue: false },
      react: { useSuspense: false },
    });

  applyLanguageEffects(normalize(i18n.resolvedLanguage));

  i18n.on('languageChanged', (language) => {
    applyLanguageEffects(normalize(language));
  });

  return i18n;
}

export async function setPreferredLanguage(language: string) {
  await i18n.changeLanguage(normalize(language));
}

export default i18n;
