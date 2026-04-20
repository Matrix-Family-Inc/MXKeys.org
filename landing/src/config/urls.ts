/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: 2026-02-05 UTC
 * Status: Created
 */

// Ecosystem URLs
export const URLS = {
  hushmeOnline: 'https://hushme.online',
  hushmeApp: 'https://hushme.app',
  hushmeStore: 'https://hushme.store',
  hushmeSpace: 'https://hushme.space',
  matrixFamily: 'https://matrix.family',
  appsGateway: 'https://apps.hushme.app',
  mxkeys: 'https://mxkeys.org',
  mxcore: 'https://mxcore.tech',
  mfos: 'https://mfos.tech',
};

// Documentation links
export const DOCS = {
  gettingStarted: `${URLS.mfos}/docs/getting-started`,
  api: `${URLS.mfos}/docs/api`,
  gateway: `${URLS.mfos}/docs/gateway`,
  publishing: `${URLS.mfos}/docs/publishing`,
};

// Matrix contacts (hushme.space format, not matrix.to)
export const MATRIX_CONTACTS = {
  support: { id: '@support:matrix.family', href: `${URLS.hushmeSpace}/@support:matrix.family` },
  developer: { id: '@dev:matrix.family', href: `${URLS.hushmeSpace}/@dev:matrix.family` },
  devChat: { id: '#dev:matrix.family', href: `${URLS.hushmeSpace}/%23dev:matrix.family` },
  announcements: { id: '#announcements:matrix.family', href: `${URLS.hushmeSpace}/%23announcements:matrix.family` },
};

// External links (new tab)
export const EXTERNAL = {
  github: 'https://github.com/matrixfamily/MXKeys.org',
};

const INTERNAL_DOMAINS = [
  'hushme.online', 'hushme.app', 'hushme.store', 'hushme.space',
  'matrix.family', 'apps.hushme.app', 'mxkeys.org', 'mxcore.tech', 'mfos.tech',
];

export function isInternalLink(url: string): boolean {
  if (url.startsWith('/') || url.startsWith('#')) return true;
  return INTERNAL_DOMAINS.some(domain => url.includes(domain));
}

export function getLinkProps(url: string): { target?: string; rel?: string } {
  if (isInternalLink(url)) return {};
  return { target: '_blank', rel: 'noopener noreferrer' };
}
