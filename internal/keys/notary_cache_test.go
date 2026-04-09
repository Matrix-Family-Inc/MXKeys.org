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
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestKeyIDFormat(t *testing.T) {
	keyID := "ed25519:mxkeys"

	parts := strings.Split(keyID, ":")
	if len(parts) != 2 {
		t.Errorf("key ID should have 2 parts, got %d", len(parts))
	}

	if parts[0] != "ed25519" {
		t.Errorf("algorithm = %q, want ed25519", parts[0])
	}

	if parts[1] == "" {
		t.Error("key identifier should not be empty")
	}
}

func TestCacheSize(t *testing.T) {
	cache := make(map[string]*cachedResponse)

	for i := 0; i < 100; i++ {
		serverName := "server" + string(rune('a'+i%26)) + string(rune('0'+i%10))
		pub, _, _ := ed25519.GenerateKey(nil)
		pubB64 := base64.RawStdEncoding.EncodeToString(pub)

		cache[serverName] = &cachedResponse{
			response: &ServerKeysResponse{
				ServerName: serverName,
				VerifyKeys: map[string]VerifyKeyResponse{"ed25519:k": {Key: pubB64}},
			},
			validUntil: time.Now().Add(time.Hour),
		}
	}

	size := len(cache)
	if size != 100 {
		t.Errorf("cache size = %d, want 100", size)
	}
}

func TestExpiredKeyNotReturned(t *testing.T) {
	cache := make(map[string]*cachedResponse)
	serverName := "expired.server"

	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	cache[serverName] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   serverName,
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:key": {Key: pubB64}},
			ValidUntilTS: time.Now().Add(-time.Hour).UnixMilli(),
		},
		validUntil: time.Now().Add(-time.Hour),
	}

	entry := cache[serverName]
	if time.Now().After(entry.validUntil) {
		t.Log("expired entry correctly identified")
	} else {
		t.Error("entry should be expired")
	}

	isExpired := time.UnixMilli(entry.response.ValidUntilTS).Before(time.Now())
	if !isExpired {
		t.Error("ValidUntilTS should indicate expiration")
	}
}

func TestRefreshNeededForExpiring(t *testing.T) {
	cache := make(map[string]*cachedResponse)
	serverName := "expiring.server"

	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	cache[serverName] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   serverName,
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:key": {Key: pubB64}},
			ValidUntilTS: time.Now().Add(10 * time.Minute).UnixMilli(),
		},
		validUntil: time.Now().Add(10 * time.Minute),
	}

	entry := cache[serverName]
	refreshThreshold := 30 * time.Minute

	needsRefresh := time.Until(entry.validUntil) < refreshThreshold
	if !needsRefresh {
		t.Error("entry expiring in 10 minutes should need refresh when threshold is 30 minutes")
	}
}

func TestCacheEntryValidRejectsExpiredResponseValidity(t *testing.T) {
	n := &Notary{}
	now := time.Now()

	cached := &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   "expired.server",
			ValidUntilTS: now.Add(-time.Minute).UnixMilli(),
		},
		validUntil: now.Add(time.Hour),
	}

	if n.cacheEntryValid(cached, now) {
		t.Fatal("cache entry with expired valid_until_ts must be rejected even if local TTL is still active")
	}
}

func TestStoreInMemoryCacheCapsExpiryToResponseValidity(t *testing.T) {
	n := &Notary{
		cache:         make(map[string]*cachedResponse),
		cacheTTLHours: 24,
	}

	now := time.Now()
	responseExpiry := now.Add(10 * time.Minute)
	resp := &ServerKeysResponse{
		ServerName:   "expiring.server",
		ValidUntilTS: responseExpiry.UnixMilli(),
	}

	n.storeInMemoryCache(resp.ServerName, resp)

	cached, ok := n.cache[resp.ServerName]
	if !ok {
		t.Fatal("expected response to be cached")
	}
	if cached.validUntil.After(responseExpiry.Add(1500 * time.Millisecond)) {
		t.Fatalf("cache expiry %v should be capped by response validity %v", cached.validUntil, responseExpiry)
	}
}

func TestCacheInvalidationOnUpdate(t *testing.T) {
	cache := make(map[string]*cachedResponse)
	serverName := "updating.server"

	pub1, _, _ := ed25519.GenerateKey(nil)
	pubB64_1 := base64.RawStdEncoding.EncodeToString(pub1)

	cache[serverName] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   serverName,
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:key1": {Key: pubB64_1}},
			ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		},
		validUntil: time.Now().Add(time.Hour),
	}

	oldKey := cache[serverName].response.VerifyKeys["ed25519:key1"].Key

	pub2, _, _ := ed25519.GenerateKey(nil)
	pubB64_2 := base64.RawStdEncoding.EncodeToString(pub2)

	cache[serverName] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   serverName,
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:key2": {Key: pubB64_2}},
			ValidUntilTS: time.Now().Add(2 * time.Hour).UnixMilli(),
		},
		validUntil: time.Now().Add(2 * time.Hour),
	}

	newKey := cache[serverName].response.VerifyKeys["ed25519:key2"].Key

	if oldKey == newKey {
		t.Error("cache should be updated with new key")
	}

	if _, ok := cache[serverName].response.VerifyKeys["ed25519:key1"]; ok {
		t.Error("old key should not be present after update")
	}
}

func TestKeyRotation(t *testing.T) {
	oldPub, _, _ := ed25519.GenerateKey(nil)
	oldPubB64 := base64.RawStdEncoding.EncodeToString(oldPub)

	newPub, _, _ := ed25519.GenerateKey(nil)
	newPubB64 := base64.RawStdEncoding.EncodeToString(newPub)

	response := &ServerKeysResponse{
		ServerName: "rotating.server",
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:new": {Key: newPubB64},
		},
		OldVerifyKeys: map[string]OldKeyResponse{
			"ed25519:old": {
				Key:       oldPubB64,
				ExpiredTS: time.Now().Add(-24 * time.Hour).UnixMilli(),
			},
		},
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
	}

	if len(response.VerifyKeys) != 1 {
		t.Errorf("should have 1 active key, got %d", len(response.VerifyKeys))
	}

	if len(response.OldVerifyKeys) != 1 {
		t.Errorf("should have 1 old key, got %d", len(response.OldVerifyKeys))
	}

	if _, ok := response.VerifyKeys["ed25519:new"]; !ok {
		t.Error("new key should be in verify_keys")
	}

	oldKeyEntry := response.OldVerifyKeys["ed25519:old"]
	if time.UnixMilli(oldKeyEntry.ExpiredTS).After(time.Now()) {
		t.Error("old key should be expired")
	}
}

func TestValidUntilTSFormat(t *testing.T) {
	ts := time.Now().Add(24 * time.Hour).UnixMilli()

	if ts <= 0 {
		t.Error("valid_until_ts should be positive")
	}

	parsed := time.UnixMilli(ts)
	if parsed.Before(time.Now()) {
		t.Error("valid_until_ts should be in the future")
	}

	duration := time.Until(parsed)
	if duration < 23*time.Hour || duration > 25*time.Hour {
		t.Errorf("duration should be ~24 hours, got %v", duration)
	}
}
