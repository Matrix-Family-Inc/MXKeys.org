/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Apr 07 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mxkeys/internal/zero/log"
)

const ClusterReplicatedResponseKeyID = "_server_response"

// StartCleanupRoutine starts periodic cleanup of expired keys.
func (n *Notary) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	childCtx, cancel := context.WithCancel(ctx)

	n.cleanupMu.Lock()
	if n.cleanupCancel != nil {
		n.cleanupCancel()
		n.cleanupMu.Unlock()
		n.cleanupWG.Wait()
		n.cleanupMu.Lock()
	}
	n.cleanupCancel = cancel
	n.cleanupWG.Add(1)
	n.cleanupMu.Unlock()

	ticker := time.NewTicker(interval)
	go func() {
		defer n.cleanupWG.Done()
		defer ticker.Stop()
		for {
			select {
			case <-childCtx.Done():
				return
			case <-ticker.C:
				n.cleanup()
			}
		}
	}()
}

// StopCleanupRoutine stops the periodic cleanup goroutine and waits for it to exit.
func (n *Notary) StopCleanupRoutine() {
	n.cleanupMu.Lock()
	cancel := n.cleanupCancel
	n.cleanupCancel = nil
	n.cleanupMu.Unlock()

	if cancel != nil {
		cancel()
	}
	n.cleanupWG.Wait()
}

// RunCleanup performs cleanup of expired keys (exported for initial cleanup).
func (n *Notary) RunCleanup() {
	n.cleanup()
}

func (n *Notary) cleanup() {
	n.cacheMu.Lock()
	for key, cached := range n.cache {
		if time.Now().After(cached.validUntil) {
			delete(n.cache, key)
		}
	}
	n.cacheMu.Unlock()

	deleted, err := n.storage.DeleteExpiredKeys()
	if err != nil {
		log.Error("Failed to delete expired keys", "error", err)
	} else if deleted > 0 {
		log.Info("Deleted expired keys", "count", deleted)
	}
}

func (n *Notary) cacheExpiresAt(response *ServerKeysResponse, now time.Time) time.Time {
	cacheExpiry := now.Add(time.Duration(n.cacheTTLHours) * time.Hour)
	if response == nil || response.ValidUntilTS <= 0 {
		return cacheExpiry
	}

	responseExpiry := time.UnixMilli(response.ValidUntilTS)
	if responseExpiry.Before(cacheExpiry) {
		return responseExpiry
	}
	return cacheExpiry
}

func (n *Notary) cacheEntryValid(cached *cachedResponse, now time.Time) bool {
	if cached == nil || cached.response == nil {
		return false
	}
	if !now.Before(cached.validUntil) {
		return false
	}
	return cached.response.ValidUntilTS > now.UnixMilli()
}

func (n *Notary) storeInMemoryCache(serverName string, response *ServerKeysResponse) {
	now := time.Now()
	expiresAt := n.cacheExpiresAt(response, now)

	n.cacheMu.Lock()
	defer n.cacheMu.Unlock()

	if response == nil || !expiresAt.After(now) {
		delete(n.cache, serverName)
		updateMemoryCacheSize(len(n.cache))
		return
	}

	n.cache[serverName] = &cachedResponse{
		response:   response,
		validUntil: expiresAt,
	}
	updateMemoryCacheSize(len(n.cache))
}

// GetServerName returns the notary server name.
func (n *Notary) GetServerName() string {
	return n.serverName
}

// GetServerKeyID returns the notary key ID.
func (n *Notary) GetServerKeyID() string {
	return n.serverKeyID
}

// GetCacheSize returns the number of entries in memory cache.
func (n *Notary) GetCacheSize() int {
	n.cacheMu.RLock()
	defer n.cacheMu.RUnlock()
	return len(n.cache)
}

// GetCircuitBreakerStats returns per-destination circuit breaker statistics.
func (n *Notary) GetCircuitBreakerStats() map[string]interface{} {
	return n.fetcher.circuitBreaker.Stats()
}

// SetTrustPolicy sets runtime trust policy checks for query flow.
func (n *Notary) SetTrustPolicy(tp *TrustPolicy) {
	n.configMu.Lock()
	defer n.configMu.Unlock()
	n.trustPolicy = tp
}

// SetBlockPrivateIPs configures resolved-address SSRF protection independent of policy enablement.
func (n *Notary) SetBlockPrivateIPs(enabled bool) {
	if n.fetcher != nil {
		n.fetcher.SetBlockPrivateIPs(enabled)
	}
}

// SetTransparencyLog enables transparency logging for query-path events.
func (n *Notary) SetTransparencyLog(tl *TransparencyLog) {
	n.configMu.Lock()
	defer n.configMu.Unlock()
	n.transparency = tl
}

// SetAnalytics enables runtime analytics aggregation for query-path events.
func (n *Notary) SetAnalytics(a *Analytics) {
	n.configMu.Lock()
	defer n.configMu.Unlock()
	n.analytics = a
}

