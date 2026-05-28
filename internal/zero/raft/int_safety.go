/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package raft

import (
	"fmt"
	"math"
)

// lenUint32 returns len(b) as uint32 after verifying the length is
// within both a caller-supplied cap and math.MaxUint32. Used by WAL
// and snapshot writers that embed the payload length as a 4-byte
// little-endian field. Callers pass a descriptive label so the error
// message is operator-friendly.
func lenUint32(label string, b []byte, max int) (uint32, error) {
	n := len(b)
	if n < 0 {
		return 0, fmt.Errorf("%s: negative length %d", label, n)
	}
	if n > max {
		return 0, fmt.Errorf("%s: length %d exceeds cap %d", label, n, max)
	}
	if n > math.MaxUint32 {
		return 0, fmt.Errorf("%s: length %d exceeds uint32 range", label, n)
	}
	return uint32(n), nil
}

// offsetToSlot converts an absolute Raft log index into a slice index
// relative to logOffset. Returns ok=false for indices below the
// current compaction floor (<= logOffset) and for indices beyond the
// in-memory tail.
//
// The arithmetic centralises the uint64/int conversion so no caller
// has to reason about integer safety at the use site.
func offsetToSlot(absoluteIndex, logOffset uint64, logLen int) (int, bool) {
	if absoluteIndex == 0 || absoluteIndex <= logOffset {
		return -1, false
	}
	if logLen < 0 {
		return -1, false
	}
	diff := absoluteIndex - logOffset - 1
	// A slice is indexed by int; anything wider than math.MaxInt is
	// not addressable in Go. The bound check here is the single
	// narrowing site in the package.
	if diff >= uint64(logLen) {
		return -1, false
	}
	if diff > math.MaxInt {
		return -1, false
	}
	return int(diff), true
}
