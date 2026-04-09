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

import { useTranslation } from 'react-i18next';
import { Logo } from './Logo';
import { MATRIX_CONTACTS, URLS, getLinkProps } from '../config/urls';

export function LandingFooter() {
  const { t } = useTranslation();

  return (
    <footer className="border-t border-[var(--color-border)] bg-[var(--color-bg)]">
      <div className="max-w-7xl mx-auto px-6 py-12">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-8">
          <div className="col-span-2 md:col-span-1">
            <div className="flex items-center gap-2 mb-4">
              <Logo size={28} />
              <span className="font-semibold text-[var(--color-text)]">MXKeys</span>
            </div>
            <p className="text-sm text-[var(--color-text-secondary)]">
              {t('footer.tagline')}
            </p>
          </div>

          <div>
            <h3 className="font-medium text-[var(--color-text)] mb-3">{t('footer.ecosystem')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href={URLS.matrixFamily} {...getLinkProps(URLS.matrixFamily)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.matrixFamily')}</a></li>
              <li><a href={URLS.hushmeApp} {...getLinkProps(URLS.hushmeApp)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushme')}</a></li>
              <li><a href={URLS.hushmeStore} {...getLinkProps(URLS.hushmeStore)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushmeStore')}</a></li>
              <li><a href={URLS.mxcore} {...getLinkProps(URLS.mxcore)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.mxcore')}</a></li>
              <li><a href={URLS.mfos} {...getLinkProps(URLS.mfos)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.mfos')}</a></li>
            </ul>
          </div>

          <div>
            <h3 className="font-medium text-[var(--color-text)] mb-3">{t('footer.resources')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href={URLS.hushmeOnline} {...getLinkProps(URLS.hushmeOnline)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushmeWeb')}</a></li>
              <li><a href={URLS.appsGateway} {...getLinkProps(URLS.appsGateway)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.appsGateway')}</a></li>
              <li><a href="#about" className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.architecture')}</a></li>
              <li><a href="#api" className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.apiReference')}</a></li>
            </ul>
          </div>

          <div>
            <h3 className="font-medium text-[var(--color-text)] mb-3">{t('footer.contact')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href={MATRIX_CONTACTS.support.href} {...getLinkProps(MATRIX_CONTACTS.support.href)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.support')}</a></li>
              <li><a href={MATRIX_CONTACTS.developer.href} {...getLinkProps(MATRIX_CONTACTS.developer.href)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.developer')}</a></li>
              <li><a href={MATRIX_CONTACTS.devChat.href} {...getLinkProps(MATRIX_CONTACTS.devChat.href)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.devChat')}</a></li>
            </ul>
          </div>

          <div>
            <h3 className="font-medium text-[var(--color-text)] mb-3">{t('footer.protocol')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href="https://spec.matrix.org/latest/server-server-api/#querying-keys-through-another-server" target="_blank" rel="noopener noreferrer" className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.matrixSpec')}</a></li>
              <li><a href={URLS.hushmeSpace} {...getLinkProps(URLS.hushmeSpace)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushmeSpace')}</a></li>
            </ul>
          </div>
        </div>

        <div className="mt-12 pt-8 border-t border-[var(--color-border)]">
          <p className="text-sm text-[var(--color-text-muted)] text-center">
            {t('footer.copyrightPrefix')}
            <a
              href={URLS.matrixFamily}
              {...getLinkProps(URLS.matrixFamily)}
              className="text-[var(--color-text-muted)] underline decoration-[var(--color-primary)]/50 underline-offset-2 hover:text-[var(--color-primary)]"
            >
              {t('footer.copyrightLink')}
            </a>
            {t('footer.copyrightSuffix')}
          </p>
        </div>
      </div>
    </footer>
  );
}
