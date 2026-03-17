package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestNotaryApplyTrustPolicyToRequest(t *testing.T) {
	n := &Notary{
		trustPolicy: NewTrustPolicy(TrustPolicyConfig{
			Enabled:  true,
			DenyList: []string{"denied.example"},
		}),
	}

	req := &KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{
			"allowed.example": {},
			"denied.example":  {},
		},
	}

	failures := n.applyTrustPolicyToRequest(req)
	if len(failures) != 1 {
		t.Fatalf("expected 1 policy failure, got %d", len(failures))
	}
	if _, ok := req.ServerKeys["denied.example"]; ok {
		t.Fatalf("denied server should be removed from request")
	}
	if _, ok := req.ServerKeys["allowed.example"]; !ok {
		t.Fatalf("allowed server should remain in request")
	}
}

func TestNotaryCheckResponsePolicy(t *testing.T) {
	n := &Notary{
		trustPolicy: NewTrustPolicy(TrustPolicyConfig{
			Enabled:                 true,
			RequireNotarySignatures: 1,
		}),
	}

	resp := &ServerKeysResponse{
		ServerName: "policy.example",
		Signatures: map[string]map[string]string{
			"policy.example": {"ed25519:key": "sig"},
		},
	}

	violation := n.checkResponsePolicy("policy.example", resp)
	if violation == nil || violation.Rule != "require_notary_signatures" {
		t.Fatalf("expected require_notary_signatures violation, got %#v", violation)
	}
}

func TestSortedServerNames(t *testing.T) {
	serverKeys := map[string]map[string]KeyCriteria{
		"z.example.org": {},
		"a.example.org": {},
		"m.example.org": {},
	}

	names := sortedServerNames(serverKeys)
	expected := []string{"a.example.org", "m.example.org", "z.example.org"}
	if len(names) != len(expected) {
		t.Fatalf("unexpected sorted length: %d", len(names))
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("unexpected order at %d: got %s want %s", i, names[i], expected[i])
		}
	}
}

func TestInitSigningKeyEnforcesSecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not portable on windows")
	}

	tmpDir, err := os.MkdirTemp("", "mxkeys-key-perm-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := &Notary{}
	if err := n.initSigningKey(tmpDir); err != nil {
		t.Fatalf("initSigningKey failed: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "mxkeys_ed25519.key")

	dirInfo, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat key dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("key directory permissions must be 0700, got %04o", dirInfo.Mode().Perm())
	}

	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("key file permissions must be 0600, got %04o", keyInfo.Mode().Perm())
	}
}

func TestInitSigningKeyTightensExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not portable on windows")
	}

	tmpDir, err := os.MkdirTemp("", "mxkeys-key-tighten-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := &Notary{}
	if err := n.initSigningKey(tmpDir); err != nil {
		t.Fatalf("initSigningKey failed: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "mxkeys_ed25519.key")

	if err := os.Chmod(tmpDir, 0o755); err != nil {
		t.Fatalf("failed to relax dir perms: %v", err)
	}
	if err := os.Chmod(keyPath, 0o644); err != nil {
		t.Fatalf("failed to relax file perms: %v", err)
	}

	if err := n.initSigningKey(tmpDir); err != nil {
		t.Fatalf("initSigningKey second run failed: %v", err)
	}

	dirInfo, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat key dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("key directory permissions must be tightened to 0700, got %04o", dirInfo.Mode().Perm())
	}

	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("key file permissions must be tightened to 0600, got %04o", keyInfo.Mode().Perm())
	}
}
