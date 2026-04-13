/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"mxkeys/internal/zero/log"
)

// FetchServerKeys fetches all keys from a server using proper discovery.
func (f *Fetcher) FetchServerKeys(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	// Check context first
	if ctx.Err() != nil {
		return nil, &KeyError{Op: "fetch", ServerName: serverName, Err: ErrContextCanceled}
	}

	// Check circuit breaker
	if !f.circuitBreaker.Allow(serverName) {
		return nil, &KeyError{Op: "fetch", ServerName: serverName, Err: ErrCircuitOpen}
	}

	// Acquire semaphore to limit concurrent outbound fetches
	if err := f.fetchSem.Acquire(ctx, 1); err != nil {
		return nil, &KeyError{Op: "fetch", ServerName: serverName, Err: fmt.Errorf("%w: %v", ErrConcurrencyLimit, err)}
	}
	defer f.fetchSem.Release(1)

	// Try direct fetch first (with full server discovery) with retry
	resp, err := f.fetchDirectWithRetry(ctx, serverName)
	if err == nil {
		f.circuitBreaker.RecordSuccess(serverName)
		return resp, nil
	}

	log.Debug("Direct key fetch failed, trying fallback servers",
		"server", serverName,
		"error", err,
	)

	// Try fallback servers (like matrix.org)
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

	// Record failure only when the full operation fails.
	f.circuitBreaker.RecordFailure(serverName)
	return nil, NewFetchError(serverName, fmt.Errorf("all sources failed"))
}

// fetchDirectWithRetry fetches with retry logic for transient errors
func (f *Fetcher) fetchDirectWithRetry(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	var lastErr error

	for attempt := 0; attempt < f.retryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 200ms, 400ms, 800ms...
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

		// Don't retry permanent errors
		if IsPermanentError(err) {
			return nil, err
		}

		// Check if error is retryable (network errors)
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

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "temporary failure")
}

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

// fetchDirect fetches keys directly from server using resolver.
func (f *Fetcher) fetchDirect(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	resolved, err := f.resolver.ResolveServerName(ctx, serverName)
	if err != nil {
		return nil, NewResolveError(serverName, err)
	}
	if err := f.rejectPrivateAddress(ctx, serverName, resolved); err != nil {
		return nil, NewResolveError(serverName, err)
	}

	url := fmt.Sprintf("%s/_matrix/key/v2/server", resolved.URL())

	log.Debug("Fetching server keys directly",
		"server", serverName,
		"resolved_host", resolved.Host,
		"resolved_port", resolved.Port,
		"url", url,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Host = resolved.ServerName

	resp, err := f.clientForResolved(resolved).Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "tls") || strings.Contains(err.Error(), "certificate") {
			recordUpstreamFailure(UpstreamFailureTLS)
		} else if strings.Contains(err.Error(), "timeout") {
			recordUpstreamFailure(UpstreamFailureTimeout)
		} else {
			recordUpstreamFailure(UpstreamFailureHTTP)
		}
		return nil, fmt.Errorf("HTTP request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		recordUpstreamFailure(UpstreamFailureHTTP)
		body, _ := readLimitedBody(resp.Body, maxFederationBody)
		return nil, fmt.Errorf("key fetch from %s returned status %d: %s", url, resp.StatusCode, string(body))
	}

	body, err := readLimitedBody(resp.Body, maxFederationBody)
	if err != nil {
		return nil, err
	}

	var keysResp ServerKeysResponse
	if err := json.Unmarshal(body, &keysResp); err != nil {
		return nil, fmt.Errorf("failed to parse response from %s: %w", url, err)
	}

	// Verify server name matches
	if keysResp.ServerName != serverName {
		recordUpstreamFailure(UpstreamFailureServerMismatch)
		return nil, &KeyError{
			Op:         "validate",
			ServerName: serverName,
			Err:        fmt.Errorf("%w: expected %s, got %s", ErrServerNameMismatch, serverName, keysResp.ServerName),
		}
	}

	// Verify self-signature using canonical JSON
	if err := f.verifySelfSignature(&keysResp, body); err != nil {
		recordUpstreamFailure(UpstreamFailureInvalidSignature)
		return nil, NewSignatureError(serverName, err)
	}

	log.Info("Successfully fetched server keys",
		"server", serverName,
		"keys_count", len(keysResp.VerifyKeys),
		"valid_until", time.UnixMilli(keysResp.ValidUntilTS).Format(time.RFC3339),
	)

	return &keysResp, nil
}

// fetchFromNotary fetches keys from a notary (perspective) server.
func (f *Fetcher) fetchFromNotary(ctx context.Context, notary, serverName string) (*ServerKeysResponse, error) {
	resolved, err := f.resolver.ResolveServerName(ctx, notary)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve notary %s: %w", notary, err)
	}
	if err := f.rejectPrivateAddress(ctx, notary, resolved); err != nil {
		return nil, fmt.Errorf("failed private-address check for notary %s: %w", notary, err)
	}

	url := fmt.Sprintf("%s/_matrix/key/v2/query", resolved.URL())

	reqBody := KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{
			serverName: {},
		},
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqJSON)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Host = resolved.ServerName

	resp, err := f.clientForResolved(resolved).Do(req)
	if err != nil {
		return nil, fmt.Errorf("notary query to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := readLimitedBody(resp.Body, maxFederationBody)
		return nil, fmt.Errorf("notary %s returned status %d: %s", notary, resp.StatusCode, string(body))
	}

	body, err := readLimitedBody(resp.Body, maxFederationBody)
	if err != nil {
		return nil, err
	}

	var notaryResp KeyQueryResponse
	if err := json.Unmarshal(body, &notaryResp); err != nil {
		return nil, fmt.Errorf("failed to parse notary response from %s: %w", notary, err)
	}

	for _, keys := range notaryResp.ServerKeys {
		if keys.ServerName == serverName {
			// Verify notary signature if we have a pinned key
			if err := f.verifyNotarySignature(notary, &keys); err != nil {
				return nil, err
			}
			return &keys, nil
		}
	}

	return nil, fmt.Errorf("server %s not found in notary %s response", serverName, notary)
}
