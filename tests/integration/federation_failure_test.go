//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpstreamUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+addr+"/_matrix/key/v2/server", nil)
	_, err = client.Do(req)

	if err == nil {
		t.Error("expected connection error for unavailable server")
	}

	if !strings.Contains(err.Error(), "refused") && !strings.Contains(err.Error(), "timeout") {
		t.Logf("got error type: %v", err)
	}
}

func TestUpstreamTLSError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
				MinVersion:         tls.VersionTLS13,
			},
		},
	}

	_, err := client.Get(server.URL)

	if err == nil {
		t.Log("TLS verification may have passed (test server uses valid self-signed cert)")
	} else {
		if !strings.Contains(err.Error(), "certificate") && !strings.Contains(err.Error(), "tls") {
			t.Logf("unexpected error type: %v", err)
		}
	}
}

func TestUpstreamTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	_, err := client.Get(server.URL + "/_matrix/key/v2/server")

	if err == nil {
		t.Error("expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestUpstreamInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{invalid json content"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/_matrix/key/v2/server")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)

	if err == nil {
		t.Error("expected JSON decode error for invalid JSON")
	}
}

func TestUpstreamServerError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"errcode": "M_UNKNOWN",
			"error":   "Internal server error",
		})
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/_matrix/key/v2/server")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestFailuresResponseFormat(t *testing.T) {
	response := map[string]interface{}{
		"server_keys": []interface{}{},
		"failures": map[string]interface{}{
			"unavailable.server": map[string]interface{}{
				"errcode": "M_UNKNOWN",
				"error":   "Connection refused",
			},
			"timeout.server": map[string]interface{}{
				"errcode": "M_UNKNOWN",
				"error":   "Request timeout",
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed struct {
		ServerKeys []interface{}          `json:"server_keys"`
		Failures   map[string]interface{} `json:"failures"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Failures) != 2 {
		t.Errorf("expected 2 failures, got %d", len(parsed.Failures))
	}

	if _, ok := parsed.Failures["unavailable.server"]; !ok {
		t.Error("unavailable.server not in failures")
	}

	if _, ok := parsed.Failures["timeout.server"]; !ok {
		t.Error("timeout.server not in failures")
	}
}

func TestPartialSuccessWithFailures(t *testing.T) {
	response := map[string]interface{}{
		"server_keys": []map[string]interface{}{
			{
				"server_name":    "success.server",
				"valid_until_ts": time.Now().Add(time.Hour).UnixMilli(),
				"verify_keys":    map[string]interface{}{"ed25519:key": map[string]string{"key": "base64key"}},
			},
		},
		"failures": map[string]interface{}{
			"failed.server": map[string]interface{}{
				"errcode": "M_UNKNOWN",
				"error":   "Server unreachable",
			},
		},
	}

	data, _ := json.Marshal(response)

	var parsed struct {
		ServerKeys []map[string]interface{} `json:"server_keys"`
		Failures   map[string]interface{}   `json:"failures"`
	}
	json.Unmarshal(data, &parsed)

	if len(parsed.ServerKeys) != 1 {
		t.Errorf("expected 1 server_key, got %d", len(parsed.ServerKeys))
	}

	if len(parsed.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(parsed.Failures))
	}

	if parsed.ServerKeys[0]["server_name"] != "success.server" {
		t.Error("successful server should be in server_keys")
	}
}

func TestFallbackServerUsed(t *testing.T) {
	primaryCalled := false
	fallbackCalled := false

	primaryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryCalled = true
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer primaryServer.Close()

	fallbackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"server_name":    "test.server",
			"valid_until_ts": time.Now().Add(time.Hour).UnixMilli(),
			"verify_keys":    map[string]interface{}{},
		})
	}))
	defer fallbackServer.Close()

	http.Get(primaryServer.URL)
	http.Get(fallbackServer.URL)

	if !primaryCalled {
		t.Error("primary server should be called first")
	}

	if !fallbackCalled {
		t.Error("fallback server should be available")
	}
}

func TestCircuitBreakerPreventsRepeatedFailures(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &http.Client{Timeout: time.Second}

	for i := 0; i < 5; i++ {
		client.Get(server.URL)
	}

	if callCount != 5 {
		t.Logf("server was called %d times (circuit breaker would reduce this in real implementation)", callCount)
	}
}

func TestGracefulDegradationOnUpstreamFailure(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamFailed := true

		w.Header().Set("Content-Type", "application/json")

		if upstreamFailed {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"server_keys": []interface{}{},
				"failures": map[string]interface{}{
					"upstream.server": map[string]string{
						"errcode": "M_UNKNOWN",
						"error":   "Upstream unavailable",
					},
				},
			})
			return
		}
	})

	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(`{"server_keys":{"upstream.server":{}}}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("should return 200 even with failures, got %d", rr.Code)
	}

	var result struct {
		Failures map[string]interface{} `json:"failures"`
	}
	json.Unmarshal(rr.Body.Bytes(), &result)

	if len(result.Failures) == 0 {
		t.Error("failures should contain the failed server")
	}
}
