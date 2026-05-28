package keys

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
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

func TestFetcherBlockPrivateIPsDefaultsToTrue(t *testing.T) {
	f := NewFetcherWithConfig(FetcherConfig{
		Timeout: time.Second,
	})

	if !f.blockPrivateIPs.Load() {
		t.Fatal("blockPrivateIPs must default to true when config does not override it")
	}
}

func TestFetcherBlockPrivateIPsCanBeDisabledExplicitly(t *testing.T) {
	blockPrivate := false
	f := NewFetcherWithConfig(FetcherConfig{
		Timeout:         time.Second,
		BlockPrivateIPs: &blockPrivate,
	})

	if f.blockPrivateIPs.Load() {
		t.Fatal("blockPrivateIPs should respect explicit false override")
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

func TestCircuitBreakerCleansUpIdleServers(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Minute)
	cb.serverTTL = time.Millisecond

	cb.RecordFailure("idle.server")
	time.Sleep(5 * time.Millisecond)

	if got := cb.Stats()["total_servers"]; got != 0 {
		t.Fatalf("expected expired circuit breaker entry to be cleaned up, got %v tracked servers", got)
	}
}

func TestFallbackSuccessDoesNotOpenCircuitBreaker(t *testing.T) {
	targetServer := "127.0.0.1:1"
	responseData, _ := createSignedKeysResponse(t, targetServer)

	var keysResp ServerKeysResponse
	if err := json.Unmarshal(responseData, &keysResp); err != nil {
		t.Fatalf("failed to decode signed keys response: %v", err)
	}

	notary := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_matrix/key/v2/query" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(KeyQueryResponse{
			ServerKeys: []ServerKeysResponse{keysResp},
		}); err != nil {
			t.Fatalf("failed to write notary response: %v", err)
		}
	}))
	defer notary.Close()

	client := notary.Client()
	client.Timeout = time.Second

	f := NewFetcherWithConfig(FetcherConfig{
		FallbackServers: []string{strings.TrimPrefix(notary.URL, "https://")},
		Timeout:         time.Second,
		RetryAttempts:   1,
	})
	f.SetBlockPrivateIPs(false) // Allow localhost for testing
	f.client = client

	for i := 0; i < 6; i++ {
		resp, err := f.FetchServerKeys(context.Background(), targetServer)
		if err != nil {
			t.Fatalf("FetchServerKeys() error on iteration %d: %v", i, err)
		}
		if resp.ServerName != targetServer {
			t.Fatalf("response server_name = %q, want %q", resp.ServerName, targetServer)
		}
	}

	if !f.circuitBreaker.Allow(targetServer) {
		t.Fatal("successful fallback fetches must not leave the circuit breaker open")
	}
	if got := f.circuitBreaker.Stats()["total_servers"]; got != 0 {
		t.Fatalf("circuit breaker should not retain failure state for successful fallback path, got %v tracked servers", got)
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

func TestRejectPrivateAddress(t *testing.T) {
	blockPrivate := true
	f := NewFetcherWithConfig(FetcherConfig{
		Timeout:         time.Second,
		BlockPrivateIPs: &blockPrivate,
	})

	tests := []struct {
		name     string
		resolved *ResolvedServer
		wantErr  bool
	}{
		{
			name:     "public ip literal allowed",
			resolved: &ResolvedServer{Host: "8.8.8.8", Port: 8448},
			wantErr:  false,
		},
		{
			name:     "private ip literal blocked",
			resolved: &ResolvedServer{Host: "127.0.0.1", Port: 8448},
			wantErr:  true,
		},
		{
			name:     "localhost hostname blocked after resolution",
			resolved: &ResolvedServer{Host: "localhost", Port: 8448},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f.rejectPrivateAddress(context.Background(), "example.org", tt.resolved)
			if tt.wantErr && err == nil {
				t.Fatal("expected private-address rejection")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected rejection: %v", err)
			}
		})
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
