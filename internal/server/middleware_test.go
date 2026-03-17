package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDGeneratedIfMissing(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("X-Request-ID should be generated")
	}
	if len(requestID) != 32 {
		t.Errorf("generated X-Request-ID should be 32 hex chars, got %d", len(requestID))
	}
}

func TestRequestIDPassedThrough(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id-12345")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	requestID := rr.Header().Get("X-Request-ID")
	if requestID != "my-custom-id-12345" {
		t.Errorf("X-Request-ID should be passed through, got %q", requestID)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		got := rr.Header().Get(header)
		if got != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, got)
		}
	}
}

func TestCacheControlOnAPIPath(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/_matrix/key/v2/query", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") {
		t.Errorf("API paths should have no-store cache control, got %q", cacheControl)
	}
}

func TestExtractClientIPFromRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	ip := extractClientIP(req)
	if ip != "192.168.1.100" {
		t.Errorf("expected 192.168.1.100, got %s", ip)
	}
}

func TestExtractClientIPFromXForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

	ip := extractClientIP(req)
	if ip != "203.0.113.50" {
		t.Errorf("expected first IP from X-Forwarded-For, got %s", ip)
	}
}

func TestExtractClientIPFromXRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.100")

	ip := extractClientIP(req)
	if ip != "203.0.113.100" {
		t.Errorf("expected X-Real-IP, got %s", ip)
	}
}

func TestXForwardedForPrecedence(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	req.Header.Set("X-Real-IP", "203.0.113.100")

	ip := extractClientIP(req)
	if ip != "203.0.113.50" {
		t.Errorf("X-Forwarded-For should take precedence, got %s", ip)
	}
}

func TestGetRequestIDFromContext(t *testing.T) {
	var capturedID string

	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedID != "test-request-id" {
		t.Errorf("expected test-request-id from context, got %q", capturedID)
	}
}

func TestMiddlewareChain(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RequestIDMiddleware(SecurityHeadersMiddleware(innerHandler))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID missing after chain")
	}
	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("security header missing after chain")
	}
}
