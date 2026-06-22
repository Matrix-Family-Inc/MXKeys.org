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

// Public barrel for shared UI primitives. Everything exported here is
// imported by at least one widget/feature; dead primitives are a
// documented anti-pattern (see ADR-0009).
export { Logo } from './logo';
export { TextField } from './text-field';
export type { TextFieldProps } from './text-field';
export { EcosystemBadge } from './ecosystem-badge';
export type { EcosystemBadgeProps } from './ecosystem-badge';
