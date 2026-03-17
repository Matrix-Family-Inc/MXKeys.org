/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Thu 06 Feb 2026 UTC
 * Status: Updated - Full server discovery + canonical JSON verification
 * Contact: @support:matrix.family
 *
 * Fetches and verifies keys from remote Matrix servers.
 * Uses Resolver for proper server name discovery (well-known, SRV, fallback).
 * Uses canonical JSON from mautrix for signature verification per Matrix spec.
 */

package keys

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"

	"mxkeys/internal/zero/canonical"
	"mxkeys/internal/zero/log"
)

const (
	maxConcurrentFetches = 50
	defaultRetryAttempts = 3
	retryBackoffBase     = 200 * time.Millisecond
	maxFederationBody    = 1 << 20 // 1MB per upstream response
)

// TrustedNotaryKey holds a pinned notary key
type TrustedNotaryKey struct {
	ServerName string
	KeyID      string
	PublicKey  []byte
}

// Fetcher fetches keys from remote servers using proper server discovery.
type Fetcher struct {
	client          *http.Client
	resolver        *Resolver
	fallbackServers []string
	fetchSem        *semaphore.Weighted
	circuitBreaker  *CircuitBreaker
	trustedNotaries map[string]TrustedNotaryKey
	retryAttempts   int
}

// FetcherConfig holds fetcher configuration
type FetcherConfig struct {
	FallbackServers []string
	Timeout         time.Duration
	TrustedNotaries []TrustedNotaryKey
	RetryAttempts   int
}

// NewFetcher creates a new remote key fetcher with server discovery support.
func NewFetcher(fallbackServers []string, timeout time.Duration) *Fetcher {
	return NewFetcherWithConfig(FetcherConfig{
		FallbackServers: fallbackServers,
		Timeout:         timeout,
		RetryAttempts:   defaultRetryAttempts,
	})
}

// NewFetcherWithConfig creates a new fetcher with full configuration.
func NewFetcherWithConfig(cfg FetcherConfig) *Fetcher {
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = defaultRetryAttempts
	}

	trustedMap := make(map[string]TrustedNotaryKey)
	for _, tn := range cfg.TrustedNotaries {
		trustedMap[tn.ServerName] = tn
	}

	return &Fetcher{
		client: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion:         tls.VersionTLS12,
					InsecureSkipVerify: false,
				},
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				MaxConnsPerHost:       20,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		resolver:        NewResolver(),
		fallbackServers: cfg.FallbackServers,
		fetchSem:        semaphore.NewWeighted(maxConcurrentFetches),
		circuitBreaker:  NewCircuitBreaker(5, 60*time.Second),
		trustedNotaries: trustedMap,
		retryAttempts:   cfg.RetryAttempts,
	}
}

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

	// Record failure for circuit breaker
	f.circuitBreaker.RecordFailure(serverName)

	log.Debug("Direct key fetch failed, trying fallback servers",
		"server", serverName,
		"error", err,
	)

	// Try fallback servers (like matrix.org)
	for _, fallback := range f.fallbackServers {
		resp, err := f.fetchFromNotary(ctx, fallback, serverName)
		if err == nil {
			return resp, nil
		}
		log.Debug("Fallback key fetch failed",
			"fallback", fallback,
			"server", serverName,
			"error", err,
		)
	}

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

// isRetryableError checks if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
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

	resp, err := f.client.Do(req)
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

	resp, err := f.client.Do(req)
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

