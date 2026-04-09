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
import { BookOpen, ExternalLink, Globe, MessageSquare, Server, Store } from 'lucide-react';
import { URLS, getLinkProps } from '../config/urls';

const ecosystemItems = [
  { key: 'matrixFamily', icon: Globe, href: URLS.matrixFamily },
  { key: 'hushme', icon: MessageSquare, href: URLS.hushmeApp },
  { key: 'hushmeStore', icon: Store, href: URLS.hushmeStore },
  { key: 'mxcore', icon: Server, href: URLS.mxcore },
  { key: 'mfos', icon: BookOpen, href: URLS.mfos },
];

export function EcosystemSection() {
  const { t } = useTranslation();

  return (
    <section id="ecosystem" className="py-20">
      <div className="max-w-7xl mx-auto px-6">
        <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
          {t('ecosystem.title')}
        </h2>
        <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
          {t('ecosystem.description')}
        </p>

        <div className="grid md:grid-cols-2 lg:grid-cols-5 gap-6">
          {ecosystemItems.map((item) => (
            <a
              key={item.key}
              href={item.href}
              {...getLinkProps(item.href)}
              className="card card-interactive text-center group"
            >
              <div className="w-12 h-12 rounded-lg bg-[var(--color-bg-hover)] flex items-center justify-center mx-auto mb-4 group-hover:bg-[var(--color-primary-muted)] transition-colors">
                <item.icon size={24} className="text-[var(--color-text-secondary)] group-hover:text-[var(--color-primary)] transition-colors" aria-hidden="true" />
              </div>
              <h3 className="text-lg font-semibold text-[var(--color-text)] mb-2 flex items-center justify-center gap-2">
                {t(`ecosystem.${item.key}.title`)}
                <ExternalLink size={14} className="text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity" aria-hidden="true" />
              </h3>
              <p className="text-sm text-[var(--color-text-secondary)]">
                {t(`ecosystem.${item.key}.description`)}
              </p>
            </a>
          ))}
        </div>
      </div>
    </section>
  );
}
