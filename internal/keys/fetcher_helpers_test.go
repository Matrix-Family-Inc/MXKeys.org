/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"mxkeys/internal/zero/canonical"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		errMsg    string
		retryable bool
	}{
		{"connection timeout", true},
		{"connection refused", true},
		{"connection reset by peer", true},
		{"no such host", true},
		{"i/o timeout", true},
		{"temporary failure in DNS", true},
		{"invalid JSON", false},
		{"server error", false},
	}

	for _, tt := range tests {
		err := &testError{msg: tt.errMsg}
		result := isRetryableError(err)
		if result != tt.retryable {
			t.Errorf("isRetryableError(%q) = %v, want %v", tt.errMsg, result, tt.retryable)
		}
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestKeyErrorFormat(t *testing.T) {
	err := &KeyError{
		Op:         "fetch",
		ServerName: "test.server",
		Err:        ErrFetchFailed,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("error string should not be empty")
	}

	if err.Unwrap() != ErrFetchFailed {
		t.Error("Unwrap should return underlying error")
	}
}

func TestFetcherDefaultConfig(t *testing.T) {
	f := NewFetcher([]string{"fallback.test"}, 30*time.Second)

	if f.retryAttempts != defaultRetryAttempts {
		t.Errorf("default retry attempts = %d, want %d", f.retryAttempts, defaultRetryAttempts)
	}
}

func TestTrustedNotaryKeyStruct(t *testing.T) {
	key := TrustedNotaryKey{
		ServerName: "notary.test",
		KeyID:      "ed25519:notary",
		PublicKey:  make([]byte, 32),
	}

	if key.ServerName != "notary.test" {
		t.Errorf("ServerName = %q, want notary.test", key.ServerName)
	}

	if key.KeyID != "ed25519:notary" {
		t.Errorf("KeyID = %q, want ed25519:notary", key.KeyID)
	}

	if len(key.PublicKey) != 32 {
		t.Errorf("PublicKey length = %d, want 32", len(key.PublicKey))
	}
}

func TestVerifyNotarySignatureValid(t *testing.T) {
	notaryPub, notaryPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate notary key: %v", err)
	}

	f := NewFetcherWithConfig(FetcherConfig{
		Timeout: time.Second,
		TrustedNotaries: []TrustedNotaryKey{
			{
				ServerName: "notary.example.org",
				KeyID:      "ed25519:notary",
				PublicKey:  notaryPub,
			},
		},
	})

	resp := &ServerKeysResponse{
		ServerName:   "server.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:server": {Key: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))},
		},
		OldVerifyKeys: map[string]OldKeyResponse{},
		Signatures: map[string]map[string]string{
			"server.example.org": {"ed25519:server": "origin-sig"},
		},
	}

	toSign := map[string]interface{}{
		"server_name":     resp.ServerName,
		"valid_until_ts":  resp.ValidUntilTS,
		"verify_keys":     resp.VerifyKeys,
		"old_verify_keys": resp.OldVerifyKeys,
		"signatures": map[string]map[string]string{
			"server.example.org": {"ed25519:server": "origin-sig"},
		},
	}
	canonicalBytes, err := canonical.Marshal(toSign)
	if err != nil {
		t.Fatalf("failed to canonicalize: %v", err)
	}

	resp.Signatures["notary.example.org"] = map[string]string{
		"ed25519:notary": base64.RawStdEncoding.EncodeToString(ed25519.Sign(notaryPriv, canonicalBytes)),
	}

	if err := f.verifyNotarySignature("notary.example.org", resp); err != nil {
		t.Fatalf("verifyNotarySignature returned error: %v", err)
	}
}

func TestVerifyNotarySignatureMismatch(t *testing.T) {
	notaryPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate notary key: %v", err)
	}

	f := NewFetcherWithConfig(FetcherConfig{
		Timeout: time.Second,
		TrustedNotaries: []TrustedNotaryKey{
			{
				ServerName: "notary.example.org",
				KeyID:      "ed25519:notary",
				PublicKey:  notaryPub,
			},
		},
	})

	resp := &ServerKeysResponse{
		ServerName:   "server.example.org",
		ValidUntilTS: time.Now().Add(time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:server": {Key: base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.PublicKeySize))},
		},
		OldVerifyKeys: map[string]OldKeyResponse{},
		Signatures: map[string]map[string]string{
			"server.example.org": {"ed25519:server": "origin-sig"},
			"notary.example.org": {"ed25519:notary": base64.RawStdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))},
		},
	}

	err = f.verifyNotarySignature("notary.example.org", resp)
	if !errors.Is(err, ErrNotaryKeyMismatch) {
		t.Fatalf("expected ErrNotaryKeyMismatch, got %v", err)
	}
}

func TestReadLimitedBody(t *testing.T) {
	body, err := readLimitedBody(io.NopCloser(strings.NewReader("abc")), 3)
	if err != nil {
		t.Fatalf("readLimitedBody failed: %v", err)
	}
	if string(body) != "abc" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestReadLimitedBodyTooLarge(t *testing.T) {
	_, err := readLimitedBody(io.NopCloser(strings.NewReader("abcd")), 3)
	if err == nil {
		t.Fatal("expected body-too-large error")
	}
}
