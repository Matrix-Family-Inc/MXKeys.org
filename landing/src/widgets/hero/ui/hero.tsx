/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

import { useTranslation } from 'react-i18next';
import { Logo } from '@/shared/ui';
import { HeroBackground } from './hero-background';
import { HeroBadges } from './hero-badges';
import { HeroActions } from './hero-actions';

/**
 * Hero composition root. Owns layout only; every sub-concern
 * (background pattern, badges row, actions row) lives in its own
 * file so each unit stays focused and individually testable.
 */
export function HeroSection() {
  const { t } = useTranslation();

  return (
    <section className="relative overflow-hidden">
      <HeroBackground />

      <div className="relative max-w-7xl mx-auto px-6 py-24 md:py-32">
        <div className="text-center">
          <div className="flex justify-center mb-8">
            <Logo size={120} />
          </div>

          <h1 className="text-5xl md:text-6xl font-bold text-text mb-4">
            {t('hero.title')}
          </h1>
          <p className="text-2xl md:text-3xl text-primary font-medium mb-4">
            {t('hero.subtitle')}
          </p>
          <p className="text-lg text-text-muted font-mono mb-6">
            {t('hero.tagline')}
          </p>

          <p className="text-lg text-text-secondary max-w-3xl mx-auto mb-4">
            {t('hero.description')}
          </p>

          <p className="text-sm text-text-muted font-medium mb-8">
            {t('hero.trust')}
          </p>

          <HeroBadges />
          <HeroActions />
        </div>
      </div>
    </section>
  );
}
