/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package raft

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWALGroupCommitAmortizesFsync exercises the batching path: many
// concurrent Appends must all complete without deadlock and the
// resulting WAL must contain every entry. Per-call ordering is not
// guaranteed between goroutines; the contract is "every entry lands
// on disk before its Append returns".
func TestWALGroupCommitAmortizesFsync(t *testing.T) {
	w, _ := newTestWAL(t)

	const n = 64
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			done <- w.Append(LogEntry{Index: uint64(i + 1), Term: 1, Command: json.RawMessage(`"x"`)})
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-done; err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != n {
		t.Fatalf("expected %d entries, got %d", n, len(got))
	}
}

// TestWALGroupCommitActuallyBatches asserts that N concurrent Appends
// complete in fewer fsync windows than N. A genuinely batched writer
// finishes in O(N / batch) windows; a "one fsync per Append" writer
// would need O(N) windows.
func TestWALGroupCommitActuallyBatches(t *testing.T) {
	w, _ := newTestWAL(t)

	const n = 50
	start := time.Now()

	done := make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			done <- w.Append(LogEntry{Index: uint64(i + 1), Term: 1, Command: json.RawMessage(`"x"`)})
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-done; err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// walGroupFlushInterval is 2 ms. A "one fsync per call" design
	// would pay at least N * flush-interval; a batched one pays a
	// handful of intervals total. Bound: elapsed < N * interval.
	maxAllowed := time.Duration(n) * walGroupFlushInterval
	if elapsed >= maxAllowed {
		t.Fatalf("concurrent Appends took %v, at or above the N-serialized-ticks bound %v; group commit may have regressed", elapsed, maxAllowed)
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != n {
		t.Fatalf("expected %d entries, got %d", n, len(got))
	}
}

// TestWALDurabilityContract verifies that Append returns only after
// the entry is fsync'd. Reopening the file in a fresh WAL without
// closing the original simulates a process crash.
func TestWALDurabilityContract(t *testing.T) {
	dir := t.TempDir()

	w, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	if err := w.Append(LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"durable"`)}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()

	got, err := w2.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("entry not durable after Append return: %+v", got)
	}

	_ = w.Close()
}
