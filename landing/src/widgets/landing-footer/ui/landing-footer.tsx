/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

import { useTranslation } from 'react-i18next';
import { Github } from 'lucide-react';
import { Logo } from '@/shared/ui';
import { MATRIX_CONTACTS, URLS, getLinkProps } from '@/shared/config/urls';

export function LandingFooter() {
  const { t } = useTranslation();

  return (
    <footer className="border-t border-border bg-bg">
      <div className="max-w-7xl mx-auto px-6 py-12">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-8">
          <div className="col-span-2 md:col-span-1">
            <div className="flex items-center gap-2 mb-4">
              <Logo size={28} />
              <span className="font-semibold text-text">MXKeys</span>
            </div>
            <p className="text-sm text-text-secondary">
              {t('footer.tagline')}
            </p>
          </div>

          <div>
            <h3 className="font-medium text-text mb-3">{t('footer.ecosystem')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href={URLS.matrixFamily} {...getLinkProps(URLS.matrixFamily)} className="text-text-secondary hover:text-primary">{t('footer.matrixFamily')}</a></li>
              <li><a href={URLS.hushmeApp} {...getLinkProps(URLS.hushmeApp)} className="text-text-secondary hover:text-primary">{t('footer.hushme')}</a></li>
              <li><a href={URLS.hushmeStore} {...getLinkProps(URLS.hushmeStore)} className="text-text-secondary hover:text-primary">{t('footer.hushmeStore')}</a></li>
              <li><a href={URLS.mxcore} {...getLinkProps(URLS.mxcore)} className="text-text-secondary hover:text-primary">{t('footer.mxcore')}</a></li>
              <li><a href={URLS.mfos} {...getLinkProps(URLS.mfos)} className="text-text-secondary hover:text-primary">{t('footer.mfos')}</a></li>
            </ul>
          </div>

          <div>
            <h3 className="font-medium text-text mb-3">{t('footer.resources')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href={URLS.hushmeOnline} {...getLinkProps(URLS.hushmeOnline)} className="text-text-secondary hover:text-primary">{t('footer.hushmeWeb')}</a></li>
              <li><a href={URLS.appsGateway} {...getLinkProps(URLS.appsGateway)} className="text-text-secondary hover:text-primary">{t('footer.appsGateway')}</a></li>
              <li><a href="#about" className="text-text-secondary hover:text-primary">{t('footer.architecture')}</a></li>
              <li><a href="#api" className="text-text-secondary hover:text-primary">{t('footer.apiReference')}</a></li>
            </ul>
          </div>

          <div>
            <h3 className="font-medium text-text mb-3">{t('footer.contact')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href={MATRIX_CONTACTS.support.href} {...getLinkProps(MATRIX_CONTACTS.support.href)} className="text-text-secondary hover:text-primary">{t('footer.support')}</a></li>
              <li><a href={MATRIX_CONTACTS.developer.href} {...getLinkProps(MATRIX_CONTACTS.developer.href)} className="text-text-secondary hover:text-primary">{t('footer.developer')}</a></li>
              <li><a href={MATRIX_CONTACTS.devChat.href} {...getLinkProps(MATRIX_CONTACTS.devChat.href)} className="text-text-secondary hover:text-primary">{t('footer.devChat')}</a></li>
            </ul>
          </div>

          <div>
            <h3 className="font-medium text-text mb-3">{t('footer.protocol')}</h3>
            <ul className="space-y-2 text-sm">
              <li><a href="https://spec.matrix.org/latest/server-server-api/#querying-keys-through-another-server" target="_blank" rel="noopener noreferrer" className="text-text-secondary hover:text-primary">{t('footer.matrixSpec')}</a></li>
              <li><a href={URLS.hushmeSpace} {...getLinkProps(URLS.hushmeSpace)} className="text-text-secondary hover:text-primary">{t('footer.hushmeSpace')}</a></li>
            </ul>
          </div>
        </div>

        <div className="mt-12 pt-8 border-t border-border flex flex-col items-center gap-4">
          <a
            href="https://github.com/Matrix-Family-Inc"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 text-text-muted hover:text-primary transition-colors"
          >
            <Github size={20} />
            <span className="text-sm">github.com/Matrix-Family-Inc</span>
          </a>
          <p className="text-sm text-text-muted">
            © {new Date().getFullYear()} Matrix Family Inc. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}
