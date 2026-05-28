/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
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
