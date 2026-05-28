package keys

import (
	"testing"
	"time"
)

func TestParseServerNameHostname(t *testing.T) {
	tests := []struct {
		input        string
		expectedHost string
		expectedPort int
		expectedIsIP bool
	}{
		{"matrix.org", "matrix.org", 0, false},
		{"example.com", "example.com", 0, false},
		{"sub.domain.example.com", "sub.domain.example.com", 0, false},
	}

	for _, tt := range tests {
		host, port, isIP := parseServerName(tt.input)
		if host != tt.expectedHost {
			t.Errorf("%s: host = %q, want %q", tt.input, host, tt.expectedHost)
		}
		if port != tt.expectedPort {
			t.Errorf("%s: port = %d, want %d", tt.input, port, tt.expectedPort)
		}
		if isIP != tt.expectedIsIP {
			t.Errorf("%s: isIP = %v, want %v", tt.input, isIP, tt.expectedIsIP)
		}
	}
}

func TestParseServerNameWithPort(t *testing.T) {
	tests := []struct {
		input        string
		expectedHost string
		expectedPort int
	}{
		{"matrix.org:8448", "matrix.org", 8448},
		{"example.com:443", "example.com", 443},
		{"server.test:1", "server.test", 1},
		{"server.test:65535", "server.test", 65535},
	}

	for _, tt := range tests {
		host, port, _ := parseServerName(tt.input)
		if host != tt.expectedHost {
			t.Errorf("%s: host = %q, want %q", tt.input, host, tt.expectedHost)
		}
		if port != tt.expectedPort {
			t.Errorf("%s: port = %d, want %d", tt.input, port, tt.expectedPort)
		}
	}
}

func TestParseServerNameIPv4(t *testing.T) {
	tests := []struct {
		input        string
		expectedHost string
		expectedPort int
		expectedIsIP bool
	}{
		{"1.2.3.4", "1.2.3.4", 0, true},
		{"192.168.1.1", "192.168.1.1", 0, true},
		{"10.0.0.1:8448", "10.0.0.1", 8448, true},
		{"127.0.0.1:443", "127.0.0.1", 443, true},
	}

	for _, tt := range tests {
		host, port, isIP := parseServerName(tt.input)
		if host != tt.expectedHost {
			t.Errorf("%s: host = %q, want %q", tt.input, host, tt.expectedHost)
		}
		if port != tt.expectedPort {
			t.Errorf("%s: port = %d, want %d", tt.input, port, tt.expectedPort)
		}
		if isIP != tt.expectedIsIP {
			t.Errorf("%s: isIP = %v, want %v", tt.input, isIP, tt.expectedIsIP)
		}
	}
}

func TestParseServerNameIPv6(t *testing.T) {
	tests := []struct {
		input        string
		expectedHost string
		expectedPort int
		expectedIsIP bool
	}{
		{"[2001:db8::1]", "2001:db8::1", 0, true},
		{"[::1]", "::1", 0, true},
		{"[2001:db8::1]:8448", "2001:db8::1", 8448, true},
		{"[::1]:443", "::1", 443, true},
		{"[fe80::1]", "fe80::1", 0, true},
	}

	for _, tt := range tests {
		host, port, isIP := parseServerName(tt.input)
		if host != tt.expectedHost {
			t.Errorf("%s: host = %q, want %q", tt.input, host, tt.expectedHost)
		}
		if port != tt.expectedPort {
			t.Errorf("%s: port = %d, want %d", tt.input, port, tt.expectedPort)
		}
		if isIP != tt.expectedIsIP {
			t.Errorf("%s: isIP = %v, want %v", tt.input, isIP, tt.expectedIsIP)
		}
	}
}

func TestParseServerNameTrimsWhitespace(t *testing.T) {
	host, port, _ := parseServerName("  matrix.org:8448  ")
	if host != "matrix.org" {
		t.Errorf("expected trimmed host, got %q", host)
	}
	if port != 8448 {
		t.Errorf("expected port 8448, got %d", port)
	}
}

