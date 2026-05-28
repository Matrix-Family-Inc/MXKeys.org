/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

/**
 * Single source of truth for the MXKeys release tag shown in the
 * landing UI. The value MUST match `Version` in
 * `internal/version/version.go` and the `package.json` of this
 * landing so the hero badge, crash-banner text, and any analytics
 * dimension stay aligned with the binary operators actually run.
 *
 * Release procedure bumps all three together in one commit:
 *   - internal/version/version.go  (Go constant)
 *   - landing/package.json         (npm package version)
 *   - landing/src/shared/config/version.ts (this file)
 */
export const APP_VERSION = '1.0.0';

/** Human-readable release tag with a leading `v`. */
export const APP_VERSION_TAG = `v${APP_VERSION}`;
