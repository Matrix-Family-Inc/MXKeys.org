package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mxkeys/internal/zero/canonical"
)

func TestSigningKeyGeneration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mxkeys-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	keyPath := filepath.Join(tmpDir, "test_key.key")

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	if err := os.WriteFile(keyPath, priv.Seed(), 0600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}

	seed, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key: %v", err)
	}

	loadedPriv := ed25519.NewKeyFromSeed(seed)
	loadedPub := loadedPriv.Public().(ed25519.PublicKey)

	if !pub.Equal(loadedPub) {
		t.Error("loaded key does not match generated key")
	}
}

func TestSigningKeyPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mxkeys-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	keyPath := filepath.Join(tmpDir, "persist_key.key")

	_, priv1, _ := ed25519.GenerateKey(nil)
	os.WriteFile(keyPath, priv1.Seed(), 0600)

	seed, _ := os.ReadFile(keyPath)
	priv2 := ed25519.NewKeyFromSeed(seed)

	msg := []byte("test message")
	sig1 := ed25519.Sign(priv1, msg)
	sig2 := ed25519.Sign(priv2, msg)

	pub1 := priv1.Public().(ed25519.PublicKey)
	if !ed25519.Verify(pub1, msg, sig2) {
		t.Error("loaded key should produce verifiable signatures")
	}

	_ = sig1
}

