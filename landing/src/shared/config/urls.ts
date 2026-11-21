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

import { siteURL } from './env';

/**
 * Ecosystem URLs used across the landing. mxkeys is derived from the
 * VITE_SITE_URL env var so an operator deploying their own branded notary
 * gets correct self-references without forking the code.
 */
export const URLS = {
  matrixFamily: 'https://matrix.family',
  mxkeys: siteURL,
  mxcore: 'https://mxcore.tech',
} as const;

export const MATRIX_CONTACTS = {
  support: { id: '@support:matrix.family', href: 'https://matrix.to/#/@support:matrix.family' },
  developer: { id: '@dev:matrix.family', href: 'https://matrix.to/#/@dev:matrix.family' },
  devChat: { id: '#dev:matrix.family', href: 'https://matrix.to/#/#dev:matrix.family' },
  announcements: { id: '#announcements:matrix.family', href: 'https://matrix.to/#/#announcements:matrix.family' },
} as const;

export const EXTERNAL = {
  github: 'https://github.com/Matrix-Family-Inc/MXKeys.org',
} as const;

/**
 * INTERNAL_DOMAINS covers the ecosystem hosts that should open in the
 * same tab even when the href uses an absolute URL. Matched by substring,
 * which is intentionally broad so subdomains count as internal.
 */
const INTERNAL_DOMAINS = [
  'matrix.family',
  'mxcore.tech',
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
