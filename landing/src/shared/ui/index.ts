/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

// Public barrel for shared UI primitives. Everything exported here is
// imported by at least one widget/feature; dead primitives are a
// documented anti-pattern (see ADR-0009).
export { Logo } from './logo';
export { TextField } from './text-field';
export type { TextFieldProps } from './text-field';
export { EcosystemBadge } from './ecosystem-badge';
export type { EcosystemBadgeProps } from './ecosystem-badge';
