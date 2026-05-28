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
	"time"

	"mxkeys/internal/zero/log"
)

// StartCleanupRoutine starts the periodic cleanup of expired keys.
// Restarting while a previous routine is active cancels and waits for it.
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

// StopCleanupRoutine stops the periodic cleanup goroutine and waits for exit.
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

// RunCleanup performs a one-shot cleanup of expired keys.
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

// cacheExpiresAt picks the earlier of response valid_until and TTL window.
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

// storeInMemoryCache inserts or evicts a cached response under the cache lock.
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
