/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { forwardRef } from 'react';
import type { AnchorHTMLAttributes } from 'react';
import { cn } from '../lib/cn';
import { getLinkProps, isInternalLink } from '../config/urls';

export type ExternalLinkProps = AnchorHTMLAttributes<HTMLAnchorElement> & {
  href: string;
};

/**
 * ExternalLink renders an anchor tag with correct target/rel semantics for
 * cross-site navigation. Internal links (same-origin or same-ecosystem
 * hosts in urls.ts) stay in the same tab; external links open in a new tab
 * with rel="noopener noreferrer" to prevent reverse tabnabbing.
 */
export const ExternalLink = forwardRef<HTMLAnchorElement, ExternalLinkProps>(
  ({ href, className, children, ...rest }, ref) => {
    const linkProps = getLinkProps(href);
    return (
      <a
        ref={ref}
        href={href}
        className={cn('underline-offset-4 hover:underline', className)}
        data-internal={isInternalLink(href) ? 'true' : 'false'}
        {...linkProps}
        {...rest}
      >
        {children}
      </a>
    );
  },
);
ExternalLink.displayName = 'ExternalLink';
