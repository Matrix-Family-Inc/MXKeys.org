/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package nettls

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// TestDialContextCancelsMidDial pins the ctx-aware dial contract:
// if the caller's context fires while DialContext is still trying
// to establish the TCP connection, the call must return
// ctx.Err() immediately rather than waiting for the per-attempt
// timeout to expire.
//
// We simulate a slow target by dialling an unroutable IP address.
// 203.0.113.1 is reserved by RFC 5737 (TEST-NET-3); no host
// answers on it, so the dial will block on SYN retries until the
// dialer's own timeout fires. The test cancels the context well
// before that timeout and asserts the return happens in tens of
// milliseconds rather than the full timeout window.
func TestDialContextCancelsMidDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay to catch the dial mid-flight.
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	conn, err := DialContext(ctx, "tcp", "203.0.113.1:1", 5*time.Second, Config{})
	elapsed := time.Since(start)

	if err == nil {
		_ = conn.Close()
		t.Fatalf("dial to TEST-NET-3 unexpectedly succeeded")
	}
	if !errors.Is(err, context.Canceled) && !isNetTimeoutOrCanceled(err) {
		t.Fatalf("expected context.Canceled or similar, got %v", err)
	}
	// Elapsed must be far below the 5 s hard-cap and close to the
	// 30 ms cancel deadline. Allow 500 ms headroom for scheduler
	// jitter on busy CI hosts.
	if elapsed > 500*time.Millisecond {
		t.Fatalf("DialContext took %v; expected prompt cancel near 30 ms", elapsed)
	}
}

// TestDialContextAlreadyCancelledReturnsImmediately checks the
// fast path: if ctx is already done on entry, DialContext must
// return without touching the network.
func TestDialContextAlreadyCancelledReturnsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	conn, err := DialContext(ctx, "tcp", "203.0.113.1:1", 5*time.Second, Config{})
	elapsed := time.Since(start)

	if err == nil {
		_ = conn.Close()
		t.Fatalf("expected an error on pre-cancelled ctx")
	}
	if !errors.Is(err, context.Canceled) {
		// Some platforms may wrap the error; accept any error that
		// unwraps to context.Canceled.
		if !isNetTimeoutOrCanceled(err) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("DialContext took %v; expected immediate return on cancelled ctx", elapsed)
	}
}

// isNetTimeoutOrCanceled matches the variety of errors Go's
// network stack may return when a ctx-cancelled dial is aborted:
// context.Canceled, context.DeadlineExceeded, or a net.OpError
// wrapping either.
func isNetTimeoutOrCanceled(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var oe *net.OpError
	if errors.As(err, &oe) {
		return oe.Timeout() || errors.Is(oe.Err, context.Canceled)
	}
	return false
}
