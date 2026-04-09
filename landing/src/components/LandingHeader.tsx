/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Github, Menu, X } from 'lucide-react';
import { Logo } from './Logo';
import { EXTERNAL, getLinkProps } from '../config/urls';
import { setPreferredLanguage, supportedLanguages } from '../i18n';

const navLinks = [
  { href: '#about', key: 'about' },
  { href: '#how-it-works', key: 'howItWorks' },
  { href: '#api', key: 'api' },
  { href: '#ecosystem', key: 'ecosystem' },
] as const;

type LanguageSwitchProps = {
  currentLanguage: string;
  label: string;
  onChange: (language: string) => void;
};

function LanguageSwitch({ currentLanguage, label, onChange }: LanguageSwitchProps) {
  return (
    <div className="lang-switch" role="group" aria-label={label}>
      {supportedLanguages.map((language) => (
        <button
          key={language}
          type="button"
          className={currentLanguage === language ? 'active' : undefined}
          onClick={() => onChange(language)}
          aria-pressed={currentLanguage === language}
        >
          {language.toUpperCase()}
        </button>
      ))}
    </div>
  );
}

export function LandingHeader() {
  const { t, i18n } = useTranslation();
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  const currentLanguage = i18n.resolvedLanguage === 'ru' ? 'ru' : 'en';
  const menuLabel = mobileNavOpen ? t('nav.closeMenu') : t('nav.openMenu');

  const handleLanguageChange = (language: string) => {
    setMobileNavOpen(false);
    void setPreferredLanguage(language);
  };

  return (
    <header className="sticky top-0 z-50 border-b border-[var(--color-border)] bg-[var(--color-bg)]/95 backdrop-blur-sm">
      <div className="max-w-7xl mx-auto h-16 px-6 flex items-center justify-between">
        <a href="#home" aria-label={t('nav.homeAria')} className="flex items-center gap-3">
          <Logo size={32} />
          <span className="text-lg font-semibold text-[var(--color-text)]">MXKeys</span>
        </a>

        <div className="hidden md:flex items-center gap-3">
          <nav className="flex items-center gap-1">
            {navLinks.map((link) => (
              <a key={link.key} href={link.href} className="nav-link">
                {t(`nav.${link.key}`)}
              </a>
            ))}
          </nav>

          <LanguageSwitch
            currentLanguage={currentLanguage}
            label={t('nav.language')}
            onChange={handleLanguageChange}
          />

          <a
            href={EXTERNAL.github}
            {...getLinkProps(EXTERNAL.github)}
            aria-label={t('nav.github')}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:border-[var(--color-primary)]/40 transition-colors"
          >
            <Github size={18} aria-hidden="true" />
          </a>
        </div>

        <div className="flex items-center gap-2 md:hidden">
          <LanguageSwitch
            currentLanguage={currentLanguage}
            label={t('nav.language')}
            onChange={handleLanguageChange}
          />

          <a
            href={EXTERNAL.github}
            {...getLinkProps(EXTERNAL.github)}
            aria-label={t('nav.github')}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:border-[var(--color-primary)]/40 transition-colors"
          >
            <Github size={18} aria-hidden="true" />
          </a>

          <button
            type="button"
            aria-label={menuLabel}
            aria-expanded={mobileNavOpen}
            onClick={() => setMobileNavOpen((open) => !open)}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:border-[var(--color-primary)]/40 transition-colors"
          >
            {mobileNavOpen ? <X size={18} aria-hidden="true" /> : <Menu size={18} aria-hidden="true" />}
          </button>
        </div>
      </div>

      {mobileNavOpen ? (
        <div className="border-t border-[var(--color-border)] md:hidden">
          <nav className="max-w-7xl mx-auto px-6 py-3 flex flex-col gap-1">
            {navLinks.map((link) => (
              <a
                key={link.key}
                href={link.href}
                className="nav-link"
                onClick={() => setMobileNavOpen(false)}
              >
                {t(`nav.${link.key}`)}
              </a>
            ))}
          </nav>
        </div>
      ) : null}
    </header>
  );
}
