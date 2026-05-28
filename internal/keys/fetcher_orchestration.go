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
	"context"
	"fmt"
	"time"

	"mxkeys/internal/zero/log"
)

// FetchServerKeys fetches all keys from a server using Matrix server discovery.
// Order of attempts: direct fetch with retry -> configured fallback notaries.
// The circuit breaker guards reentry per server; the semaphore caps concurrent
// outbound fetches.
func (f *Fetcher) FetchServerKeys(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	if ctx.Err() != nil {
		return nil, &KeyError{Op: "fetch", ServerName: serverName, Err: ErrContextCanceled}
	}

	if !f.circuitBreaker.Allow(serverName) {
		return nil, &KeyError{Op: "fetch", ServerName: serverName, Err: ErrCircuitOpen}
	}

	if err := f.fetchSem.Acquire(ctx, 1); err != nil {
		return nil, &KeyError{Op: "fetch", ServerName: serverName, Err: fmt.Errorf("%w: %v", ErrConcurrencyLimit, err)}
	}
	defer f.fetchSem.Release(1)

	resp, err := f.fetchDirectWithRetry(ctx, serverName)
	if err == nil {
		f.circuitBreaker.RecordSuccess(serverName)
		return resp, nil
	}

	log.Debug("Direct key fetch failed, trying fallback servers",
		"server", serverName,
		"error", err,
	)

	for _, fallback := range f.fallbackServers {
		resp, err := f.fetchFromNotary(ctx, fallback, serverName)
		if err == nil {
			f.circuitBreaker.RecordSuccess(serverName)
			return resp, nil
		}
		log.Debug("Fallback key fetch failed",
			"fallback", fallback,
			"server", serverName,
			"error", err,
		)
	}

	f.circuitBreaker.RecordFailure(serverName)
	return nil, NewFetchError(serverName, fmt.Errorf("all sources failed"))
}

// fetchDirectWithRetry wraps fetchDirect with exponential backoff for transient
// errors. Permanent errors short-circuit without retry.
func (f *Fetcher) fetchDirectWithRetry(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	var lastErr error

	for attempt := 0; attempt < f.retryAttempts; attempt++ {
		if attempt > 0 {
			backoff := retryBackoffBase * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := f.fetchDirect(ctx, serverName)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if IsPermanentError(err) {
			return nil, err
		}

		if !isRetryableError(err) {
			return nil, err
		}

		log.Debug("Retrying fetch",
			"server", serverName,
			"attempt", attempt+1,
			"max_attempts", f.retryAttempts,
			"error", err,
		)
	}

	return nil, lastErr
}
