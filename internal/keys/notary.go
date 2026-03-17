/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 */

package keys

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"mxkeys/internal/zero/canonical"
	"mxkeys/internal/zero/log"
)

// Notary is the main key notary service
type Notary struct {
	serverName    string
	serverKeyID   string
	serverKeyPair ed25519.PrivateKey

	storage *Storage
	fetcher *Fetcher

	cache      map[string]*cachedResponse
	cacheMu    sync.RWMutex
	fetchGroup singleflight.Group

	validityHours int
	cacheTTLHours int
	trustPolicy   *TrustPolicy
}

type cachedResponse struct {
	response   *ServerKeysResponse
	validUntil time.Time
}

// NewNotary creates new notary service
func NewNotary(db *sql.DB, serverName string, keyStoragePath string, validityHours, cacheTTLHours int, fallbackServers []string, fetchTimeout time.Duration) (*Notary, error) {
	storage, err := NewStorage(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	fetcher := NewFetcher(fallbackServers, fetchTimeout)

	n := &Notary{
		serverName:    serverName,
		storage:       storage,
		fetcher:       fetcher,
		cache:         make(map[string]*cachedResponse),
		validityHours: validityHours,
		cacheTTLHours: cacheTTLHours,
	}

	// Load or generate server signing key
	if err := n.initSigningKey(keyStoragePath); err != nil {
		return nil, fmt.Errorf("failed to init signing key: %w", err)
	}

	return n, nil
}

// initSigningKey loads existing key or generates new one
func (n *Notary) initSigningKey(keyStoragePath string) error {
	keyPath := filepath.Join(keyStoragePath, "mxkeys_ed25519.key")

	if err := os.MkdirAll(keyStoragePath, 0700); err != nil {
		return fmt.Errorf("failed to create key storage directory: %w", err)
	}
	if err := os.Chmod(keyStoragePath, 0700); err != nil {
		return fmt.Errorf("failed to enforce key storage directory permissions: %w", err)
	}

	// Try to load existing key
	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) == ed25519.PrivateKeySize {
			n.serverKeyPair = ed25519.PrivateKey(data)
			n.serverKeyID = "ed25519:mxkeys"
			if err := os.Chmod(keyPath, 0600); err != nil {
				return fmt.Errorf("failed to enforce key file permissions: %w", err)
			}
			log.Info("Loaded existing notary signing key", "key_id", n.serverKeyID)
			return nil
		}
	}

	// Generate new key
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	n.serverKeyPair = privateKey
	n.serverKeyID = "ed25519:mxkeys"

	// Save key
	if err := os.WriteFile(keyPath, privateKey, 0600); err != nil {
		return fmt.Errorf("failed to save notary signing key: %w", err)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("failed to enforce key file permissions: %w", err)
	}
	log.Info("Generated and saved new notary signing key", "key_id", n.serverKeyID)

	return nil
}

// GetOwnKeys returns this notary's own public keys
func (n *Notary) GetOwnKeys() *ServerKeysResponse {
	publicKey := n.serverKeyPair.Public().(ed25519.PublicKey)
	publicKeyB64 := base64.RawStdEncoding.EncodeToString(publicKey)

	validUntil := time.Now().Add(time.Duration(n.validityHours) * time.Hour)

	response := &ServerKeysResponse{
		ServerName:   n.serverName,
		ValidUntilTS: validUntil.UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			n.serverKeyID: {Key: publicKeyB64},
		},
		OldVerifyKeys: make(map[string]OldKeyResponse),
	}

	// Sign the response
	n.signResponse(response)

	return response
}

// QueryKeys queries keys for multiple servers (notary functionality)
func (n *Notary) QueryKeys(ctx context.Context, request *KeyQueryRequest) *KeyQueryResponse {
	response := &KeyQueryResponse{
		ServerKeys: make([]ServerKeysResponse, 0),
		Failures:   make(map[string]interface{}),
	}

	for serverName, failure := range n.applyTrustPolicyToRequest(request) {
		response.Failures[serverName] = failure
	}

	for _, serverName := range sortedServerNames(request.ServerKeys) {
		keyCriteria := request.ServerKeys[serverName]
		// Determine minimum valid_until_ts from criteria
		var minValidUntil int64
		for _, criteria := range keyCriteria {
			if criteria.MinimumValidUntilTS > minValidUntil {
				minValidUntil = criteria.MinimumValidUntilTS
			}
		}

		keys, err := n.GetServerKeysWithCriteria(ctx, serverName, minValidUntil)
		if err != nil {
			log.Warn("Failed to get keys for server", "server", serverName, "error", err)

			response.Failures[serverName] = map[string]interface{}{
				"errcode": "M_UNKNOWN",
				"error":   err.Error(),
			}
			continue
		}

		// Add notary signature (perspective signature)
		n.addNotarySignature(keys)

		if violation := n.checkResponsePolicy(serverName, keys); violation != nil {
			response.Failures[serverName] = map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   fmt.Sprintf("Trust policy violation (%s): %s", violation.Rule, violation.Details),
			}
			continue
		}

		response.ServerKeys = append(response.ServerKeys, *keys)
	}

	return response
}

