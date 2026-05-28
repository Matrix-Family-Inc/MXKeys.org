/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 09 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestNotarySetBlockPrivateIPs(t *testing.T) {
	n := &Notary{
		fetcher: NewFetcherWithConfig(FetcherConfig{Timeout: time.Second}),
	}

	n.SetBlockPrivateIPs(true)
	if !n.fetcher.blockPrivateIPs.Load() {
		t.Fatal("expected blockPrivateIPs to be enabled")
	}

	n.SetBlockPrivateIPs(false)
	if n.fetcher.blockPrivateIPs.Load() {
		t.Fatal("expected blockPrivateIPs to be disabled")
	}
}

func TestValidateReplicatedServerResponseRequiresCryptographicValidity(t *testing.T) {
	raw, _ := createSignedKeysResponse(t, "server.example.org")
	var response ServerKeysResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("failed to decode signed response: %v", err)
	}

	n := &Notary{
		fetcher: NewFetcherWithConfig(FetcherConfig{Timeout: time.Second}),
	}

	validated, err := n.validateReplicatedServerResponse("server.example.org", string(raw), response.ValidUntilTS)
	if err != nil {
		t.Fatalf("expected valid replicated response, got %v", err)
	}
	if validated.ServerName != "server.example.org" {
		t.Fatalf("unexpected server name %q", validated.ServerName)
	}

	tampered := strings.Replace(string(raw), `"server_name":"server.example.org"`, `"server_name":"other.example.org"`, 1)
	if _, err := n.validateReplicatedServerResponse("server.example.org", tampered, response.ValidUntilTS); err == nil {
		t.Fatal("expected tampered replicated response to be rejected")
	}
}

func TestNotaryCleanupRoutineStopsCleanly(t *testing.T) {
	n := &Notary{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n.StartCleanupRoutine(ctx, time.Hour)
	if n.cleanupCancel == nil {
		t.Fatal("expected cleanup routine cancel function to be registered")
	}

	n.StopCleanupRoutine()
	if n.cleanupCancel != nil {
		t.Fatal("expected cleanup cancel function to be cleared after stop")
	}

	// Must stay idempotent after the first stop.
	n.StopCleanupRoutine()
}

// TestIsRetryableStorageError validates the typed-error classification.
// Tests exercise real error shapes the PG driver surfaces rather than
// plain errors.New text (the previous string fallback is gone).
func TestIsRetryableStorageError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "driver.ErrBadConn", err: driver.ErrBadConn, want: true},
		{name: "wrapped driver.ErrBadConn", err: fmt.Errorf("exec: %w", driver.ErrBadConn), want: true},
		{name: "context.DeadlineExceeded", err: context.DeadlineExceeded, want: true},
		{name: "syscall ECONNRESET", err: syscall.ECONNRESET, want: true},
		{name: "os.SyscallError ECONNREFUSED",
			err:  &os.SyscallError{Syscall: "connect", Err: syscall.ECONNREFUSED},
			want: true},
		{name: "plain permanent (was matched by old string fallback, now ignored)",
			err:  errors.New("statement timeout"),
			want: false},
		{name: "plain unique-constraint", err: errors.New("duplicate key value violates unique constraint"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableStorageError(tt.err); got != tt.want {
				t.Fatalf("isRetryableStorageError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
