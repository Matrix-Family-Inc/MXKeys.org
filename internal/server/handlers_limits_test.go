/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"mxkeys/internal/config"
)

func newValidationOnlyServer() *Server {
	return &Server{
		config: &config.Config{
			Security: config.SecurityConfig{
				MaxServerNameLength: 255,
				MaxServersPerQuery:  100,
			},
		},
	}
}

func decodeMatrixErrorBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	return out
}

func TestHandleKeyQueryRejectsOversizedBody(t *testing.T) {
	s := newValidationOnlyServer()
	hugeBody := `{"server_keys":{"example.org":{}},"padding":"` + strings.Repeat("a", maxRequestBodySize+16) + `"}`

	req := httptest.NewRequest(http.MethodPost, "/_matrix/key/v2/query", strings.NewReader(hugeBody))
	rr := httptest.NewRecorder()
	s.handleKeyQuery(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
	errBody := decodeMatrixErrorBody(t, rr)
	if errBody["errcode"] != "M_TOO_LARGE" {
		t.Fatalf("expected M_TOO_LARGE, got %q", errBody["errcode"])
	}
}

func TestHandleKeyQueryEmptyServerKeys(t *testing.T) {
	s := newValidationOnlyServer()

	req := httptest.NewRequest(http.MethodPost, "/_matrix/key/v2/query", strings.NewReader(`{"server_keys":{}}`))
	rr := httptest.NewRecorder()
	s.handleKeyQuery(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		ServerKeys []json.RawMessage `json:"server_keys"`
		Failures   map[string]any    `json:"failures"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ServerKeys == nil || len(body.ServerKeys) != 0 {
		t.Fatalf("server_keys = %#v, want an empty array", body.ServerKeys)
	}
	if len(body.Failures) != 0 {
		t.Fatalf("failures = %#v", body.Failures)
	}
}

func TestHandleKeyQueryRespectsConfiguredMaxServers(t *testing.T) {
	s := newValidationOnlyServer()
	s.config.Security.MaxServersPerQuery = 2

	var b strings.Builder
	b.WriteString(`{"server_keys":{`)
	for i := 0; i < 3; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"s-`)
		b.WriteString(strconv.Itoa(i))
		b.WriteString(`.example.org":{}`)
	}
	b.WriteString(`}}`)

	req := httptest.NewRequest(http.MethodPost, "/_matrix/key/v2/query", strings.NewReader(b.String()))
	rr := httptest.NewRecorder()
	s.handleKeyQuery(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
	errBody := decodeMatrixErrorBody(t, rr)
	if errBody["errcode"] != "M_BAD_JSON" {
		t.Fatalf("expected M_BAD_JSON, got %q", errBody["errcode"])
	}
}

func TestHandleKeyQueryRejectsInvalidServerAndKeyCriteria(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantErr    string
		statusCode int
	}{
		{
			name:       "invalid server name",
			body:       `{"server_keys":{"../etc/passwd":{"ed25519:auto":{}}}}`,
			wantErr:    "M_INVALID_PARAM",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "invalid key id",
			body:       `{"server_keys":{"example.org":{"rsa:bad":{}}}}`,
			wantErr:    "M_INVALID_PARAM",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "negative minimum_valid_until_ts",
			body:       `{"server_keys":{"example.org":{"ed25519:auto":{"minimum_valid_until_ts":-1}}}}`,
			wantErr:    "M_INVALID_PARAM",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newValidationOnlyServer()
			req := httptest.NewRequest(http.MethodPost, "/_matrix/key/v2/query", strings.NewReader(tt.body))
			rr := httptest.NewRecorder()

			s.handleKeyQuery(rr, req)

			if rr.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, rr.Code)
			}
			errBody := decodeMatrixErrorBody(t, rr)
			if errBody["errcode"] != tt.wantErr {
				t.Fatalf("expected errcode %s, got %q", tt.wantErr, errBody["errcode"])
			}
		})
	}
}
