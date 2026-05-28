package server

import (
	"strings"
	"testing"
)

func TestValidServerNames(t *testing.T) {
	validNames := []string{
		"matrix.org",
		"example.com",
		"sub.domain.example.com",
		"matrix.org:8448",
		"example.com:443",
		"1.2.3.4",
		"1.2.3.4:8448",
		"192.168.1.1",
		"10.0.0.1:8080",
		"a.b",
		"test-server.example.org",
		"server123.test.com",
		"xn--e1afmkfd.xn--p1ai",
	}

	for _, name := range validNames {
		if err := ValidateServerName(name, 255); err != nil {
			t.Errorf("server name %q should be valid, got: %v", name, err)
		}
	}
}

func TestInvalidServerNames(t *testing.T) {
	invalidNames := []string{
		"",
		"../../../etc/passwd",
		"server\x00name",
		"server\nname",
		"-invalid.com",
		"invalid-.com",
		".invalid.com",
		"invalid..com",
		"server name with spaces",
		"server\ttab",
		// Raw Cyrillic IDN "example.rf" - must be punycoded before
		// reaching the validator; kept as \u escapes so the source
		// file stays ASCII-only.
		"\u043f\u0440\u0438\u043c\u0435\u0440.\u0440\u0444",
	}

	for _, name := range invalidNames {
		if err := ValidateServerName(name, 255); err == nil {
			t.Errorf("server name %q should be invalid", name)
		}
	}
}

func TestServerNameTooLong(t *testing.T) {
	longName := strings.Repeat("a", 256)
	if err := ValidateServerName(longName, 255); err == nil {
		t.Error("server name > 255 chars should be rejected")
	}

	exactName := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63)
	if err := ValidateServerName(exactName, 255); err != nil {
		t.Errorf("valid long name should pass: %v", err)
	}
}

func TestServerNameWithPort(t *testing.T) {
	tests := []struct {
		name    string
		isValid bool
	}{
		{"example.com:8448", true},
		{"example.com:1", true},
		{"example.com:65535", true},
		{"example.com:0", false},
		{"example.com:65536", false},
		{"example.com:abc", false},
		{"example.com:", false},
		{"example.com:99999", false},
	}

	for _, tt := range tests {
		err := ValidateServerName(tt.name, 255)
		if tt.isValid && err != nil {
			t.Errorf("%q should be valid, got: %v", tt.name, err)
		}
		if !tt.isValid && err == nil {
			t.Errorf("%q should be invalid", tt.name)
		}
	}
}

func TestIPv6ServerNames(t *testing.T) {
	tests := []struct {
		name    string
		isValid bool
	}{
		{"[2001:db8::1]", true},
		{"[2001:db8::1]:8448", true},
		{"[::1]", true},
		{"[::1]:443", true},
		{"[2001:db8::1]:65535", true},
		{"[fe80::1%eth0]", false}, // zone ID not allowed
		{"2001:db8::1", false},    // missing brackets
		{"[2001:db8::zz]", false}, // invalid hex
		{"[2001:db8::1]:0", false},
		{"[2001:db8::1]:65536", false},
	}

	for _, tt := range tests {
		err := ValidateServerName(tt.name, 255)
		if tt.isValid && err != nil {
			t.Errorf("%q should be valid, got: %v", tt.name, err)
		}
		if !tt.isValid && err == nil {
			t.Errorf("%q should be invalid", tt.name)
		}
	}
}

func TestValidKeyIDs(t *testing.T) {
	validIDs := []string{
		"ed25519:abc",
		"ed25519:ABC",
		"ed25519:abc123",
		"ed25519:a_b_c",
		"ed25519:key_id_123",
		"ed25519:mxkeys",
	}

	for _, id := range validIDs {
		if err := ValidateKeyID(id); err != nil {
			t.Errorf("key ID %q should be valid, got: %v", id, err)
		}
	}
}

func TestInvalidKeyIDs(t *testing.T) {
	invalidIDs := []string{
		"",
		"rsa:abc",
		"ecdsa:abc",
		"ed25519",
		"ed25519:",
		"abc",
		"ed25519:abc-def", // hyphen not allowed
		"ed25519:abc.def", // dot not allowed
		"ed25519:abc def", // space not allowed
		"ed25519:abc/def", // slash not allowed
		":abc",
		"ed25519:abc:def",
	}

	for _, id := range invalidIDs {
		if err := ValidateKeyID(id); err == nil {
			t.Errorf("key ID %q should be invalid", id)
		}
	}
}

func TestKeyIDTooLong(t *testing.T) {
	longID := "ed25519:" + strings.Repeat("a", 200)
	if err := ValidateKeyID(longID); err == nil {
		t.Error("key ID > 128 chars should be rejected")
	}
}

func TestEmptyServerName(t *testing.T) {
	err := ValidateServerName("", 255)
	if err == nil {
		t.Error("empty server name should be rejected")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty, got: %v", err)
	}
}

func TestEmptyKeyID(t *testing.T) {
	err := ValidateKeyID("")
	if err == nil {
		t.Error("empty key ID should be rejected")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty, got: %v", err)
	}
}

func TestUnsupportedKeyAlgorithm(t *testing.T) {
	err := ValidateKeyID("rsa:keyid")
	if err == nil {
		t.Error("non-ed25519 key should be rejected")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error should mention unsupported, got: %v", err)
	}
}
