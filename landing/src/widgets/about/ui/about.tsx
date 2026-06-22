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
import { Lock, Key } from 'lucide-react';

export function AboutSection() {
  const { t } = useTranslation();

  return (
    <section id="about" className="py-20 bg-bg-surface">
      <div className="max-w-7xl mx-auto px-6">
        <div className="max-w-3xl mx-auto text-center mb-12">
          <h2 className="text-3xl font-bold text-text mb-4">
            {t('about.title')}
          </h2>
          <p className="text-lg text-text-secondary">
            {t('about.description')}
          </p>
        </div>

        <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
          <div className="card">
            <div className="w-12 h-12 rounded-lg bg-[rgba(244,67,54,0.15)] flex items-center justify-center mb-4">
              <Lock size={24} className="text-[#f44336]" aria-hidden="true" />
            </div>
            <h3 className="text-xl font-semibold text-text mb-3">
              {t('about.problem.title')}
            </h3>
            <p className="text-text-secondary">
              {t('about.problem.description')}
            </p>
          </div>

          <div className="card">
            <div className="w-12 h-12 rounded-lg bg-primary-muted flex items-center justify-center mb-4">
              <Key size={24} className="text-primary" aria-hidden="true" />
            </div>
            <h3 className="text-xl font-semibold text-text mb-3">
              {t('about.solution.title')}
            </h3>
            <p className="text-text-secondary">
              {t('about.solution.description')}
            </p>
          </div>
        </div>
      </div>
    </section>
  );
}
