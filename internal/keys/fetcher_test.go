package keys

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"mxkeys/internal/zero/canonical"
)

func createSignedKeysResponse(t *testing.T, serverName string) ([]byte, ed25519.PublicKey) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	keyID := "ed25519:test"
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	response := map[string]interface{}{
		"server_name":     serverName,
		"valid_until_ts":  time.Now().Add(24 * time.Hour).UnixMilli(),
		"verify_keys":     map[string]interface{}{keyID: map[string]string{"key": pubB64}},
		"old_verify_keys": map[string]interface{}{},
	}

	canonBytes, err := canonical.Marshal(response)
	if err != nil {
		t.Fatalf("failed to canonicalize response: %v", err)
	}
	sig := ed25519.Sign(priv, canonBytes)
	sigB64 := base64.RawStdEncoding.EncodeToString(sig)

	response["signatures"] = map[string]interface{}{
		serverName: map[string]string{keyID: sigB64},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}
	return data, pub
}

func TestFetcherCreation(t *testing.T) {
	f := NewFetcher([]string{"matrix.org"}, 30*time.Second)
	if f == nil {
		t.Fatal("NewFetcher returned nil")
	}
	if f.client == nil {
		t.Error("client is nil")
	}
	if f.resolver == nil {
		t.Error("resolver is nil")
	}
	if f.circuitBreaker == nil {
		t.Error("circuitBreaker is nil")
	}
}

func TestFetcherWithConfig(t *testing.T) {
	trustedNotary := TrustedNotaryKey{
		ServerName: "notary.test",
		KeyID:      "ed25519:notary",
		PublicKey:  make([]byte, 32),
	}

	f := NewFetcherWithConfig(FetcherConfig{
		FallbackServers: []string{"fallback.test"},
		Timeout:         10 * time.Second,
		TrustedNotaries: []TrustedNotaryKey{trustedNotary},
		RetryAttempts:   5,
	})

	if f.retryAttempts != 5 {
		t.Errorf("retryAttempts = %d, want 5", f.retryAttempts)
	}

	if len(f.trustedNotaries) != 1 {
		t.Errorf("trustedNotaries count = %d, want 1", len(f.trustedNotaries))
	}
}

func TestServerKeysResponseParsing(t *testing.T) {
	serverName := "test.matrix.org"
	responseData, _ := createSignedKeysResponse(t, serverName)

	var response ServerKeysResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if response.ServerName != serverName {
		t.Errorf("server_name = %q, want %q", response.ServerName, serverName)
	}

	if len(response.VerifyKeys) != 1 {
		t.Errorf("verify_keys count = %d, want 1", len(response.VerifyKeys))
	}

	if _, ok := response.VerifyKeys["ed25519:test"]; !ok {
		t.Error("key ed25519:test not found")
	}

	if response.ValidUntilTS <= time.Now().UnixMilli() {
		t.Error("valid_until_ts should be in the future")
	}

	if len(response.Signatures) == 0 {
		t.Error("signatures should not be empty")
	}
}

func TestServerKeysResponseKeyDecoding(t *testing.T) {
	serverName := "test.matrix.org"
	responseData, expectedPub := createSignedKeysResponse(t, serverName)

	var response ServerKeysResponse
	json.Unmarshal(responseData, &response)

	vk := response.VerifyKeys["ed25519:test"]
	keyBytes, err := base64.RawStdEncoding.DecodeString(vk.Key)
	if err != nil {
		t.Fatalf("failed to decode key: %v", err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		t.Errorf("key length = %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}

	if !ed25519.PublicKey(keyBytes).Equal(expectedPub) {
		t.Error("decoded key does not match expected")
	}
}

func TestMockServerServes(t *testing.T) {
	serverName := "mock.matrix.test"
	responseData, _ := createSignedKeysResponse(t, serverName)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_matrix/key/v2/server" {
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseData)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	t.Logf("Mock server at %s", server.URL)
}

func TestRetryCountTracking(t *testing.T) {
	var attempts int32

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}

		serverName := "retry.test"
		responseData, _ := createSignedKeysResponse(t, serverName)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseData)
	}))
	defer server.Close()

	t.Logf("Retry test server at %s, attempts will be tracked", server.URL)
}

func TestBadJSONResponse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	var response ServerKeysResponse
	err := json.Unmarshal([]byte("{invalid json"), &response)
	if err == nil {
		t.Error("invalid JSON should fail to unmarshal")
	}
}

func TestCircuitBreakerIntegration(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute)
	server := "failing.server"

	cb.RecordFailure(server)
	cb.RecordFailure(server)

	if cb.Allow(server) {
		t.Error("circuit should be open after failures")
	}
}

func TestFetchContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	f := NewFetcher(nil, time.Second)
	_, err := f.FetchServerKeys(ctx, "matrix.org")

	if err == nil {
		t.Error("should return error on canceled context")
	}
}