func TestGetOwnKeysFormat(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)
	serverName := "test.mxkeys.org"
	keyID := "ed25519:mxkeys"

	ownKeys := &ServerKeysResponse{
		ServerName: serverName,
		VerifyKeys: map[string]VerifyKeyResponse{
			keyID: {Key: pubB64},
		},
		OldVerifyKeys: map[string]OldKeyResponse{},
		ValidUntilTS:  time.Now().Add(24 * time.Hour).UnixMilli(),
	}

	if ownKeys.ServerName != serverName {
		t.Errorf("server_name = %q, want %q", ownKeys.ServerName, serverName)
	}

	if len(ownKeys.VerifyKeys) != 1 {
		t.Errorf("verify_keys count = %d, want 1", len(ownKeys.VerifyKeys))
	}

	vk, ok := ownKeys.VerifyKeys[keyID]
	if !ok {
		t.Fatalf("key %s not found", keyID)
	}

	keyBytes, err := base64.RawStdEncoding.DecodeString(vk.Key)
	if err != nil {
		t.Fatalf("failed to decode key: %v", err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		t.Errorf("key length = %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}
}

func TestSignResponse(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)
	serverName := "signer.test"
	keyID := "ed25519:signer"

	original := map[string]interface{}{
		"server_name":     "upstream.test",
		"valid_until_ts":  time.Now().Add(time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{"ed25519:key": map[string]string{"key": pubB64}},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, err := canonical.Marshal(original)
	if err != nil {
		t.Fatalf("failed to canonicalize: %v", err)
	}

	sig := ed25519.Sign(priv, canonBytes)
	sigB64 := base64.RawStdEncoding.EncodeToString(sig)

	original["signatures"] = map[string]interface{}{
		serverName: map[string]string{keyID: sigB64},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal signed payload: %v", err)
	}
	var response ServerKeysResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("failed to unmarshal signed payload: %v", err)
	}

	if len(response.Signatures) != 1 {
		t.Errorf("signatures count = %d, want 1", len(response.Signatures))
	}

	serverSigs, ok := response.Signatures[serverName]
	if !ok {
		t.Fatal("signer signatures not found")
	}

	if _, ok := serverSigs[keyID]; !ok {
		t.Error("key ID not in signatures")
	}
}

func TestAddNotarySignature(t *testing.T) {
	upstreamPub, upstreamPriv, _ := ed25519.GenerateKey(nil)
	upstreamPubB64 := base64.RawStdEncoding.EncodeToString(upstreamPub)
	upstreamName := "upstream.matrix"
	upstreamKeyID := "ed25519:upstream"

	original := map[string]interface{}{
		"server_name":     upstreamName,
		"valid_until_ts":  time.Now().Add(time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{upstreamKeyID: map[string]string{"key": upstreamPubB64}},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, _ := canonical.Marshal(original)
	upstreamSig := ed25519.Sign(upstreamPriv, canonBytes)
	upstreamSigB64 := base64.RawStdEncoding.EncodeToString(upstreamSig)

	original["signatures"] = map[string]interface{}{
		upstreamName: map[string]string{upstreamKeyID: upstreamSigB64},
	}

	notaryPub, notaryPriv, _ := ed25519.GenerateKey(nil)
	_ = notaryPub
	notaryName := "notary.mxkeys"
	notaryKeyID := "ed25519:notary"

	canonBytes2, _ := canonical.Marshal(original)
	notarySig := ed25519.Sign(notaryPriv, canonBytes2)
	notarySigB64 := base64.RawStdEncoding.EncodeToString(notarySig)

	sigs := original["signatures"].(map[string]interface{})
	sigs[notaryName] = map[string]string{notaryKeyID: notarySigB64}

	if len(sigs) != 2 {
		t.Errorf("expected 2 signers (upstream + notary), got %d", len(sigs))
	}

	if _, ok := sigs[upstreamName]; !ok {
		t.Error("upstream signature was lost")
	}

	if _, ok := sigs[notaryName]; !ok {
		t.Error("notary signature not added")
	}
}

func TestCacheHitMemory(t *testing.T) {
	cache := make(map[string]*cachedResponse)
	serverName := "cached.server"

	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	cache[serverName] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   serverName,
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:key": {Key: pubB64}},
			ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		},
		validUntil: time.Now().Add(time.Hour),
	}

	cached, ok := cache[serverName]
	if !ok {
		t.Fatal("expected cache hit")
	}

	if cached.response.ServerName != serverName {
		t.Errorf("cached server = %q, want %q", cached.response.ServerName, serverName)
	}
}

func TestCacheMinimumValidUntil(t *testing.T) {
	cache := make(map[string]*cachedResponse)
	serverName := "expiring.server"

	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	cache[serverName] = &cachedResponse{
		response: &ServerKeysResponse{
			ServerName:   serverName,
			VerifyKeys:   map[string]VerifyKeyResponse{"ed25519:key": {Key: pubB64}},
			ValidUntilTS: time.Now().Add(30 * time.Minute).UnixMilli(),
		},
		validUntil: time.Now().Add(30 * time.Minute),
	}

	cached := cache[serverName]
	minimumRequired := time.Now().Add(time.Hour).UnixMilli()

	satisfiesMinimum := cached.response.ValidUntilTS >= minimumRequired
	if satisfiesMinimum {
		t.Error("cache entry with 30min validity should not satisfy 1hr minimum")
	}
}

func TestCleanupExpiredEntries(t *testing.T) {
	cache := make(map[string]*cachedResponse)

	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	cache["expired"] = &cachedResponse{
		response:   &ServerKeysResponse{ServerName: "expired", VerifyKeys: map[string]VerifyKeyResponse{"ed25519:k": {Key: pubB64}}},
		validUntil: time.Now().Add(-time.Hour),
	}

	cache["valid"] = &cachedResponse{
		response:   &ServerKeysResponse{ServerName: "valid", VerifyKeys: map[string]VerifyKeyResponse{"ed25519:k": {Key: pubB64}}},
		validUntil: time.Now().Add(time.Hour),
	}

	now := time.Now()
	for key, entry := range cache {
		if entry.validUntil.Before(now) {
			delete(cache, key)
		}
	}

	if _, ok := cache["expired"]; ok {
		t.Error("expired entry should be removed")
	}

	if _, ok := cache["valid"]; !ok {
		t.Error("valid entry should remain")
	}
}
