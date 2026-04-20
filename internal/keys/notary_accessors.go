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
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GetServerName returns the notary server name.
func (n *Notary) GetServerName() string {
	return n.serverName
}

// GetServerKeyID returns the notary key ID.
func (n *Notary) GetServerKeyID() string {
	return n.serverKeyID
}

// GetCacheSize returns the number of entries in the in-memory cache.
func (n *Notary) GetCacheSize() int {
	n.cacheMu.RLock()
	defer n.cacheMu.RUnlock()
	return len(n.cache)
}

// GetCircuitBreakerStats returns per-destination circuit breaker statistics.
func (n *Notary) GetCircuitBreakerStats() map[string]interface{} {
	return n.fetcher.circuitBreaker.Stats()
}

// GetPublicKeyInfo returns the notary public key info and a self-signed payload
// for external STH verification.
func (n *Notary) GetPublicKeyInfo() map[string]interface{} {
	publicKey := n.serverKeyPair.Public().(ed25519.PublicKey)
	pubB64 := base64.RawStdEncoding.EncodeToString(publicKey)
	fingerprint := sha256.Sum256(publicKey)
	fpHex := hex.EncodeToString(fingerprint[:])

	payload := fmt.Sprintf("%s|%s|%s|%s", n.serverName, n.serverKeyID, pubB64, fpHex)
	signature := ed25519.Sign(n.serverKeyPair, []byte(payload))
	sigB64 := base64.RawStdEncoding.EncodeToString(signature)

	return map[string]interface{}{
		"server_name":    n.serverName,
		"key_id":         n.serverKeyID,
		"algorithm":      "ed25519",
		"public_key":     pubB64,
		"fingerprint":    fpHex,
		"self_signature": sigB64,
		"sign_payload":   payload,
	}
}

// SetTrustPolicy installs runtime trust policy checks for the query path.
func (n *Notary) SetTrustPolicy(tp *TrustPolicy) {
	n.configMu.Lock()
	defer n.configMu.Unlock()
	n.trustPolicy = tp
}

// SetBlockPrivateIPs configures resolved-address SSRF protection independent
// of policy enablement.
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

// SetKeyBroadcastHook configures the callback used to broadcast key updates to
// cluster peers.
func (n *Notary) SetKeyBroadcastHook(fn func(serverName, keyID, keyData string, validUntilTS int64)) {
	n.configMu.Lock()
	defer n.configMu.Unlock()
	n.keyBroadcastHook = fn
}

func (n *Notary) getTrustPolicy() *TrustPolicy {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.trustPolicy
}

func (n *Notary) getTransparency() *TransparencyLog {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.transparency
}

func (n *Notary) getAnalytics() *Analytics {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.analytics
}

func (n *Notary) getKeyBroadcastHook() func(serverName, keyID, keyData string, validUntilTS int64) {
	n.configMu.RLock()
	defer n.configMu.RUnlock()
	return n.keyBroadcastHook
}
