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
	"crypto/tls"
	"errors"
	"net"
	"testing"
)

func TestSplitHostPortCases(t *testing.T) {
	cases := []struct {
		in       string
		wantHost string
		wantPort int
	}{
		{"matrix.org", "matrix.org", 0},
		{"matrix.org:8448", "matrix.org", 8448},
		{"matrix.org:", "matrix.org:", 0},
		{"matrix.org:invalid", "matrix.org:invalid", 0},
		{"[fe80::1]", "fe80::1", 0},
		{"[fe80::1]:8448", "fe80::1", 8448},
		{"[fe80::1]:999999", "fe80::1", 0},
		{"", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			h, p := splitHostPort(tc.in)
			if h != tc.wantHost || p != tc.wantPort {
				t.Fatalf("splitHostPort(%q) = (%q, %d), want (%q, %d)", tc.in, h, p, tc.wantHost, tc.wantPort)
			}
		})
	}
}

func TestResolveFederationTargetPrefersExplicitPort(t *testing.T) {
	h, p := resolveFederationTarget("matrix.org", 9000, &ServerInfoDNS{WellKnownServer: "ignored:443"})
	if h != "matrix.org" || p != 9000 {
		t.Fatalf("explicit port must win, got %s:%d", h, p)
	}
}

func TestResolveFederationTargetUsesWellKnown(t *testing.T) {
	h, p := resolveFederationTarget("matrix.org", 0, &ServerInfoDNS{WellKnownServer: "matrix.host:443"})
	if h != "matrix.host" || p != 443 {
		t.Fatalf("well-known must win over default, got %s:%d", h, p)
	}
}

func TestResolveFederationTargetFallsBackTo8448(t *testing.T) {
	h, p := resolveFederationTarget("matrix.org", 0, &ServerInfoDNS{})
	if h != "matrix.org" || p != 8448 {
		t.Fatalf("default fallback must be 8448, got %s:%d", h, p)
	}
}

func TestTLSVersionName(t *testing.T) {
	cases := map[uint16]string{
		tls.VersionTLS10: "TLS 1.0",
		tls.VersionTLS11: "TLS 1.1",
		tls.VersionTLS12: "TLS 1.2",
		tls.VersionTLS13: "TLS 1.3",
		0x0000:           "0x0000",
	}
	for in, want := range cases {
		if got := tlsVersionName(in); got != want {
			t.Fatalf("tlsVersionName(%04x) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyReachabilityError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"timeout", &net.DNSError{IsTimeout: true}, "timeout"},
		{"refused", errors.New("dial tcp: connection refused"), "connection refused"},
		{"nxdomain", errors.New("lookup: no such host"), "DNS lookup failed"},
		{"unreach", errors.New("network unreachable"), "network unreachable"},
		{"tls", errors.New("remote error: tls: handshake failure"), "TLS handshake failed"},
		{"other", errors.New("something else"), "unreachable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyReachabilityError(tc.err); got != tc.want {
				t.Fatalf("classify %v = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}
