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

const supportedLanguages = ['en', 'ru'];

function detectBrowserLanguage(): string {
  if (typeof navigator === 'undefined') {
    return 'en';
  }

  const browserLangs = navigator.languages || [navigator.language];
  
  for (const lang of browserLangs) {
    const code = lang.split('-')[0].toLowerCase();
    if (supportedLanguages.includes(code)) {
      return code;
    }
  }
  
  return 'en';
}

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    ru: { translation: ru },
  },
  lng: detectBrowserLanguage(),
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;
