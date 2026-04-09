package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDMiddlewareGeneratesAndPropagatesID(t *testing.T) {
	var seenInContext string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenInContext = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatalf("X-Request-ID should be generated")
	}
	if len(requestID) != 32 {
		t.Fatalf("generated X-Request-ID should be 32 hex chars, got %d", len(requestID))
	}
	if seenInContext != requestID {
		t.Fatalf("request ID in context = %q, header = %q", seenInContext, requestID)
	}
}

func TestRequestIDMiddlewarePreservesIncomingID(t *testing.T) {
	expected := "my-custom-id-12345"
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", expected)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("X-Request-ID"); got != expected {
		t.Fatalf("X-Request-ID should be passed through, got %q", got)
	}
}

func TestSecurityHeadersMiddlewareContract(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/_matrix/key/v2/query", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for header, expected := range expectedHeaders {
		if got := rr.Header().Get(header); got != expected {
			t.Fatalf("header %s: expected %q, got %q", header, expected, got)
		}
	}
	if cacheControl := rr.Header().Get("Cache-Control"); !strings.Contains(cacheControl, "no-store") {
		t.Fatalf("API paths should have no-store cache control, got %q", cacheControl)
	}
}

func TestExtractClientIPPrecedence(t *testing.T) {
	if err := ConfigureClientIPPolicy(true, []string{"127.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure client IP policy: %v", err)
	}

	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		want       string
	}{
		{
			name:       "takes rightmost untrusted x-forwarded-for IP",
			remoteAddr: "127.0.0.1:12345",
			xff:        "203.0.113.50, 127.0.0.2",
			xri:        "203.0.113.100",
			want:       "203.0.113.50",
		},
		{
			name:       "ignores spoofed leftmost IP before trusted chain",
			remoteAddr: "127.0.0.1:12345",
			xff:        "198.51.100.10, 203.0.113.50, 127.0.0.2",
			want:       "203.0.113.50",
		},
		{
			name:       "falls back to x-real-ip",
			remoteAddr: "127.0.0.1:12345",
			xri:        "203.0.113.100",
			want:       "203.0.113.100",
		},
		{
			name:       "falls back to remote address",
			remoteAddr: "192.168.1.100:12345",
			want:       "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			if got := extractClientIP(req); got != tt.want {
				t.Fatalf("extractClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractIPUsesSameClientAddressPolicy(t *testing.T) {
	if err := ConfigureClientIPPolicy(true, []string{"192.168.1.0/24"}); err != nil {
		t.Fatalf("failed to configure client IP policy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Forwarded-For", "invalid, 203.0.113.50, 192.168.1.10")
	req.Header.Set("X-Real-IP", "203.0.113.100")

	gotClient := extractClientIP(req)
	gotRateLimit := extractIP(req)
	if gotClient != "203.0.113.50" {
		t.Fatalf("extractClientIP() = %q, want 203.0.113.50", gotClient)
	}
	if gotRateLimit != gotClient {
		t.Fatalf("extractIP() = %q, want same value as extractClientIP() = %q", gotRateLimit, gotClient)
	}
}

func TestRequestIDRequirementMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		required    bool
		headerValue string
		wantCode    int
	}{
		{name: "required and missing", required: true, headerValue: "", wantCode: http.StatusBadRequest},
		{name: "required and whitespace", required: true, headerValue: "   ", wantCode: http.StatusBadRequest},
		{name: "required and present", required: true, headerValue: "req-123", wantCode: http.StatusOK},
		{name: "optional and missing", required: false, headerValue: "", wantCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := RequestIDRequirementMiddleware(tt.required, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/_matrix/key/v2/query", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Request-ID", tt.headerValue)
			}
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantCode)
			}
			if tt.wantCode == http.StatusBadRequest {
				var payload map[string]string
				if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
					t.Fatalf("bad request payload should be JSON: %v", err)
				}
				if payload["errcode"] != "M_INVALID_PARAM" {
					t.Fatalf("errcode = %q, want M_INVALID_PARAM", payload["errcode"])
				}
				if called {
					t.Fatalf("next handler must not be called on rejection")
				}
			} else if !called {
				t.Fatalf("next handler should be called")
			}
		})
	}
}
