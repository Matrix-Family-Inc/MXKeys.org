/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { ar } from './locales/ar';
import { bn } from './locales/bn';
import { de } from './locales/de';
import { en } from './locales/en';
import { es } from './locales/es';
import { fr } from './locales/fr';
import { he } from './locales/he';
import { hi } from './locales/hi';
import { id } from './locales/id';
import { it } from './locales/it';
import { ja } from './locales/ja';
import { ko } from './locales/ko';
import { nl } from './locales/nl';
import { pl } from './locales/pl';
import { pt } from './locales/pt';
import { ru } from './locales/ru';
import { th } from './locales/th';
import { tr } from './locales/tr';
import { uk } from './locales/uk';
import { ur } from './locales/ur';
import { vi } from './locales/vi';
import { zh } from './locales/zh';

export const supportedLanguages = [
  'ar', 'bn', 'de', 'en', 'es', 'fr', 'he', 'hi', 'id', 'it', 'ja',
  'ko', 'nl', 'pl', 'pt', 'ru', 'th', 'tr', 'uk', 'ur', 'vi', 'zh',
] as const;

const rtlLanguages = new Set(['ar', 'he', 'ur']);

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
    document.documentElement.dir = rtlLanguages.has(language) ? 'rtl' : 'ltr';
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
    ar: { translation: ar },
    bn: { translation: bn },
    de: { translation: de },
    en: { translation: en },
    es: { translation: es },
    fr: { translation: fr },
    he: { translation: he },
    hi: { translation: hi },
    id: { translation: id },
    it: { translation: it },
    ja: { translation: ja },
    ko: { translation: ko },
    nl: { translation: nl },
    pl: { translation: pl },
    pt: { translation: pt },
    ru: { translation: ru },
    th: { translation: th },
    tr: { translation: tr },
    uk: { translation: uk },
    ur: { translation: ur },
    vi: { translation: vi },
    zh: { translation: zh },
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
