/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
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

func newEnterpriseRouteTestServer(token string) *Server {
	s := &Server{
		config: &config.Config{
			Security: config.SecurityConfig{},
		},
		mux:                   http.NewServeMux(),
		rateLimiter:           NewRateLimiter(DefaultRateLimitConfig()),
		analytics:             keys.NewAnalytics(nil, keys.AnalyticsConfig{Enabled: true}),
		enterpriseAccessToken: token,
	}
	s.setupRoutes()
	return s
}

func TestEnterpriseRoutesNotRegisteredWithoutToken(t *testing.T) {
	s := newEnterpriseRouteTestServer("")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when enterprise token is absent", rr.Code)
	}
}

func TestEnterpriseRoutesRequireToken(t *testing.T) {
	s := newEnterpriseRouteTestServer("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestEnterpriseRoutesAcceptBearerToken(t *testing.T) {
	s := newEnterpriseRouteTestServer("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestEnterpriseRoutesAcceptCaseInsensitiveBearerToken(t *testing.T) {
	s := newEnterpriseRouteTestServer("secret-token")
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/analytics/summary", nil)
	req.Header.Set("Authorization", "bearer    secret-token")
	rr := httptest.NewRecorder()

	s.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}
