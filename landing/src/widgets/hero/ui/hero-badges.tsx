/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

import { useTranslation } from 'react-i18next';
import { EcosystemBadge } from '@/shared/ui';
import { URLS } from '@/shared/config/urls';
import { APP_VERSION_TAG } from '@/shared/config/version';

/**
 * Row of hero pills: live-status indicator, running release tag,
 * and ecosystem cross-links. Each pill has a single responsibility
 * so the set stays easy to extend (e.g. future "Matrix v1.16 ready"
 * badge) without mutating hero.tsx itself.
 */
export function HeroBadges() {
  const { t } = useTranslation();

  return (
    <div className="flex flex-wrap items-center justify-center gap-3 mb-10">
      <span className="badge badge-primary flex items-center gap-2">
        <span
          className="w-2 h-2 bg-primary rounded-full animate-pulse"
          aria-hidden="true"
        />
        {t('status.online')}
      </span>
      <span className="badge badge-primary">{APP_VERSION_TAG}</span>
      <EcosystemBadge href={URLS.matrixFamily} label={t('hero.matrixFamilyBadge')} />
      <EcosystemBadge href={URLS.hushmeApp} label={t('hero.hushmeBadge')} />
    </div>
  );
}
