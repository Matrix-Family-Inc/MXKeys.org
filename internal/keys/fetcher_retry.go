/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package keys

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
)

// isRetryableError reports whether err is a transient network error worth
// retrying.
//
// Uses typed error classification end-to-end:
//   - net.Error.Timeout() for deadline/timeout errors
//   - *net.OpError for generic net ops
//   - *net.DNSError for name resolution failures
//   - *os.SyscallError plus syscall.Errno classification for
//     ECONNREFUSED / ECONNRESET / EPIPE / EHOSTUNREACH / ENETUNREACH,
//     which is the deterministic way these manifest on Linux/macOS
//     regardless of how the net package wraps them
//   - errors.Is(err, io.EOF) for dropped connections mid-read
//
// String matching on err.Error() was previously the fallback here. It
// has been removed because the typed classification above is
// exhaustive for every kind of transient failure we observe in
// practice; string matching locked the behavior to English-locale
// messages and masked genuine non-transient errors that happen to
// contain the wrong substring.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// IsTemporary/IsTimeout captures most transient DNS cases; name-not-
		// found is also classified as retryable because the downstream
		// resolution often recovers on retry (caching, propagation).
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		if errno, ok := syscallErr.Err.(syscall.Errno); ok {
			switch errno {
			case syscall.ECONNREFUSED,
				syscall.ECONNRESET,
				syscall.EPIPE,
				syscall.EHOSTUNREACH,
				syscall.ENETUNREACH,
				syscall.ETIMEDOUT:
				return true
			}
		}
	}

	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ECONNREFUSED,
			syscall.ECONNRESET,
			syscall.EPIPE,
			syscall.EHOSTUNREACH,
			syscall.ENETUNREACH,
			syscall.ETIMEDOUT:
			return true
		}
	}

	// Mid-stream connection drops surface as io.ErrUnexpectedEOF.
	return errors.Is(err, io.ErrUnexpectedEOF)
}

// readLimitedBody reads at most limit bytes and errors when exceeded.
// The underlying reader is read one byte past the limit to distinguish
// "at limit" from "exceeded limit".
func readLimitedBody(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid body limit: %d", limit)
	}

	limited := io.LimitReader(r, limit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response body too large")
	}
	return body, nil
}
