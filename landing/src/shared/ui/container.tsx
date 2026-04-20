/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import type { HTMLAttributes } from 'react';
import { cn } from '../lib/cn';

export type ContainerProps = HTMLAttributes<HTMLDivElement> & {
  as?: 'div' | 'section' | 'main' | 'header' | 'footer';
};

/**
 * Container is the canonical max-width wrapper used by every widget.
 * Centralizing this here keeps the horizontal rhythm consistent and
 * avoids scattering `max-w-7xl mx-auto px-6` across every section.
 */
export function Container({
  as: Tag = 'div',
  className,
  children,
  ...rest
}: ContainerProps) {
  return (
    <Tag className={cn('max-w-7xl mx-auto px-6', className)} {...rest}>
      {children}
    </Tag>
  );
}
