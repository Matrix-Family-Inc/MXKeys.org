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
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

// TestWALAppendSurfacesFsyncFailure verifies the durability contract
// under a synthetic fsync error. An Append call must return a non-nil
// error when the batcher's fsync fails; the caller MUST NOT be told
// the entry is durable.
func TestWALAppendSurfacesFsyncFailure(t *testing.T) {
	// Install a fault-injection hook that fails the next fsync and then
	// restores normal behaviour. Restore deferred to t.Cleanup so other
	// tests are not affected even on panic.
	t.Cleanup(func() { syncHookForTest = nil })

	var fails atomic.Int32
	fails.Store(1)
	syncHookForTest = func(f *os.File) error {
		if fails.Add(-1) >= 0 {
			return errors.New("synthetic fsync failure")
		}
		return f.Sync()
	}

	w, _ := newTestWAL(t)
	err := w.Append(LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"x"`)})
	if err == nil {
		t.Fatal("Append must return an error when fsync fails")
	}
}

// TestRaftSubmitTruncatesTailOnPersistFailure is the end-to-end
// correctness check for the persist-outside-lock path. The in-memory
// log must not keep an entry whose fsync failed; otherwise a later
// successful Submit would race with a retained zombie entry.
func TestRaftSubmitTruncatesTailOnPersistFailure(t *testing.T) {
	t.Cleanup(func() { syncHookForTest = nil })

	n := NewNode(Config{
		NodeID:       "n1",
		SharedSecret: "test-wal-hmac-key-32-bytes-or-so!",
	})
	dir := t.TempDir()
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })

	// Transition to leader without running full Start so we have a
	// quiet node to drive synchronously.
	n.mu.Lock()
	n.state = Leader
	n.currentTerm = 1
	n.leaderId = "n1"
	n.mu.Unlock()

	// Fail the next fsync exactly once.
	var once sync.Once
	syncHookForTest = func(f *os.File) error {
		var fail bool
		once.Do(func() { fail = true })
		if fail {
			return errors.New("synthetic fsync failure")
		}
		return f.Sync()
	}

	// A Submit that loses its fsync must return an error AND must
	// leave n.log empty so a retry can reclaim the same index.
	err := n.Submit(t.Context(), []byte(`"first"`))
	if err == nil {
		t.Fatal("Submit must fail when fsync fails")
	}
	n.mu.RLock()
	logLen := len(n.log)
	n.mu.RUnlock()
	if logLen != 0 {
		t.Fatalf("in-memory log must be truncated after persist failure, got %d entries", logLen)
	}

	// The next Submit must succeed and take the same index.
	if err := n.Submit(t.Context(), []byte(`"second"`)); err != nil {
		t.Fatalf("retry Submit: %v", err)
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	if len(n.log) != 1 {
		t.Fatalf("expected one log entry after successful retry, got %d", len(n.log))
	}
	if n.log[0].Index != 1 {
		t.Fatalf("retry took index %d, want 1", n.log[0].Index)
	}
}
