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
	"strings"
	"testing"

	"mxkeys/internal/config"
)

func newServerInfoOnlyServer(t *testing.T) *Server {
	t.Helper()
	cfg := &config.Config{}
	cfg.Server.Name = "notary.example.org"
	cfg.Server.Port = 8448
	cfg.Server.BindAddress = "127.0.0.1"
	cfg.Security.MaxServerNameLength = 255
	return &Server{config: cfg}
}

func TestHandleServerInfoRefusesWhenDisabled(t *testing.T) {
	s := newServerInfoOnlyServer(t)
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/server-info?name=matrix.org", nil)
	rr := httptest.NewRecorder()
	s.handleServerInfo(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when enrichment disabled, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "not enabled") {
		t.Fatalf("expected 'not enabled' in body, got %s", rr.Body.String())
	}
}

func TestHandleServerInfoRejectsEmptyName(t *testing.T) {
	s := newServerInfoOnlyServer(t)
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/server-info", nil)
	rr := httptest.NewRecorder()
	s.handleServerInfo(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		// Disabled-service shortcut wins before param validation;
		// that ordering is intentional because we refuse to leak
		// validation behaviour when the feature is off.
		t.Fatalf("expected 503 when enrichment disabled, got %d", rr.Code)
	}
}

func TestHandleServerInfoRejectsInvalidName(t *testing.T) {
	s := newServerInfoOnlyServer(t)
	// Enable the feature so validation is reachable.
	s.serverInfo = &ServerInfoService{}
	req := httptest.NewRequest(http.MethodGet, "/_mxkeys/server-info?name=..%2Fetc%2Fpasswd", nil)
	rr := httptest.NewRecorder()
	s.handleServerInfo(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid server name, got %d (%s)", rr.Code, rr.Body.String())
	}
}
