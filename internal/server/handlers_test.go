package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mxkeys/internal/config"
	"mxkeys/internal/keys"
	"mxkeys/internal/version"
)

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()
	data := map[string]string{"key": "<value>&ok"}

	writeJSON(rr, data)

	var result map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	if result["key"] != "<value>&ok" {
		t.Errorf("expected %q, got %s", "<value>&ok", result["key"])
	}
	if strings.Contains(rr.Body.String(), "\\u003c") || strings.Contains(rr.Body.String(), "\\u003e") {
		t.Fatalf("JSON output must not HTML-escape content: %q", rr.Body.String())
	}
}

type matrixError struct {
	ErrCode string `json:"errcode"`
	Error   string `json:"error"`
}

func parseMatrixError(body []byte) (*matrixError, error) {
	var e matrixError
	if err := json.Unmarshal(body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func TestWriteMatrixError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeMatrixError(rr, http.StatusBadRequest, "M_BAD_JSON", "bad request")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	e, err := parseMatrixError(rr.Body.Bytes())
	if err != nil {
		t.Fatalf("failed to parse matrix error: %v", err)
	}
	if e.ErrCode != "M_BAD_JSON" || e.Error != "bad request" {
		t.Fatalf("unexpected matrix error payload: %+v", e)
	}
}

func TestDecodeStrictJSONSingleObject(t *testing.T) {
	var out map[string]interface{}
	err := decodeStrictJSON(strings.NewReader(`{"server_keys":{"example.org":{}}}`), &out, 10)
	if err != nil {
		t.Fatalf("decodeStrictJSON failed: %v", err)
	}
}

func TestDecodeStrictJSONTrailingData(t *testing.T) {
	var out map[string]interface{}
	err := decodeStrictJSON(strings.NewReader(`{"server_keys":{"example.org":{}}} {"extra":1}`), &out, 10)
	if err == nil {
		t.Fatal("expected trailing JSON error")
	}
}

func TestDecodeStrictJSONMaxBytesError(t *testing.T) {
	rec := httptest.NewRecorder()
	body := http.MaxBytesReader(rec, io.NopCloser(strings.NewReader(`{"server_keys":{"example.org":{}}}`)), 8)
	defer body.Close()

	var out map[string]interface{}
	err := decodeStrictJSON(body, &out, 10)
	if err == nil {
		t.Fatal("expected max body size error")
	}

	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		t.Fatalf("expected max bytes error, got: %v", err)
	}
}

func TestDecodeStrictJSONMaxDepth(t *testing.T) {
	var out map[string]interface{}
	err := decodeStrictJSON(strings.NewReader(`{"outer":{"inner":{"deep":{}}}}`), &out, 2)
	if err == nil {
		t.Fatal("expected JSON depth validation error")
	}
}

func newHelperServer() *Server {
	return &Server{
		config: &config.Config{
			Server: config.ServerConfig{Name: "mxkeys.test"},
			Security: config.SecurityConfig{
				MaxServerNameLength: 255,
			},
		},
	}
}

func TestHandleHealthContract(t *testing.T) {
	s := newHelperServer()

	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/health", nil)
	rr := httptest.NewRecorder()
	s.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected JSON content-type, got %q", ct)
	}

	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("status = %q, want healthy", payload["status"])
	}
	if payload["server"] != "mxkeys.test" {
		t.Fatalf("server = %q, want mxkeys.test", payload["server"])
	}
	if payload["version"] != version.Version {
		t.Fatalf("version = %q, want %q", payload["version"], version.Version)
	}
}

func TestHandleLivenessContract(t *testing.T) {
	s := newHelperServer()
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/live", nil)
	rr := httptest.NewRecorder()
	s.handleLiveness(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var payload map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if payload["status"] != "alive" {
		t.Fatalf("status = %q, want alive", payload["status"])
	}
}

func TestHandleVersionContract(t *testing.T) {
	s := newHelperServer()
	req := httptest.NewRequest(http.MethodGet, "/_matrix/federation/v1/version", nil)
	rr := httptest.NewRecorder()
	s.handleVersion(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Header().Get("Server") == "" {
		t.Fatalf("Server header must be set")
	}

	var payload struct {
		Server struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"server"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if payload.Server.Name != version.Name || payload.Server.Version != version.Version {
		t.Fatalf("unexpected version payload: %+v", payload.Server)
	}
}

func TestHandleServerKeysRejectsInvalidPathKeyID(t *testing.T) {
	s := newHelperServer()
	req := httptest.NewRequest(http.MethodGet, "/_matrix/key/v2/server/rsa:invalid", nil)
	req.SetPathValue("keyID", "rsa:invalid")
	rr := httptest.NewRecorder()

	s.handleServerKeys(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	e, err := parseMatrixError(rr.Body.Bytes())
	if err != nil {
		t.Fatalf("failed to parse matrix error: %v", err)
	}
	if e.ErrCode != "M_INVALID_PARAM" {
		t.Fatalf("expected M_INVALID_PARAM, got %q", e.ErrCode)
	}
}

func TestValidateKeyQueryServerKeys(t *testing.T) {
	tests := []struct {
		name      string
		serverMap map[string]map[string]keys.KeyCriteria
		wantErr   bool
	}{
		{
			name: "valid request",
			serverMap: map[string]map[string]keys.KeyCriteria{
				"s-a.mxtest.tech": {"ed25519:keyA": {MinimumValidUntilTS: 0}},
				"s-b.mxtest.tech": {},
			},
			wantErr: false,
		},
		{
			name: "invalid key id",
			serverMap: map[string]map[string]keys.KeyCriteria{
				"s-a.mxtest.tech": {"rsa:bad": {}},
			},
			wantErr: true,
		},
		{
			name: "negative minimum_valid_until_ts",
			serverMap: map[string]map[string]keys.KeyCriteria{
				"s-a.mxtest.tech": {"ed25519:keyA": {MinimumValidUntilTS: -1}},
			},
			wantErr: true,
		},
		{
			name: "empty key id",
			serverMap: map[string]map[string]keys.KeyCriteria{
				"s-a.mxtest.tech": {"": {}},
			},
			wantErr: true,
		},
		{
			name: "unicode hostname must be punycode",
			serverMap: map[string]map[string]keys.KeyCriteria{
				"пример.рф": {"ed25519:keyA": {}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyQueryServerKeys(tt.serverMap, 255)
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
