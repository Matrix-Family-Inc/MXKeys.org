/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import {
  defaultLanguage,
  parseSupportedLanguage,
  rtlLanguages,
  type SupportedLanguage,
} from './supported';

const languageStorageKey = 'mxkeys.landing.lang';

function detectQueryLanguage(): SupportedLanguage | null {
  if (typeof window === 'undefined') return null;
  const requested = new URLSearchParams(window.location.search).get('lang');
  if (!requested) return null;
  return parseSupportedLanguage(requested) ?? defaultLanguage;
}

function detectStoredLanguage(): SupportedLanguage | null {
  if (typeof window === 'undefined') return null;
  try {
    return parseSupportedLanguage(window.localStorage.getItem(languageStorageKey));
  } catch {
    return null;
  }
}

function detectBrowserLanguage(): SupportedLanguage {
  if (typeof navigator === 'undefined') return defaultLanguage;
  const candidates = navigator.languages ?? [navigator.language];
  for (const lang of candidates) {
    const supported = parseSupportedLanguage(lang);
    if (supported) return supported;
  }
  return defaultLanguage;
}

/**
 * detectInitialLanguage walks the precedence chain the landing uses:
 * explicit ?lang= query parameter > previously-selected language in
 * localStorage > navigator language. Anything unknown collapses to the
 * default language.
 */
export function detectInitialLanguage(): SupportedLanguage {
  return detectQueryLanguage() ?? detectStoredLanguage() ?? detectBrowserLanguage();
}

/**
 * applyLanguageEffects updates document direction, persists the choice
 * to localStorage, and reflects it in the URL so deep links survive.
 */
export function applyLanguageEffects(language: SupportedLanguage) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = language;
    document.documentElement.dir = rtlLanguages.has(language) ? 'rtl' : 'ltr';
  }
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(languageStorageKey, language);
  } catch {
    // Incognito or storage-disabled modes: the site still works, just
    // without persistence. Intentionally silent.
  }
  const url = new URL(window.location.href);
  if (language === defaultLanguage) {
    url.searchParams.delete('lang');
  } else {
    url.searchParams.set('lang', language);
  }
  window.history.replaceState(window.history.state, '', url);
}
