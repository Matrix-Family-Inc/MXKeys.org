/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

//go:build linux

package keyprovider

import (
	"syscall"
)

// mlockBestEffort attempts to lock the backing pages of b into RAM so
// the seed bytes do not end up in a swap file. Errors are returned to
// the caller (they land in a WARN-level log, not a hard failure):
// mlock requires CAP_IPC_LOCK or a sufficiently large RLIMIT_MEMLOCK
// and may legitimately be unavailable in some container runtimes.
//
// The corresponding unlock is best-effort too; when the provider
// holds the key for the whole process lifetime the locked pages
// stay locked, which is the desired behaviour.
func mlockBestEffort(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	return syscall.Mlock(b)
}