func sortedServerNames(serverKeys map[string]map[string]KeyCriteria) []string {
	names := make([]string, 0, len(serverKeys))
	for serverName := range serverKeys {
		names = append(names, serverName)
	}
	sort.Strings(names)
	return names
}

func (n *Notary) applyTrustPolicyToRequest(request *KeyQueryRequest) map[string]interface{} {
	failures := make(map[string]interface{})
	if request == nil || n.trustPolicy == nil {
		return failures
	}

	for serverName := range request.ServerKeys {
		if violation := n.trustPolicy.CheckServer(serverName); violation != nil {
			failures[serverName] = map[string]interface{}{
				"errcode": "M_FORBIDDEN",
				"error":   fmt.Sprintf("Trust policy violation (%s): %s", violation.Rule, violation.Details),
			}
			delete(request.ServerKeys, serverName)
		}
	}

	return failures
}

func (n *Notary) checkResponsePolicy(serverName string, resp *ServerKeysResponse) *PolicyViolation {
	if n.trustPolicy == nil || resp == nil {
		return nil
	}
	return n.trustPolicy.CheckResponse(serverName, resp)
}

// GetServerKeysWithCriteria gets keys for a server respecting minimum_valid_until_ts
func (n *Notary) GetServerKeysWithCriteria(ctx context.Context, serverName string, minValidUntil int64) (*ServerKeysResponse, error) {
	keys, err := n.GetServerKeys(ctx, serverName)
	if err != nil {
		return nil, err
	}

	// If minimum_valid_until_ts is specified and cached keys don't meet it, refetch
	if minValidUntil > 0 && keys.ValidUntilTS < minValidUntil {
		log.Debug("Cached keys don't meet minimum_valid_until_ts, refetching",
			"server", serverName,
			"cached_valid", keys.ValidUntilTS,
			"required_valid", minValidUntil,
		)

		recordRefetch(RefetchReasonMinValidUntil)

		// Force refetch by bypassing cache
		keys, err = n.fetchAndStoreWithSource(ctx, serverName, FetchSourceRefetch)
		if err != nil {
			return nil, err
		}
	}

	return keys, nil
}

// GetServerKeys gets keys for a server (from cache, storage, or fetch)
func (n *Notary) GetServerKeys(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	// Check context first
	if ctx.Err() != nil {
		return nil, &KeyError{Op: "get_keys", ServerName: serverName, Err: ErrContextCanceled}
	}

	// Check memory cache first
	n.cacheMu.RLock()
	cached, ok := n.cache[serverName]
	n.cacheMu.RUnlock()

	if ok && time.Now().Before(cached.validUntil) {
		log.Debug("Returning keys from memory cache", "server", serverName)
		recordMemoryCacheHit()
		return cached.response, nil
	}
	recordMemoryCacheMiss()

	// Check database cache (with graceful degradation)
	stored, dbErr := n.storage.GetServerResponse(serverName)
	if dbErr != nil {
		log.Warn("Database cache unavailable, continuing with remote fetch", "error", dbErr)
	}
	if stored != nil {
		recordDBCacheHit()
		// Update memory cache
		n.cacheMu.Lock()
		n.cache[serverName] = &cachedResponse{
			response:   stored,
			validUntil: time.Now().Add(time.Duration(n.cacheTTLHours) * time.Hour),
		}
		updateMemoryCacheSize(len(n.cache))
		n.cacheMu.Unlock()

		log.Debug("Returning keys from database cache", "server", serverName)
		return stored, nil
	}
	recordDBCacheMiss()

	// Use singleflight to deduplicate concurrent fetches for the same server
	result, err, _ := n.fetchGroup.Do(serverName, func() (interface{}, error) {
		return n.fetchAndStore(ctx, serverName)
	})

	if err != nil {
		// If fetch failed and we have expired memory cache, return it as fallback
		n.cacheMu.RLock()
		expired, hasExpired := n.cache[serverName]
		n.cacheMu.RUnlock()

		if hasExpired && expired.response != nil {
			validUntil := time.UnixMilli(expired.response.ValidUntilTS)
			if validUntil.After(time.Now()) {
				log.Warn("Using stale memory cache entry after fetch failure",
					"server", serverName,
					"error", err,
					"response_valid_until", validUntil,
				)
				return expired.response, nil
			}
		}

		return nil, err
	}

	return result.(*ServerKeysResponse), nil
}

// fetchAndStore fetches keys from remote and stores them
func (n *Notary) fetchAndStore(ctx context.Context, serverName string) (*ServerKeysResponse, error) {
	return n.fetchAndStoreWithSource(ctx, serverName, FetchSourceDirect)
}

