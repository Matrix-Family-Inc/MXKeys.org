/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { en } from './locales/en';
import { ru } from './locales/ru';

export const supportedLanguages = ['en', 'ru'] as const;

type SupportedLanguage = (typeof supportedLanguages)[number];

const defaultLanguage: SupportedLanguage = 'en';
const languageStorageKey = 'mxkeys.landing.lang';

function isSupportedLanguage(language: string): language is SupportedLanguage {
  return supportedLanguages.includes(language as SupportedLanguage);
}

function parseSupportedLanguage(language: string | null | undefined): SupportedLanguage | null {
  if (!language) {
    return null;
  }

  const code = language.split('-')[0].toLowerCase();
  return isSupportedLanguage(code) ? code : null;
}

function normalizeLanguage(language: string | null | undefined): SupportedLanguage {
  return parseSupportedLanguage(language) ?? defaultLanguage;
}

function detectQueryLanguage(): SupportedLanguage | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const requested = new URLSearchParams(window.location.search).get('lang');
  if (!requested) {
    return null;
  }

  return parseSupportedLanguage(requested) ?? defaultLanguage;
}

function detectStoredLanguage(): SupportedLanguage | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const stored = window.localStorage.getItem(languageStorageKey);
  return parseSupportedLanguage(stored);
}

function detectBrowserLanguage(): SupportedLanguage {
  if (typeof navigator === 'undefined') {
    return defaultLanguage;
  }

  const browserLangs = navigator.languages || [navigator.language];

  for (const lang of browserLangs) {
    const supported = parseSupportedLanguage(lang);
    if (supported) {
      return supported;
    }
  }

  return defaultLanguage;
}

function detectInitialLanguage(): SupportedLanguage {
  return detectQueryLanguage() ?? detectStoredLanguage() ?? detectBrowserLanguage();
}

function applyLanguageEffects(language: SupportedLanguage) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = language;
  }

  if (typeof window !== 'undefined') {
    window.localStorage.setItem(languageStorageKey, language);

    const url = new URL(window.location.href);
    if (language === defaultLanguage) {
      url.searchParams.delete('lang');
    } else {
      url.searchParams.set('lang', language);
    }
    window.history.replaceState(window.history.state, '', url);
  }
}

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    ru: { translation: ru },
  },
  lng: detectInitialLanguage(),
  fallbackLng: defaultLanguage,
  interpolation: {
    escapeValue: false,
  },
});

const initialLanguage = normalizeLanguage(i18n.resolvedLanguage);
applyLanguageEffects(initialLanguage);
i18n.on('languageChanged', (language) => {
  applyLanguageEffects(normalizeLanguage(language));
});

export async function setPreferredLanguage(language: string) {
  await i18n.changeLanguage(normalizeLanguage(language));
}

export default i18n;
