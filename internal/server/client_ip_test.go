/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPIgnoresForwardedHeadersByDefault(t *testing.T) {
	if err := ConfigureClientIPPolicy(false, nil); err != nil {
		t.Fatalf("failed to configure policy: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.11:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.42")
	req.Header.Set("X-Real-IP", "198.51.100.55")

	got := clientIPFromRequest(req)
	if got != "203.0.113.11" {
		t.Fatalf("unexpected client IP: got %s want 203.0.113.11", got)
	}
}

func TestClientIPUsesForwardedHeadersOnlyFromTrustedProxy(t *testing.T) {
	if err := ConfigureClientIPPolicy(true, []string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure policy: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.42, 10.1.2.3")

	got := clientIPFromRequest(req)
	if got != "198.51.100.42" {
		t.Fatalf("unexpected client IP: got %s want 198.51.100.42", got)
	}
}

func TestClientIPPrefersRightmostUntrustedForwardedHop(t *testing.T) {
	if err := ConfigureClientIPPolicy(true, []string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure policy: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 203.0.113.50, 10.1.2.3")

	got := clientIPFromRequest(req)
	if got != "203.0.113.50" {
		t.Fatalf("unexpected client IP: got %s want 203.0.113.50", got)
	}
}

func TestClientIPRejectsForwardedHeadersFromUntrustedProxy(t *testing.T) {
	if err := ConfigureClientIPPolicy(true, []string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("failed to configure policy: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.11:443"
	req.Header.Set("X-Forwarded-For", "198.51.100.42")

	got := clientIPFromRequest(req)
	if got != "203.0.113.11" {
		t.Fatalf("unexpected client IP: got %s want 203.0.113.11", got)
	}
}