// fetchAndStoreWithSource fetches keys with source tracking for metrics
func (n *Notary) fetchAndStoreWithSource(ctx context.Context, serverName, source string) (*ServerKeysResponse, error) {
	start := time.Now()
	keys, err := n.fetcher.FetchServerKeys(ctx, serverName)
	duration := time.Since(start).Seconds()

	if err != nil {
		recordFetchFailure(source, duration)
		return nil, err
	}
	recordFetchSuccess(source, duration)

	// Store in database
	validUntil := time.UnixMilli(keys.ValidUntilTS)
	if err := n.storage.StoreServerResponse(serverName, keys, validUntil); err != nil {
		log.Warn("Failed to store server response in database", "error", err)
	}

	// Store individual keys
	for keyID, verifyKey := range keys.VerifyKeys {
		pubKeyBytes, err := base64.RawStdEncoding.DecodeString(verifyKey.Key)
		if err != nil {
			continue
		}
		if err := n.storage.StoreKey(serverName, keyID, pubKeyBytes, validUntil); err != nil {
			log.Warn("Failed to store key", "error", err)
		}
	}

	// Update memory cache
	n.cacheMu.Lock()
	n.cache[serverName] = &cachedResponse{
		response:   keys,
		validUntil: time.Now().Add(time.Duration(n.cacheTTLHours) * time.Hour),
	}
	updateMemoryCacheSize(len(n.cache))
	n.cacheMu.Unlock()

	return keys, nil
}

// signResponse signs a response with this notary's key
func (n *Notary) signResponse(response *ServerKeysResponse) {
	// Create copy without signatures for signing
	toSign := map[string]interface{}{
		"server_name":     response.ServerName,
		"valid_until_ts":  response.ValidUntilTS,
		"verify_keys":     response.VerifyKeys,
		"old_verify_keys": response.OldVerifyKeys,
	}

	canonicalBytes, err := canonical.Marshal(toSign)
	if err != nil {
		log.Error("Failed to create canonical JSON for signing", "error", err)
		return
	}

	signature := ed25519.Sign(n.serverKeyPair, canonicalBytes)
	signatureB64 := base64.RawStdEncoding.EncodeToString(signature)

	if response.Signatures == nil {
		response.Signatures = make(map[string]map[string]string)
	}
	response.Signatures[n.serverName] = map[string]string{
		n.serverKeyID: signatureB64,
	}
}

// addNotarySignature adds notary's perspective signature to response
func (n *Notary) addNotarySignature(response *ServerKeysResponse) {
	// Create copy without our signature for signing
	toSign := map[string]interface{}{
		"server_name":     response.ServerName,
		"valid_until_ts":  response.ValidUntilTS,
		"verify_keys":     response.VerifyKeys,
		"old_verify_keys": response.OldVerifyKeys,
	}

	// Include original server's signature
	if response.Signatures != nil {
		toSign["signatures"] = response.Signatures
	}

	canonicalBytes, err := canonical.Marshal(toSign)
	if err != nil {
		log.Error("Failed to create canonical JSON for notary signing", "error", err)
		return
	}

	signature := ed25519.Sign(n.serverKeyPair, canonicalBytes)
	signatureB64 := base64.RawStdEncoding.EncodeToString(signature)

	if response.Signatures == nil {
		response.Signatures = make(map[string]map[string]string)
	}
	if response.Signatures[n.serverName] == nil {
		response.Signatures[n.serverName] = make(map[string]string)
	}
	response.Signatures[n.serverName][n.serverKeyID] = signatureB64
}

// StartCleanupRoutine starts periodic cleanup of expired keys
func (n *Notary) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				n.cleanup()
			}
		}
	}()
}

// RunCleanup performs cleanup of expired keys (exported for initial cleanup)
func (n *Notary) RunCleanup() {
	n.cleanup()
}

func (n *Notary) cleanup() {
	// Clean memory cache
	n.cacheMu.Lock()
	for key, cached := range n.cache {
		if time.Now().After(cached.validUntil) {
			delete(n.cache, key)
		}
	}
	n.cacheMu.Unlock()

	// Clean database
	deleted, err := n.storage.DeleteExpiredKeys()
	if err != nil {
		log.Error("Failed to delete expired keys", "error", err)
	} else if deleted > 0 {
		log.Info("Deleted expired keys", "count", deleted)
	}
}

// GetServerName returns the notary server name
func (n *Notary) GetServerName() string {
	return n.serverName
}

// GetServerKeyID returns the notary key ID
func (n *Notary) GetServerKeyID() string {
	return n.serverKeyID
}

// GetCacheSize returns the number of entries in memory cache
func (n *Notary) GetCacheSize() int {
	n.cacheMu.RLock()
	defer n.cacheMu.RUnlock()
	return len(n.cache)
}

// SetTrustPolicy sets runtime trust policy checks for query flow.
func (n *Notary) SetTrustPolicy(tp *TrustPolicy) {
	n.trustPolicy = tp
}
