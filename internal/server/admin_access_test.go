/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"mxkeys/internal/config"
	"mxkeys/internal/keys"
)

func newAdminRouteTestServer(token string) *Server {
	s := &Server{
		config: &config.Config{
			Security: config.SecurityConfig{},
		},
		mux:              http.NewServeMux(),
		rateLimiter:      NewRateLimiter(DefaultRateLimitConfig()),
		analytics:        keys.NewAnalytics(nil, keys.AnalyticsConfig{Enabled: true}),
		adminAccessToken: token,
	}
	s.setupRoutes()
	return s
}

func TestAdminRoutesNotRegisteredWithoutToken(t *testing.T) {
	s := newAdminRouteTestServer("")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when admin token is absent", rr.Code)
	}
}

func TestAdminRoutesRequireToken(t *testing.T) {
	s := newAdminRouteTestServer("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestAdminRoutesAcceptBearerToken(t *testing.T) {
	s := newAdminRouteTestServer("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestAdminRoutesAcceptCaseInsensitiveBearerToken(t *testing.T) {
	s := newAdminRouteTestServer("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	req.Header.Set("Authorization", "bearer    secret-token")
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestOperationalAccessMiddlewareRequiresToken(t *testing.T) {
	s := newAdminRouteTestServer("secret-token")
	handler := s.withOperationalAccess(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/status", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestOperationalAccessMiddlewareAcceptsBearerToken(t *testing.T) {
	s := newAdminRouteTestServer("secret-token")
	handler := s.withOperationalAccess(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/status", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestOperationalAccessHandlerProtectsMetricsStyleHandlers(t *testing.T) {
	s := newAdminRouteTestServer("secret-token")
	handler := s.withOperationalAccessHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/metrics", nil)
	req.Header.Set("X-MXKeys-Admin-Token", "secret-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}
