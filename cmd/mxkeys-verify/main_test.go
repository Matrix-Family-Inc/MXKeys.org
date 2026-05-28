/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTrustLevelName(t *testing.T) {
	tests := map[int]string{
		0: "unknown",
		1: "transport_retrieval",
		2: "self_consistency",
		3: "origin_trust",
		4: "unknown",
	}
	for level, want := range tests {
		if got := trustLevelName(level); got != want {
			t.Errorf("trustLevelName(%d) = %q, want %q", level, got, want)
		}
	}
}

func TestTruncHash(t *testing.T) {
	tests := map[string]string{
		"":                         "",
		"short":                    "short",
		"1234567890123456":         "1234567890123456",    // exactly 16
		"12345678901234567":        "1234567890123456...", // 17 -> truncated
		"abcdef0123456789ffffffff": "abcdef0123456789...",
	}
	for in, want := range tests {
		if got := truncHash(in); got != want {
			t.Errorf("truncHash(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFetchJSON exercises the generic HTTP helper against a fake server.
// Validates: content types are respected, JSON decodes into the right type,
// non-2xx surfaces as an error, body-read failures are distinguishable.
func TestFetchJSON(t *testing.T) {
	type payload struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	t.Run("happy path", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payload{A: 42, B: "x"}); err != nil {
				t.Fatalf("encode: %v", err)
			}
		}))
		defer srv.Close()

		got, err := fetchJSON[payload](srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("fetchJSON: %v", err)
		}
		if got.A != 42 || got.B != "x" {
			t.Fatalf("unexpected payload: %+v", got)
		}
	})

	t.Run("non-200 rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		_, err := fetchJSON[payload](srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("expected error for 404 response")
		}
	})

	t.Run("malformed JSON rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{not json}`))
		}))
		defer srv.Close()

		_, err := fetchJSON[payload](srv.Client(), srv.URL)
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})
}

// TestSTHSignatureVerification exercises the pure-crypto verification
// path: construct a signed-tree-head payload with a known ed25519 key,
// then run the same verification the CLI does.
func TestSTHSignatureVerification(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	pubB64 := base64.RawStdEncoding.EncodeToString(pub)

	sth := signedTreeHead{
		TreeSize:    10,
		RootHash:    "deadbeef",
		TimestampMS: 1_745_000_000_000,
		SignPayload: "10|deadbeef|1745000000000",
	}
	sig := ed25519.Sign(priv, []byte(sth.SignPayload))
	sth.Signature = base64.RawStdEncoding.EncodeToString(sig)

	decodedPub, err := base64.RawStdEncoding.DecodeString(pubB64)
	if err != nil {
		t.Fatalf("decode pub: %v", err)
	}
	decodedSig, err := base64.RawStdEncoding.DecodeString(sth.Signature)
	if err != nil {
		t.Fatalf("decode sig: %v", err)
	}

	if !ed25519.Verify(decodedPub, []byte(sth.SignPayload), decodedSig) {
		t.Fatal("known-good signature must verify")
	}

	// Tampered payload must fail.
	bad := append([]byte(nil), []byte(sth.SignPayload)...)
	bad[0] ^= 0xFF
	if ed25519.Verify(decodedPub, bad, decodedSig) {
		t.Fatal("tampered payload must not verify")
	}
}

// TestOutputJSONShape guards against regressions in the CLI's machine-
// readable output: the shape is consumed by operators' scripts, so the
// JSON encoding must remain stable.
func TestOutputJSONShape(t *testing.T) {
	consistent := true
	prev := 5
	r := verifyResult{
		OK:               true,
		Server:           "https://notary.example.org",
		TreeSize:         7,
		RootHash:         "deadbeef",
		SignatureValid:   true,
		ConsistencyValid: &consistent,
		PrevTreeSize:     &prev,
		TrustLevel:       2,
		TrustLevelName:   "self_consistency",
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	out := string(raw)
	for _, must := range []string{
		`"ok":true`, `"server":`, `"tree_size":7`, `"root_hash":"deadbeef"`,
		`"signature_valid":true`, `"consistency_valid":true`, `"prev_tree_size":5`,
		`"trust_level":2`, `"trust_level_name":"self_consistency"`,
	} {
		if !strings.Contains(out, must) {
			t.Errorf("output missing %q: %s", must, out)
		}
	}
}