func TestParseServerNameInvalidPort(t *testing.T) {
	tests := []string{
		"matrix.org:0",
		"matrix.org:65536",
		"matrix.org:abc",
		"matrix.org:-1",
	}

	for _, input := range tests {
		_, port, _ := parseServerName(input)
		if port != 0 {
			t.Errorf("%s: expected port=0 for invalid port, got %d", input, port)
		}
	}
}

func TestResolvedServerURL(t *testing.T) {
	tests := []struct {
		host     string
		port     int
		expected string
	}{
		{"matrix.org", 8448, "https://matrix.org:8448"},
		{"192.168.1.1", 443, "https://192.168.1.1:443"},
		{"2001:db8::1", 8448, "https://[2001:db8::1]:8448"},
		{"example.com", 8080, "https://example.com:8080"},
	}

	for _, tt := range tests {
		rs := &ResolvedServer{Host: tt.host, Port: tt.port}
		if url := rs.URL(); url != tt.expected {
			t.Errorf("URL() = %q, want %q", url, tt.expected)
		}
	}
}

func TestWellKnownCacheMiss(t *testing.T) {
	cache := newWellKnownCache()

	_, ok := cache.get("unknown.server")
	if ok {
		t.Error("expected cache miss for unknown server")
	}
}

func TestWellKnownCacheSetGet(t *testing.T) {
	cache := newWellKnownCache()

	cache.set("test.server", &wellKnownEntry{
		host:      "delegated.server",
		port:      8448,
		fetchedAt: time.Now(),
		isError:   false,
	})

	entry, ok := cache.get("test.server")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.host != "delegated.server" {
		t.Errorf("host = %q, want %q", entry.host, "delegated.server")
	}
	if entry.port != 8448 {
		t.Errorf("port = %d, want %d", entry.port, 8448)
	}
}

func TestWellKnownCacheErrorEntry(t *testing.T) {
	cache := newWellKnownCache()

	cache.set("error.server", &wellKnownEntry{
		fetchedAt: time.Now(),
		isError:   true,
		errType:   errNotFound,
	})

	entry, ok := cache.get("error.server")
	if !ok {
		t.Fatal("expected cache hit for error entry")
	}
	if !entry.isError {
		t.Error("expected error entry")
	}
	if entry.errType != errNotFound {
		t.Errorf("errType = %v, want errNotFound", entry.errType)
	}
}

func TestSRVCacheMiss(t *testing.T) {
	cache := newSRVCache()

	_, ok := cache.get("unknown.server")
	if ok {
		t.Error("expected cache miss for unknown server")
	}
}

func TestSRVCacheSetGet(t *testing.T) {
	cache := newSRVCache()

	cache.set("test.server", &srvEntry{
		target:    "target.server",
		port:      8448,
		fetchedAt: time.Now(),
		isError:   false,
	})

	entry, ok := cache.get("test.server")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.target != "target.server" {
		t.Errorf("target = %q, want %q", entry.target, "target.server")
	}
	if entry.port != 8448 {
		t.Errorf("port = %d, want %d", entry.port, 8448)
	}
}

func TestSRVCacheErrorEntry(t *testing.T) {
	cache := newSRVCache()

	cache.set("error.server", &srvEntry{
		fetchedAt: time.Now(),
		isError:   true,
	})

	entry, ok := cache.get("error.server")
	if !ok {
		t.Fatal("expected cache hit for error entry")
	}
	if !entry.isError {
		t.Error("expected error entry")
	}
}

func TestNewResolver(t *testing.T) {
	r := NewResolver()

	if r == nil {
		t.Fatal("NewResolver returned nil")
	}
	if r.client == nil {
		t.Error("client is nil")
	}
	if r.cache == nil {
		t.Error("cache is nil")
	}
	if r.srvCache == nil {
		t.Error("srvCache is nil")
	}
}

func TestHashPreview(t *testing.T) {
	if got := hashPreview(""); got != "" {
		t.Errorf("hashPreview empty = %q, want empty", got)
	}

	short := "genesis"
	if got := hashPreview(short); got != short {
		t.Errorf("hashPreview short = %q, want %q", got, short)
	}

	long := "1234567890abcdefXYZ"
	if got := hashPreview(long); got != "1234567890abcdef..." {
		t.Errorf("hashPreview long = %q, want prefix preview", got)
	}
}
