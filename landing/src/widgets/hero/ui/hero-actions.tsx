/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

import { useTranslation } from 'react-i18next';
import { ArrowRight, Github } from 'lucide-react';
import { EXTERNAL } from '@/shared/config/urls';

/**
 * Call-to-action row under the hero copy. Three equal-weight
 * actions: learn-more (anchor to #about), view-API (anchor to #api),
 * and GitHub (external). GitHub is the only external link here, so
 * it carries target/rel locally instead of routing through
 * `getLinkProps` - the other two are always in-page anchors.
 */
export function HeroActions() {
  const { t } = useTranslation();

  return (
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
  );
}
