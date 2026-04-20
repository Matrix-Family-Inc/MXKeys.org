/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sat Mar 15 2026 UTC
 * Status: Created
 */

package keys

import (
	"errors"
	"fmt"
)

var (
	ErrResolveFailed      = errors.New("server resolution failed")
	ErrFetchFailed        = errors.New("key fetch failed")
	ErrInvalidResponse    = errors.New("invalid server response")
	ErrSignatureInvalid   = errors.New("signature verification failed")
	ErrServerNameMismatch = errors.New("server name mismatch")
	ErrKeysExpired        = errors.New("keys are expired")
	ErrCriteriaNotMet     = errors.New("criteria not satisfied")
	ErrConcurrencyLimit   = errors.New("concurrency limit reached")
	ErrContextCanceled    = errors.New("context canceled")
	ErrCircuitOpen        = errors.New("circuit breaker open")
	ErrNotaryKeyMismatch  = errors.New("notary key does not match pinned key")
)

type KeyError struct {
	Op         string // operation that failed
	ServerName string // server involved
	Err        error  // underlying error
}

func (e *KeyError) Error() string {
	if e.ServerName != "" {
		return fmt.Sprintf("%s [%s]: %v", e.Op, e.ServerName, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *KeyError) Unwrap() error {
	return e.Err
}

func (e *KeyError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

func NewResolveError(serverName string, err error) error {
	return &KeyError{
		Op:         "resolve",
		ServerName: serverName,
		Err:        fmt.Errorf("%w: %v", ErrResolveFailed, err),
	}
}

func NewFetchError(serverName string, err error) error {
	return &KeyError{
		Op:         "fetch",
		ServerName: serverName,
		Err:        fmt.Errorf("%w: %v", ErrFetchFailed, err),
	}
}

func NewValidationError(serverName string, err error) error {
	return &KeyError{
		Op:         "validate",
		ServerName: serverName,
		Err:        fmt.Errorf("%w: %v", ErrInvalidResponse, err),
	}
}

func NewSignatureError(serverName string, err error) error {
	return &KeyError{
		Op:         "verify",
		ServerName: serverName,
		Err:        fmt.Errorf("%w: %v", ErrSignatureInvalid, err),
	}
}

func IsTemporaryError(err error) bool {
	var keyErr *KeyError
	if errors.As(err, &keyErr) {
		return errors.Is(keyErr.Err, ErrFetchFailed) ||
			errors.Is(keyErr.Err, ErrResolveFailed) ||
			errors.Is(keyErr.Err, ErrConcurrencyLimit)
	}
	return false
}

func IsPermanentError(err error) bool {
	var keyErr *KeyError
	if errors.As(err, &keyErr) {
		return errors.Is(keyErr.Err, ErrSignatureInvalid) ||
			errors.Is(keyErr.Err, ErrServerNameMismatch) ||
			errors.Is(keyErr.Err, ErrInvalidResponse) ||
			errors.Is(keyErr.Err, ErrNotaryKeyMismatch)
	}
	return false
}
