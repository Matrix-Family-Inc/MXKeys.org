//go:build integration

package integration

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mxkeys/internal/zero/canonical"
)

type serverKeysResponse struct {
	ServerName   string                       `json:"server_name"`
	VerifyKeys   map[string]verifyKeyResponse `json:"verify_keys"`
	ValidUntilTS int64                        `json:"valid_until_ts"`
	Signatures   map[string]map[string]string `json:"signatures"`
}

type verifyKeyResponse struct {
	Key string `json:"key"`
}

func createMockMatrixServer(t *testing.T, serverName string) *httptest.Server {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	keyID := "ed25519:test"
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_matrix/key/v2/server" {
			http.NotFound(w, r)
			return
		}

		response := map[string]interface{}{
			"server_name":     serverName,
			"valid_until_ts":  time.Now().Add(24 * time.Hour).UnixMilli(),
			"verify_keys":     map[string]interface{}{keyID: map[string]string{"key": pubB64}},
			"old_verify_keys": map[string]interface{}{},
		}

		canonBytes, _ := canonical.Marshal(response)
		sig := ed25519.Sign(priv, canonBytes)
		sigB64 := base64.RawStdEncoding.EncodeToString(sig)

		response["signatures"] = map[string]interface{}{
			serverName: map[string]string{keyID: sigB64},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestQueryFlowMockServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	mockServer := createMockMatrixServer(t, "mock.matrix.test")
	defer mockServer.Close()

	t.Logf("Mock server running at %s", mockServer.URL)
}

func TestKeyQueryRequestFormat(t *testing.T) {
	body := `{"server_keys": {"matrix.org": {}}}`
	req := httptest.NewRequest("POST", "/_matrix/key/v2/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	var request struct {
		ServerKeys map[string]map[string]interface{} `json:"server_keys"`
	}

	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if len(request.ServerKeys) != 1 {
		t.Errorf("expected 1 server, got %d", len(request.ServerKeys))
	}

	if _, ok := request.ServerKeys["matrix.org"]; !ok {
		t.Error("matrix.org not in server_keys")
	}
}

func TestKeyQueryResponseFormat(t *testing.T) {
	response := map[string]interface{}{
		"server_keys": []map[string]interface{}{
			{
				"server_name":     "matrix.org",
				"valid_until_ts":  time.Now().Add(24 * time.Hour).UnixMilli(),
				"verify_keys":     map[string]interface{}{"ed25519:key1": map[string]string{"key": "base64key"}},
				"old_verify_keys": map[string]interface{}{},
				"signatures": map[string]interface{}{
					"matrix.org":         map[string]string{"ed25519:key1": "sig1"},
					"mxkeys.example.org": map[string]string{"ed25519:mxkeys": "sig2"},
				},
			},
		},
		"failures": map[string]interface{}{},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed struct {
		ServerKeys []serverKeysResponse   `json:"server_keys"`
		Failures   map[string]interface{} `json:"failures"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.ServerKeys) != 1 {
		t.Errorf("expected 1 server key, got %d", len(parsed.ServerKeys))
	}

	sk := parsed.ServerKeys[0]
	if sk.ServerName != "matrix.org" {
		t.Errorf("expected matrix.org, got %s", sk.ServerName)
	}

	if len(sk.Signatures) != 2 {
		t.Errorf("expected 2 signatures (origin + notary), got %d", len(sk.Signatures))
	}

	if _, ok := sk.Signatures["matrix.org"]; !ok {
		t.Error("missing origin signature")
	}

	if _, ok := sk.Signatures["mxkeys.example.org"]; !ok {
		t.Error("missing notary signature")
	}
}

func TestQueryWithMinimumValidUntilTS(t *testing.T) {
	body := fmt.Sprintf(`{
		"server_keys": {
			"matrix.org": {
				"ed25519:key1": {
					"minimum_valid_until_ts": %d
				}
			}
		}
	}`, time.Now().Add(time.Hour).UnixMilli())

	var request struct {
		ServerKeys map[string]map[string]struct {
			MinimumValidUntilTS int64 `json:"minimum_valid_until_ts"`
		} `json:"server_keys"`
	}

	if err := json.Unmarshal([]byte(body), &request); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	matrixKeys, ok := request.ServerKeys["matrix.org"]
	if !ok {
		t.Fatal("matrix.org not found")
	}

	criteria, ok := matrixKeys["ed25519:key1"]
	if !ok {
		t.Fatal("ed25519:key1 criteria not found")
	}

	if criteria.MinimumValidUntilTS <= 0 {
		t.Error("minimum_valid_until_ts should be set")
	}
}
