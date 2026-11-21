/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

// offlineResolver never touches the network: address lookups fail and SRV
// answers come from srv, keyed by the full record name.
func offlineResolver(srv map[string]*net.SRV) (*Resolver, *[]string) {
	r := NewResolver()
	var asked []string
	r.lookupIP = func(context.Context, string, string) ([]net.IP, error) {
		return nil, &net.DNSError{Err: "no such host", IsNotFound: true}
	}
	r.lookupSRV = func(_ context.Context, service, proto, name string) (string, []*net.SRV, error) {
		full := "_" + service + "._" + proto + "." + name
		asked = append(asked, full)
		if rec, ok := srv[full]; ok {
			return "", []*net.SRV{rec}, nil
		}
		return "", nil, &net.DNSError{Err: "no such host", IsNotFound: true}
	}
	return r, &asked
}

func TestResolveBindsTLSNameAndHostToDiscoveryName(t *testing.T) {
	r, asked := offlineResolver(map[string]*net.SRV{
		"_matrix-fed._tcp.srv.example": {Target: "federation.hosting.example.", Port: 8449},
	})

	cases := []struct {
		name               string
		host               string
		port               int
		tlsName, authority string
	}{
		{"1.2.3.4", "1.2.3.4", 8448, "1.2.3.4", "1.2.3.4"},
		{"[2001:db8::1]:8443", "2001:db8::1", 8443, "2001:db8::1", "[2001:db8::1]:8443"},
		{"explicit.example:8443", "explicit.example", 8443, "explicit.example", "explicit.example:8443"},
		{"srv.example", "federation.hosting.example", 8449, "srv.example", "srv.example"},
		{"plain.example", "plain.example", 8448, "plain.example", "plain.example"},
	}
	for _, tc := range cases {
		got, err := r.ResolveServerName(context.Background(), tc.name)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got.Host != tc.host || got.Port != tc.port || got.TLSServerName() != tc.tlsName || got.HostHeader() != tc.authority {
			t.Fatalf("%s: got host=%s port=%d tls=%s authority=%s", tc.name, got.Host, got.Port, got.TLSServerName(), got.HostHeader())
		}
	}

	for _, q := range *asked {
		if len(q) > 1 && q[1] == '_' {
			t.Fatalf("SRV query name must not be double-prefixed: %s", q)
		}
	}
}

func TestResolveDelegatedBindsToDelegatedName(t *testing.T) {
	r, _ := offlineResolver(map[string]*net.SRV{
		"_matrix-fed._tcp.matrix.example.org": {Target: "edge.cdn.example.", Port: 443},
	})
	cases := []struct {
		entry              wellKnownEntry
		host               string
		port               int
		tlsName, authority string
	}{
		{wellKnownEntry{host: "matrix.example.org"}, "edge.cdn.example", 443, "matrix.example.org", "matrix.example.org"},
		{wellKnownEntry{host: "matrix.example.org", port: 8448}, "matrix.example.org", 8448, "matrix.example.org", "matrix.example.org:8448"},
		{wellKnownEntry{host: "no-srv.example.org"}, "no-srv.example.org", 8448, "no-srv.example.org", "no-srv.example.org"},
		{wellKnownEntry{host: "203.0.113.9"}, "203.0.113.9", 8448, "203.0.113.9", "203.0.113.9"},
	}
	for _, tc := range cases {
		entry := tc.entry
		got, err := r.resolveDelegated(context.Background(), &entry, "example.org")
		if err != nil {
			t.Fatal(err)
		}
		if got.Host != tc.host || got.Port != tc.port || got.TLSServerName() != tc.tlsName || got.HostHeader() != tc.authority {
			t.Fatalf("%+v: got host=%s port=%d tls=%s authority=%s", tc.entry, got.Host, got.Port, got.TLSServerName(), got.HostHeader())
		}
	}
}

func TestWellKnownDialRefusesPrivateAddresses(t *testing.T) {
	r := NewResolver()
	r.lookupIP = func(context.Context, string, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("203.0.113.5"), net.ParseIP("10.0.0.7")}, nil
	}
	for _, address := range []string{"internal.example:443", "127.0.0.1:443", "[::1]:443", "169.254.169.254:443"} {
		if _, err := r.dialWellKnown(context.Background(), "tcp", address); err == nil {
			t.Fatalf("dial to %s must be refused", address)
		}
	}
}

func TestValidDelegatedHost(t *testing.T) {
	for _, ok := range []string{"matrix.example.org", "1.2.3.4", "2001:db8::1"} {
		if !validDelegatedHost(ok) {
			t.Fatalf("%q must be accepted", ok)
		}
	}
	for _, bad := range []string{"", "evil.example/path", "user@host", "host?x", ".lead", "-lead", "sp ace"} {
		if validDelegatedHost(bad) {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}

func TestClientForResolvedVerifiesDiscoveryName(t *testing.T) {
	f := NewFetcherWithConfig(FetcherConfig{Timeout: time.Second})
	client := f.clientForResolved(&ResolvedServer{Host: "edge.cdn.example", Port: 443, TLSName: "matrix.example.org", PinnedIPs: []string{"203.0.113.5"}})
	transport := client.Transport.(*http.Transport)
	if transport.TLSClientConfig.ServerName != "matrix.example.org" {
		t.Fatalf("TLS server name = %q", transport.TLSClientConfig.ServerName)
	}
	if !transport.DisableKeepAlives {
		t.Fatal("single-use transports must not keep idle connections")
	}
	if f.client.Transport.(*http.Transport).TLSClientConfig.ServerName != "" {
		t.Fatal("the shared transport must stay untouched")
	}
}

func TestBoundedCacheEvictsAndCountsNegatives(t *testing.T) {
	c := newBoundedCache(2, func(bool) time.Duration { return time.Hour }, func(negative bool) bool { return negative })
	now := time.Now()
	c.set("a", true, now)
	c.set("b", false, now)
	c.set("c", true, now)

	if _, ok := c.get("a", now); ok {
		t.Fatal("oldest entry must be evicted at capacity")
	}
	if total, negative := c.stats(); total != 2 || negative != 1 {
		t.Fatalf("stats = %d/%d, want 2/1", total, negative)
	}
	if _, ok := c.get("c", now.Add(2*time.Hour)); ok {
		t.Fatal("expired entry must not be returned")
	}
	if total, negative := c.stats(); total != 1 || negative != 0 {
		t.Fatalf("stats after expiry = %d/%d, want 1/0", total, negative)
	}
}

func TestIsNXDOMAIN(t *testing.T) {
	if !isNXDOMAIN(&net.DNSError{IsNotFound: true}) || isNXDOMAIN(errors.New("x")) || isNXDOMAIN(nil) {
		t.Fatal("isNXDOMAIN classification is wrong")
	}
}