// SetKeyBroadcastHook configures callback used to broadcast key updates to cluster peers.
func (n *Notary) SetKeyBroadcastHook(fn func(serverName, keyID, keyData string, validUntilTS int64)) {
	n.configMu.Lock()
	defer n.configMu.Unlock()
	n.keyBroadcastHook = fn
}

// getTrustPolicy returns the trust policy with read lock.
func (n *Notary) getTrustPolicy() *TrustPolicy {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.trustPolicy
}

// getTransparency returns the transparency log with read lock.
func (n *Notary) getTransparency() *TransparencyLog {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.transparency
}

// getAnalytics returns the analytics with read lock.
func (n *Notary) getAnalytics() *Analytics {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.analytics
}

// getKeyBroadcastHook returns the broadcast hook with read lock.
func (n *Notary) getKeyBroadcastHook() func(serverName, keyID, keyData string, validUntilTS int64) {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.keyBroadcastHook
}

// ApplyReplicatedServerResponse applies server response received from cluster replication.
func (n *Notary) ApplyReplicatedServerResponse(serverName string, rawResponse string, validUntilTS int64) error {
	response, err := n.validateReplicatedServerResponse(serverName, rawResponse, validUntilTS)
	if err != nil {
		return err
	}
	validUntil := time.UnixMilli(response.ValidUntilTS)
	if err := n.storage.StoreServerResponse(serverName, response, validUntil); err != nil {
		return fmt.Errorf("failed to store replicated server response: %w", err)
	}

	for keyID, verifyKey := range response.VerifyKeys {
		pubKeyBytes, err := decodeBase64(verifyKey.Key)
		if err != nil {
			log.Debug("Skipping replicated key with invalid base64", "server", serverName, "key_id", keyID, "error", err)
			continue
		}
		if err := n.storage.StoreKey(serverName, keyID, pubKeyBytes, validUntil); err != nil {
			log.Warn("Failed to store replicated key", "server", serverName, "key_id", keyID, "error", err)
		}
	}

	n.storeInMemoryCache(serverName, cloneServerKeysResponse(response))

	return nil
}

func (n *Notary) validateReplicatedServerResponse(serverName string, rawResponse string, validUntilTS int64) (*ServerKeysResponse, error) {
	if strings.TrimSpace(rawResponse) == "" {
		return nil, fmt.Errorf("replicated response payload is empty")
	}

	var response ServerKeysResponse
	if err := json.Unmarshal([]byte(rawResponse), &response); err != nil {
		return nil, fmt.Errorf("failed to decode replicated response: %w", err)
	}
	if response.ServerName == "" {
		return nil, fmt.Errorf("replicated response server_name is missing")
	}
	if response.ServerName != serverName {
		return nil, fmt.Errorf("replicated response server mismatch: expected %s got %s", serverName, response.ServerName)
	}
	if response.ValidUntilTS <= 0 {
		return nil, fmt.Errorf("replicated response valid_until_ts is missing")
	}
	if response.ValidUntilTS != validUntilTS {
		return nil, fmt.Errorf("replicated response valid_until_ts mismatch: expected %d got %d", validUntilTS, response.ValidUntilTS)
	}
	if n.fetcher == nil {
		return nil, fmt.Errorf("replicated response verification is unavailable")
	}
	if err := n.fetcher.verifySelfSignature(&response, []byte(rawResponse)); err != nil {
		return nil, fmt.Errorf("replicated response failed verification: %w", err)
	}
	return &response, nil
}

// SignedTreeHead returns a signed snapshot of the transparency log merkle tree.
func (n *Notary) SignedTreeHead() (map[string]interface{}, error) {
	tl := n.getTransparency()
	if tl == nil {
		return nil, fmt.Errorf("transparency log not enabled")
	}

	tl.mu.RLock()
	treeSize := tl.merkleTree.Size()
	rootHash := tl.merkleTree.RootHex()
	tl.mu.RUnlock()

	now := time.Now().UTC()
	payload := fmt.Sprintf("%d|%s|%d", treeSize, rootHash, now.UnixMilli())

	signature := ed25519.Sign(n.serverKeyPair, []byte(payload))
	signatureB64 := base64.RawStdEncoding.EncodeToString(signature)

	return map[string]interface{}{
		"tree_size":    treeSize,
		"root_hash":    rootHash,
		"timestamp":    now.Format(time.RFC3339),
		"timestamp_ms": now.UnixMilli(),
		"signer":       n.serverName,
		"key_id":       n.serverKeyID,
		"signature":    signatureB64,
		"sign_payload": payload,
	}, nil
}

func decodeBase64(v string) ([]byte, error) {
	b, err := base64.RawStdEncoding.DecodeString(v)
	if err == nil {
		return b, nil
	}
	return base64.StdEncoding.DecodeString(v)
}
