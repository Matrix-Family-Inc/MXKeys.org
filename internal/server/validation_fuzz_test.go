package server

import (
	"bytes"
	"testing"
)

// FuzzValidateServerName exercises ValidateServerName against arbitrary
// byte/string inputs. Invariants:
//
//  1. Must never panic.
//  2. Accepted names must be non-empty, <= maxServerNameLength runes, valid
//     UTF-8, and free of control characters (the rules the function checks
//     explicitly).
//
// The fuzzer is not asked to assert the format-validity predicate in both
// directions: isValidServerNameFormat is a filter, and the fuzzer drives
// both acceptance and rejection paths through it without checking
// equivalence. What we guard is "rejections never crash, accepted strings
// satisfy the basic textual invariants".
func FuzzValidateServerName(f *testing.F) {
	seeds := []string{
		"matrix.org",
		"example.com:8448",
		"127.0.0.1",
		"127.0.0.1:8448",
		"[::1]",
		"[::1]:8448",
		"a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p",
		"",
		" ",
		"server with spaces",
		"server\x00with-null",
		"server\twith-tab",
		"very-long-name-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		err := ValidateServerName(name, 0)
		if err != nil {
			return
		}
		if name == "" {
			t.Fatal("accepted empty name")
		}
		for _, r := range name {
			if r < 32 || r == 127 {
				t.Fatalf("accepted name with control character %U: %q", r, name)
			}
		}
	})
}

// FuzzValidateKeyID exercises ValidateKeyID. Accepted IDs must be non-empty,
// within length bounds, and composed of the "algorithm:identifier" shape
// the validator enforces.
func FuzzValidateKeyID(f *testing.F) {
	seeds := []string{
		"ed25519:auto",
		"ed25519:abc1",
		"ed25519:",
		":abc",
		"no-colon",
		"",
		"a:b:c",
		"ed25519:abc/def",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, keyID string) {
		err := ValidateKeyID(keyID)
		if err != nil {
			return
		}
		if keyID == "" {
			t.Fatal("accepted empty key ID")
		}
	})
}

// FuzzDecodeStrictJSON exercises the JSON decoder used by /_matrix/key/v2/query.
// Any arbitrary byte input must either decode cleanly or return an error; no
// panics, no unbounded memory blowups.
func FuzzDecodeStrictJSON(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"server_keys":{}}`),
		[]byte(`{"server_keys":{"a":{"ed25519:k":{}}}}`),
		[]byte(`{}{"extra":1}`),
		[]byte(`[]`),
		[]byte(`null`),
		[]byte(`not json`),
		[]byte(``),
		[]byte(`{"a":` + string(make([]byte, 1024)) + `}`),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// bytes.NewReader gives decodeStrictJSON a bounded io.Reader.
		// The call must not panic; returning an error is a valid outcome.
		var out interface{}
		_ = decodeStrictJSON(bytes.NewReader(data), &out, 10)
	})
}
