/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// Public barrel for the shared UI kit. Widgets/features import from
// '@/shared/ui', not from leaf files, so the kit surface stays stable as
// primitives are reorganized internally.
export { Button } from './button';
export type { ButtonProps } from './button';

export { Container } from './container';
export type { ContainerProps } from './container';

export { ExternalLink } from './external-link';
export type { ExternalLinkProps } from './external-link';

export { Logo } from './logo';
