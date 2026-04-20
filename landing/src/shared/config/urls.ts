/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

import { siteURL } from './env';

/**
 * Ecosystem URLs used across the landing. mxkeys is derived from the
 * VITE_SITE_URL env var so an operator deploying their own branded notary
 * gets correct self-references without forking the code.
 */
export const URLS = {
  hushmeOnline: 'https://hushme.online',
  hushmeApp: 'https://hushme.app',
  hushmeStore: 'https://hushme.store',
  hushmeSpace: 'https://hushme.space',
  matrixFamily: 'https://matrix.family',
  appsGateway: 'https://apps.hushme.app',
  mxkeys: siteURL,
  mxcore: 'https://mxcore.tech',
  mfos: 'https://mfos.tech',
} as const;

export const DOCS = {
  gettingStarted: `${URLS.mfos}/docs/getting-started`,
  api: `${URLS.mfos}/docs/api`,
  gateway: `${URLS.mfos}/docs/gateway`,
  publishing: `${URLS.mfos}/docs/publishing`,
} as const;

export const MATRIX_CONTACTS = {
  support: { id: '@support:matrix.family', href: `${URLS.hushmeSpace}/@support:matrix.family` },
  developer: { id: '@dev:matrix.family', href: `${URLS.hushmeSpace}/@dev:matrix.family` },
  devChat: { id: '#dev:matrix.family', href: `${URLS.hushmeSpace}/%23dev:matrix.family` },
  announcements: { id: '#announcements:matrix.family', href: `${URLS.hushmeSpace}/%23announcements:matrix.family` },
} as const;

export const EXTERNAL = {
  github: 'https://github.com/matrixfamily/MXKeys.org',
} as const;

/**
 * INTERNAL_DOMAINS covers the ecosystem hosts that should open in the
 * same tab even when the href uses an absolute URL. Matched by substring,
 * which is intentionally broad so subdomains count as internal.
 */
const INTERNAL_DOMAINS = [
  'hushme.online',
  'hushme.app',
  'hushme.store',
  'hushme.space',
  'matrix.family',
  'apps.hushme.app',
  'mxcore.tech',
  'mfos.tech',
] as const;

/** isInternalLink reports whether url points to the ecosystem or the same origin. */
export function isInternalLink(url: string): boolean {
  if (url.startsWith('/') || url.startsWith('#')) return true;
  if (url.startsWith(siteURL)) return true;
  return INTERNAL_DOMAINS.some((domain) => url.includes(domain));
}

/** getLinkProps returns {target,rel} for anchors based on link scope. */
export function getLinkProps(url: string): { target?: string; rel?: string } {
  if (isInternalLink(url)) return {};
  return { target: '_blank', rel: 'noopener noreferrer' };
}
