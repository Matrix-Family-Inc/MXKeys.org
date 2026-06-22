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

import { getLinkProps } from '@/shared/config/urls';

/**
 * Pill-style hyperlink used to cross-link between Matrix Family
 * ecosystem sites (e.g. Matrix Family hub, HushMe). Kept in shared
 * so every widget that displays an ecosystem affiliation renders
 * the same visual contract (accent pill + hover opacity).
 *
 * `href` is passed through `getLinkProps` so internal-ecosystem
 * links stay same-tab while external links open with
 * `target="_blank" rel="noopener noreferrer"`.
 */
export interface EcosystemBadgeProps {
  href: string;
  label: string;
}

export function EcosystemBadge({ href, label }: EcosystemBadgeProps) {
  return (
    <a
      href={href}
      {...getLinkProps(href)}
      className="badge badge-accent hover:opacity-80 transition-opacity"
    >
      {label}
    </a>
  );
}
