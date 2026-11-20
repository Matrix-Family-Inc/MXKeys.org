/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"mxkeys/internal/zero/canonical"
	"mxkeys/internal/zero/signedjson"
)

// fakeTimeoutError implements net.Error with Timeout()=true, which is how
// the standard library signals deadline/timeout errors through the
// errors.As typing we rely on post-Phase3 audit.
type fakeTimeoutError struct{}

func (fakeTimeoutError) Error() string   { return "fake timeout" }
func (fakeTimeoutError) Timeout() bool   { return true }
func (fakeTimeoutError) Temporary() bool { return true }

// TestIsRetryableError validates the typed-error classification that
// replaced the prior string-matching fallback. Every case is a real
// error type the net / os / syscall packages produce at runtime.
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"nil", nil, false},
		{"plain text (was matched by old string fallback, now ignored)",
			errors.New("connection timeout"), false},
		{"net.Error timeout", fakeTimeoutError{}, true},
		{"net.OpError", &net.OpError{Op: "dial", Err: errors.New("x")}, true},
		{"net.DNSError", &net.DNSError{Err: "no such host", Name: "example.test"}, true},
		{"syscall ECONNREFUSED (wrapped in os.SyscallError)",
			&os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED}, true},
		{"syscall ECONNRESET bare", syscall.ECONNRESET, true},
		{"syscall EPIPE bare", syscall.EPIPE, true},
		{"syscall EHOSTUNREACH bare", syscall.EHOSTUNREACH, true},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF, true},
		{"plain permanent error", errors.New("invalid signature"), false},
		{"wrapped permanent", fmt.Errorf("decode: %w", errors.New("bad json")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.retryable {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.retryable)
			}
		})
	}
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

func pinnedNotaryFetcher(pub ed25519.PublicKey) *Fetcher {
	return NewFetcherWithConfig(FetcherConfig{
		Timeout: time.Second,
		TrustedNotaries: []TrustedNotaryKey{
			{ServerName: "notary.example.org", KeyID: "ed25519:notary", PublicKey: pub},
		},
	})
}

func TestVerifyNotarySignatureAcceptsSpecSignature(t *testing.T) {
	notaryPub, notaryPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := createSignedKeysResponse(t, "server.example.org")
	countersigned, err := signedjson.Sign(origin, "notary.example.org", "ed25519:notary", notaryPriv)
	if err != nil {
		t.Fatal(err)
	}

	if err := pinnedNotaryFetcher(notaryPub).verifyNotarySignature("notary.example.org", countersigned); err != nil {
		t.Fatalf("spec-compliant notary signature rejected: %v", err)
	}
}

func TestVerifyNotarySignatureRejectsSignatureOverSignatures(t *testing.T) {
	notaryPub, notaryPriv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := createSignedKeysResponse(t, "server.example.org")
	obj, err := signedjson.Decode(origin)
	if err != nil {
		t.Fatal(err)
	}
	// The pre-1.1 scheme signed the object together with the origin's
	// signatures. Matrix verifiers strip signatures, so such a signature
	// must not verify.
	msg, err := canonical.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	obj["signatures"].(map[string]interface{})["notary.example.org"] = map[string]interface{}{
		"ed25519:notary": base64.RawStdEncoding.EncodeToString(ed25519.Sign(notaryPriv, msg)),
	}
	raw, err := canonical.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}

	err = pinnedNotaryFetcher(notaryPub).verifyNotarySignature("notary.example.org", raw)
	if !errors.Is(err, ErrNotaryKeyMismatch) {
		t.Fatalf("expected ErrNotaryKeyMismatch, got %v", err)
	}
}

func TestVerifyNotarySignatureMissingPinnedKey(t *testing.T) {
	notaryPub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	origin, _ := createSignedKeysResponse(t, "server.example.org")
	if err := pinnedNotaryFetcher(notaryPub).verifyNotarySignature("notary.example.org", origin); err == nil {
		t.Fatal("a document without the pinned notary signature must be rejected")
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
