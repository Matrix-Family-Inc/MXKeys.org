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
import { ArrowRight, CheckCircle, Database, Globe, Key, Server } from 'lucide-react';

const steps = [
  { key: 'request', icon: Server },
  { key: 'cache', icon: Database },
  { key: 'fetch', icon: Globe },
  { key: 'verify', icon: CheckCircle },
  { key: 'sign', icon: Key },
  { key: 'respond', icon: ArrowRight },
];

export function HowItWorksSection() {
  const { t } = useTranslation();

  return (
    <section id="how-it-works" className="py-20 bg-[var(--color-bg-surface)]">
      <div className="max-w-7xl mx-auto px-6">
        <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
          {t('howItWorks.title')}
        </h2>
        <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
          {t('howItWorks.description')}
        </p>

        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {steps.map((step, index) => (
            <div key={step.key} className="card relative">
              <div className="absolute top-4 right-4">
                <span className="step-number">{index + 1}</span>
              </div>
              <div className="w-10 h-10 rounded-lg bg-[var(--color-primary-muted)] flex items-center justify-center mb-4">
                <step.icon size={20} className="text-[var(--color-primary)]" aria-hidden="true" />
              </div>
              <h3 className="text-lg font-semibold text-[var(--color-text)] mb-2">
                {t(`howItWorks.steps.${step.key}.title`)}
              </h3>
              <p className="text-sm text-[var(--color-text-secondary)]">
                {t(`howItWorks.steps.${step.key}.description`)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