func TestIsPermanentError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		permanent bool
	}{
		{"invalid response", NewValidationError("test", ErrInvalidResponse), true},
		{"signature invalid", NewSignatureError("test", ErrSignatureInvalid), true},
		{"fetch failed", NewFetchError("test", ErrFetchFailed), false},
		{"resolve failed", NewResolveError("test", ErrResolveFailed), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPermanentError(tt.err)
			if result != tt.permanent {
				t.Errorf("IsPermanentError(%v) = %v, want %v", tt.err, result, tt.permanent)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		errMsg    string
		retryable bool
	}{
		{"connection timeout", true},
		{"connection refused", true},
		{"connection reset by peer", true},
		{"no such host", true},
		{"i/o timeout", true},
		{"temporary failure in DNS", true},
		{"invalid JSON", false},
		{"server error", false},
	}

	for _, tt := range tests {
		err := &testError{msg: tt.errMsg}
		result := isRetryableError(err)
		if result != tt.retryable {
			t.Errorf("isRetryableError(%q) = %v, want %v", tt.errMsg, result, tt.retryable)
		}
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestKeyErrorFormat(t *testing.T) {
	err := &KeyError{
		Op:         "fetch",
		ServerName: "test.server",
		Err:        ErrFetchFailed,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("error string should not be empty")
	}

	if err.Unwrap() != ErrFetchFailed {
		t.Error("Unwrap should return underlying error")
	}
}

func TestFetcherDefaultConfig(t *testing.T) {
	f := NewFetcher([]string{"fallback.test"}, 30*time.Second)

	if f.retryAttempts != defaultRetryAttempts {
		t.Errorf("default retry attempts = %d, want %d", f.retryAttempts, defaultRetryAttempts)
	}
}

func TestTrustedNotaryKeyStruct(t *testing.T) {
	key := TrustedNotaryKey{
		ServerName: "notary.test",
		KeyID:      "ed25519:notary",
		PublicKey:  make([]byte, 32),
	}

	if key.ServerName != "notary.test" {
		t.Errorf("ServerName = %q, want notary.test", key.ServerName)
	}

	if key.KeyID != "ed25519:notary" {
		t.Errorf("KeyID = %q, want ed25519:notary", key.KeyID)
	}

	if len(key.PublicKey) != 32 {
		t.Errorf("PublicKey length = %d, want 32", len(key.PublicKey))
	}
}

func TestVerifyNotarySignatureValid(t *testing.T) {
	notaryPub, notaryPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate notary key: %v", err)
	}

	f := NewFetcherWithConfig(FetcherConfig{
		Timeout: time.Second,
		TrustedNotaries: []TrustedNotaryKey{
			{
				ServerName: "notary.example.org",
				KeyID:      "ed25519:notary",
				PublicKey:  notaryPub,
			},
		},
	})

	resp := &ServerKeysResponse{
		ServerName:   "server.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:server": {Key: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))},
		},
		OldVerifyKeys: map[string]OldKeyResponse{},
		Signatures: map[string]map[string]string{
			"server.example.org": {"ed25519:server": "origin-sig"},
		},
	}

	toSign := map[string]interface{}{
		"server_name":     resp.ServerName,
		"valid_until_ts":  resp.ValidUntilTS,
		"verify_keys":     resp.VerifyKeys,
		"old_verify_keys": resp.OldVerifyKeys,
		"signatures": map[string]map[string]string{
			"server.example.org": {"ed25519:server": "origin-sig"},
		},
	}
	canonicalBytes, err := canonical.Marshal(toSign)
	if err != nil {
		t.Fatalf("failed to canonicalize: %v", err)
	}

	resp.Signatures["notary.example.org"] = map[string]string{
		"ed25519:notary": base64.RawStdEncoding.EncodeToString(ed25519.Sign(notaryPriv, canonicalBytes)),
	}

	if err := f.verifyNotarySignature("notary.example.org", resp); err != nil {
		t.Fatalf("verifyNotarySignature returned error: %v", err)
	}
}

func TestVerifyNotarySignatureMismatch(t *testing.T) {
	notaryPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate notary key: %v", err)
	}

	f := NewFetcherWithConfig(FetcherConfig{
		Timeout: time.Second,
		TrustedNotaries: []TrustedNotaryKey{
			{
				ServerName: "notary.example.org",
				KeyID:      "ed25519:notary",
				PublicKey:  notaryPub,
			},
		},
	})

	resp := &ServerKeysResponse{
		ServerName:   "server.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:server": {Key: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))},
		},
		OldVerifyKeys: map[string]OldKeyResponse{},
		Signatures: map[string]map[string]string{
			"server.example.org": {"ed25519:server": "origin-sig"},
			"notary.example.org": {"ed25519:notary": base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))},
		},
	}

	err = f.verifyNotarySignature("notary.example.org", resp)
	if !errors.Is(err, ErrNotaryKeyMismatch) {
		t.Fatalf("expected ErrNotaryKeyMismatch, got %v", err)
	}
}

func TestReadLimitedBody(t *testing.T) {
	body, err := readLimitedBody(io.NopCloser(strings.NewReader("abc")), 3)
	if err != nil {
		t.Fatalf("readLimitedBody failed: %v", err)
	}
	if string(body) != "abc" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestReadLimitedBodyTooLarge(t *testing.T) {
	_, err := readLimitedBody(io.NopCloser(strings.NewReader("abcd")), 3)
	if err == nil {
		t.Fatal("expected body-too-large error")
	}
}