// verifyNotarySignature verifies the notary's signature if we have a pinned key
func (f *Fetcher) verifyNotarySignature(notary string, resp *ServerKeysResponse) error {
	trusted, hasPinned := f.trustedNotaries[notary]
	if !hasPinned {
		// No pinned key, trust based on TLS
		return nil
	}

	// Check if notary signed this response
	notarySigs, ok := resp.Signatures[notary]
	if !ok {
		return fmt.Errorf("notary %s did not sign the response", notary)
	}

	sig, ok := notarySigs[trusted.KeyID]
	if !ok {
		return fmt.Errorf("notary %s did not sign with pinned key %s", notary, trusted.KeyID)
	}

	// Decode signature
	sigBytes, err := base64.RawStdEncoding.DecodeString(sig)
	if err != nil {
		return fmt.Errorf("failed to decode notary signature: %w", err)
	}

	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid notary signature length: %d", len(sigBytes))
	}

	if len(trusted.PublicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid pinned public key length: %d", len(trusted.PublicKey))
	}

	toVerify := map[string]interface{}{
		"server_name":     resp.ServerName,
		"valid_until_ts":  resp.ValidUntilTS,
		"verify_keys":     resp.VerifyKeys,
		"old_verify_keys": resp.OldVerifyKeys,
	}

	// The notary signs the object that includes all existing signatures
	// except its own just-added signature.
	if resp.Signatures != nil {
		signatures := make(map[string]map[string]string, len(resp.Signatures))
		for signer, signerSigs := range resp.Signatures {
			if signer == notary {
				continue
			}
			copied := make(map[string]string, len(signerSigs))
			for keyID, value := range signerSigs {
				copied[keyID] = value
			}
			signatures[signer] = copied
		}
		if len(signatures) > 0 {
			toVerify["signatures"] = signatures
		}
	}

	canonicalBytes, err := canonical.Marshal(toVerify)
	if err != nil {
		return fmt.Errorf("failed to canonicalize notary payload: %w", err)
	}

	if !ed25519.Verify(ed25519.PublicKey(trusted.PublicKey), canonicalBytes, sigBytes) {
		return ErrNotaryKeyMismatch
	}

	log.Debug("Notary signature verified",
		"notary", notary,
		"key_id", trusted.KeyID,
	)

	return nil
}

// verifySelfSignature verifies that server signed its own keys
// using Matrix canonical JSON for signature verification.
func (f *Fetcher) verifySelfSignature(resp *ServerKeysResponse, rawJSON []byte) error {
	// Verify required fields
	if resp.ServerName == "" {
		return fmt.Errorf("server_name is empty")
	}
	if len(resp.VerifyKeys) == 0 {
		return fmt.Errorf("verify_keys is empty")
	}
	if resp.ValidUntilTS <= time.Now().UnixMilli() {
		return fmt.Errorf("valid_until_ts is in the past")
	}
	if resp.Signatures == nil {
		return fmt.Errorf("no signatures in response")
	}

	serverSigs, ok := resp.Signatures[resp.ServerName]
	if !ok {
		return fmt.Errorf("no self-signature found for %s", resp.ServerName)
	}

	// Remove signatures and unsigned for canonical JSON
	var parsed map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.UseNumber()
	if err := dec.Decode(&parsed); err != nil {
		return fmt.Errorf("failed to parse JSON for verification: %w", err)
	}
	delete(parsed, "signatures")
	delete(parsed, "unsigned")

	// Re-encode without signatures
	stripped, err := json.Marshal(parsed)
	if err != nil {
		return fmt.Errorf("failed to re-encode JSON: %w", err)
	}

	// Convert to Matrix canonical JSON (sorted keys, compact)
	canonicalBytes, err := canonical.JSON(stripped)
	if err != nil {
		return fmt.Errorf("failed to canonicalize JSON: %w", err)
	}

	// Verify at least one signature matches
	for keyID, sigBase64 := range serverSigs {
		verifyKey, ok := resp.VerifyKeys[keyID]
		if !ok {
			// Check old_verify_keys
			if oldKey, ok := resp.OldVerifyKeys[keyID]; ok {
				pubKeyBytes, err := base64.RawStdEncoding.DecodeString(oldKey.Key)
				if err != nil {
					continue
				}
				// Verify ed25519 public key length (32 bytes)
				if len(pubKeyBytes) != ed25519.PublicKeySize {
					continue
				}
				sig, err := base64.RawStdEncoding.DecodeString(sigBase64)
				if err != nil {
					continue
				}
				// Verify ed25519 signature length (64 bytes)
				if len(sig) != ed25519.SignatureSize {
					continue
				}
				if ed25519.Verify(ed25519.PublicKey(pubKeyBytes), canonicalBytes, sig) {
					return nil
				}
			}
			continue
		}

		pubKeyBytes, err := base64.RawStdEncoding.DecodeString(verifyKey.Key)
		if err != nil {
			continue
		}

		// Verify ed25519 public key length (32 bytes)
		if len(pubKeyBytes) != ed25519.PublicKeySize {
			continue
		}

		sig, err := base64.RawStdEncoding.DecodeString(sigBase64)
		if err != nil {
			continue
		}

		// Verify ed25519 signature length (64 bytes)
		if len(sig) != ed25519.SignatureSize {
			continue
		}

		if ed25519.Verify(ed25519.PublicKey(pubKeyBytes), canonicalBytes, sig) {
			return nil // Valid signature found
		}
	}

	return fmt.Errorf("no valid self-signature for %s", resp.ServerName)
}
