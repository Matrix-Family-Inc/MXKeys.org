/*
 * Project: MXKeys
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 */

import { useTranslation } from 'react-i18next';

export function LanguageSwitcher() {
  const { i18n } = useTranslation();

  const changeLanguage = (lng: string) => {
    i18n.changeLanguage(lng);
    localStorage.setItem('mxkeys-lang', lng);
  };

  return (
    <div className="lang-switch">
      <button
        onClick={() => changeLanguage('en')}
        className={i18n.language === 'en' ? 'active' : ''}
      >
        EN
      </button>
      <button
        onClick={() => changeLanguage('ru')}
        className={i18n.language === 'ru' ? 'active' : ''}
      >
        RU
      </button>
    </div>
  );
}
