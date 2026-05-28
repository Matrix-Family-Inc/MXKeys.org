/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

//go:build !linux

package keyprovider

// mlockBestEffort is a no-op on non-Linux platforms. macOS and Windows
// have their own mlock-equivalent APIs (mlock and VirtualLock) but
// the notary's documented deployment target is Linux containers and
// Linux VMs, so we do not attempt to lock elsewhere.
//
// Operators running on other platforms see the key stored in normal
// process memory, which is the same posture all prior releases had.
func mlockBestEffort(b []byte) error { _ = b; return nil }
