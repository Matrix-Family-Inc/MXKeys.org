/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import { useTranslation } from 'react-i18next';
import { Github, Menu, X } from 'lucide-react';

import { useMobileNav } from '../../../features/mobile-nav';
import { EXTERNAL, getLinkProps } from '../../../shared/config/urls';
import { Logo } from '../../../shared/ui';

const navLinks = [
  { href: '#about', key: 'about' },
  { href: '#how-it-works', key: 'howItWorks' },
  { href: '#api', key: 'api' },
  { href: '#ecosystem', key: 'ecosystem' },
] as const;

export function LandingHeader() {
  const { t } = useTranslation();
  const open = useMobileNav((s) => s.open);
  const toggle = useMobileNav((s) => s.toggle);
  const close = useMobileNav((s) => s.close);

  const menuLabel = open ? t('nav.closeMenu') : t('nav.openMenu');
  const iconBtn =
    'inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:border-[var(--color-primary)]/40 transition-colors';

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

          <a
            href={EXTERNAL.github}
            {...getLinkProps(EXTERNAL.github)}
            aria-label={t('nav.github')}
            className={iconBtn}
          >
            <Github size={18} aria-hidden="true" />
          </a>
        </div>

        <div className="flex items-center gap-2 md:hidden">
          <a
            href={EXTERNAL.github}
            {...getLinkProps(EXTERNAL.github)}
            aria-label={t('nav.github')}
            className={iconBtn}
          >
            <Github size={18} aria-hidden="true" />
          </a>

          <button
            type="button"
            aria-label={menuLabel}
            aria-expanded={open}
            onClick={toggle}
            className={iconBtn}
          >
            {open ? <X size={18} aria-hidden="true" /> : <Menu size={18} aria-hidden="true" />}
          </button>
        </div>
      </div>

      {open ? (
        <div className="border-t border-[var(--color-border)] md:hidden">
          <nav className="max-w-7xl mx-auto px-6 py-3 flex flex-col gap-1">
            {navLinks.map((link) => (
              <a key={link.key} href={link.href} className="nav-link" onClick={close}>
                {t(`nav.${link.key}`)}
              </a>
            ))}
          </nav>
        </div>
      ) : null}
    </header>
  );
}
