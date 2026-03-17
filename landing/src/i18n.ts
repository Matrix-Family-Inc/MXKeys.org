/*
 * Project: MXKeys
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { en } from './locales/en';
import { ru } from './locales/ru';

const savedLang = typeof localStorage !== 'undefined' 
  ? localStorage.getItem('mxkeys-lang') 
  : null;

i18n.use(initReactI18next).init({
  resources: {
    en: { translation: en },
    ru: { translation: ru },
  },
  lng: savedLang || 'en',
  fallbackLng: 'en',
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;
