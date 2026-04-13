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
import { ArrowRight, Github } from 'lucide-react';
import { Logo } from './Logo';
import { URLS, EXTERNAL, getLinkProps } from '../config/urls';

export function HeroSection() {
  const { t } = useTranslation();

  return (
    <section className="relative overflow-hidden">
      <div className="absolute inset-0 opacity-[0.03]">
        <div
          className="absolute inset-0"
          style={{
            backgroundImage: `repeating-linear-gradient(
              -15deg,
              transparent,
              transparent 70px,
              rgba(61, 153, 112, 0.15) 70px,
              rgba(61, 153, 112, 0.15) 71px
            )`,
          }}
        />
      </div>

      <div className="relative max-w-7xl mx-auto px-6 py-24 md:py-32">
        <div className="text-center">
          <div className="flex justify-center mb-8">
            <Logo size={120} animated />
          </div>

          <h1 className="text-5xl md:text-6xl font-bold text-[var(--color-text)] mb-4">
            {t('hero.title')}
          </h1>
          <p className="text-2xl md:text-3xl text-[var(--color-primary)] font-medium mb-4">
            {t('hero.subtitle')}
          </p>
          <p className="text-lg text-[var(--color-text-muted)] font-mono mb-6">
            {t('hero.tagline')}
          </p>

          <p className="text-lg text-[var(--color-text-secondary)] max-w-3xl mx-auto mb-4">
            {t('hero.description')}
          </p>

          <p className="text-sm text-[var(--color-text-muted)] font-medium mb-8">
            {t('hero.trust')}
          </p>

          <div className="flex flex-wrap items-center justify-center gap-3 mb-10">
            <span className="badge badge-primary flex items-center gap-2">
              <span className="w-2 h-2 bg-[var(--color-primary)] rounded-full animate-pulse" />
              {t('status.online')}
            </span>
            <span className="badge badge-primary">v0.2.0</span>
            <a
              href={URLS.matrixFamily}
              {...getLinkProps(URLS.matrixFamily)}
              className="badge badge-accent hover:opacity-80 transition-opacity"
            >
              {t('hero.matrixFamilyBadge')}
            </a>
            <a
              href={URLS.hushmeApp}
              {...getLinkProps(URLS.hushmeApp)}
              className="badge badge-accent hover:opacity-80 transition-opacity"
            >
              {t('hero.hushmeBadge')}
            </a>
          </div>

          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <a href="#about" className="btn btn-primary">
              {t('hero.learnMore')}
              <ArrowRight size={18} aria-hidden="true" />
            </a>
            <a href="#api" className="btn btn-outline">
              {t('hero.viewAPI')}
            </a>
            <a
              href={EXTERNAL.github}
              target="_blank"
              rel="noopener noreferrer"
              className="btn btn-outline"
            >
              <Github size={18} aria-hidden="true" />
              {t('hero.github')}
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}
