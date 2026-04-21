/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package raft

import (
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

// TestCompactLogSerializesWithInstallSnapshot pins the invariant
// introduced by snapMu: the on-disk raft.snapshot file and the
// matching in-memory (snapshotIndex, snapshotTerm) bookkeeping
// cannot roll backwards across a race between CompactLog and
// handleInstallSnapshot.
//
// Scenario:
//
//  1. A single-node cluster has committed/applied three entries.
//  2. CompactLog runs with a provider that blocks just long enough
//     for an InstallSnapshot handler to try to interleave.
//  3. An InstallSnapshot RPC for a strictly newer index
//     (idx=100) arrives in parallel.
//
// Without serialisation CompactLog could rename its older snapshot
// file on top of the newer one persisted by InstallSnapshot. With
// snapMu both sequences are atomic; whichever one runs first
// leaves the disk in a state the other's pre-validation rejects.
// In every interleaving the final on-disk LastIncludedIndex must
// be >= the higher of the two values (100), never the older one.
func TestCompactLogSerializesWithInstallSnapshot(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:          "n",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })

	// Seed log state at index=3 so CompactLog has something to
	// persist and the provider index is legal under termAt.
	n.mu.Lock()
	n.currentTerm = 1
	n.log = []LogEntry{
		{Index: 1, Term: 1, Command: json.RawMessage(`"a"`)},
		{Index: 2, Term: 1, Command: json.RawMessage(`"b"`)},
		{Index: 3, Term: 1, Command: json.RawMessage(`"c"`)},
	}
	for _, e := range n.log {
		if err := n.wal.Append(e); err != nil {
			n.mu.Unlock()
			t.Fatalf("wal seed: %v", err)
		}
	}
	n.commitIndex = 3
	n.lastApplied = 3
	n.mu.Unlock()

	// Block inside the provider until the InstallSnapshot handler
	// has had its chance to try to start. snapMu MUST prevent it
	// from entering its own handler body until CompactLog releases.
	providerEntered := make(chan struct{})
	providerRelease := make(chan struct{})
	n.SetSnapshotProvider(func() ([]byte, uint64, error) {
		close(providerEntered)
		<-providerRelease
		return []byte("old"), 3, nil
	})
	n.SetSnapshotInstaller(func(r io.Reader, _ int64, _, _ uint64) error {
		_, _ = io.Copy(io.Discard, r)
		return nil
	})

	// Launch the CompactLog; it will block in the provider.
	compactDone := make(chan error, 1)
	go func() {
		compactDone <- n.CompactLog()
	}()
	<-providerEntered

	// With CompactLog blocked under snapMu, fire an InstallSnapshot
	// for a strictly newer (idx=100, term=1) tuple. The handler
	// MUST block on snapMu until we unblock the provider.
	installDone := make(chan *RPCMessage, 1)
	go func() {
		req := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: 100, LastIncludedTerm: 1,
			Offset: 0, Done: true, Data: []byte("new"),
		}
		payload, _ := json.Marshal(req)
		installDone <- n.handleInstallSnapshot(&RPCMessage{
			Type:    MsgInstallSnapshot,
			Payload: payload,
		})
	}()

	// Give the InstallSnapshot goroutine some wall-clock time to
	// reach snapMu.Lock and block there. If snapMu were missing it
	// would race ahead and complete before CompactLog returns.
	time.Sleep(50 * time.Millisecond)

	// Verify the install has NOT completed yet.
	select {
	case <-installDone:
		t.Fatalf("InstallSnapshot completed while CompactLog held snapMu — serialization is broken")
	default:
	}

	// Release the provider; CompactLog now returns. snapMu is
	// released, InstallSnapshot proceeds.
	close(providerRelease)

	if err := <-compactDone; err != nil {
		t.Fatalf("CompactLog: %v", err)
	}
	resp := <-installDone
	var out InstallSnapshotResponse
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		t.Fatalf("decode install response: %v", err)
	}
	if !out.Success {
		t.Fatalf("InstallSnapshot must succeed post-serialisation: %+v", out)
	}

	// Final invariant: the on-disk snapshot file reflects the
	// higher index (100), not the older one (3). A rollback bug
	// would leave LastIncludedIndex at 3 because CompactLog
	// finished last on disk.
	snap, err := LoadSnapshot(dir)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if snap.Meta.LastIncludedIndex != 100 {
		t.Fatalf("on-disk LastIncludedIndex = %d, want 100 (newest snapshot must survive)",
			snap.Meta.LastIncludedIndex)
	}
}

// TestConcurrentInstallSnapshotHandlersDoNotTrashSpillFile pins
// the invariant that two InstallSnapshot handlers running in
// parallel cannot interleave their pending-transfer state or
// spill files. Previously the shared pendingSnapshot* fields and
// the single recv file were not protected during the installer /
// SaveSnapshot window, letting a concurrent handler reset or
// remove another's in-flight transfer.
//
// Serialisation interacts with the monotonicity check: whichever
// handler runs first persists its snapshot; the second handler
// sees req.LastIncludedIndex <= n.snapshotIndex (for the lower
// index) and acks idempotently without re-entering the transfer
// path. Either ordering leaves the on-disk snapshot at the
// strictly higher of the two indices, never a torn mix.
func TestConcurrentInstallSnapshotHandlersDoNotTrashSpillFile(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:          "n",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })

	// Installer records which transfer it saw. Under snapMu the two
	// handlers serialise; under the monotonicity check only the
	// handler that ran first (and any later handler with idx above
	// the new snapshotIndex) actually invokes the installer. A
	// reordering bug would either call the installer with stale
	// ordering or skip both.
	var mu sync.Mutex
	var seen []uint64
	n.SetSnapshotInstaller(func(r io.Reader, _ int64, idx, _ uint64) error {
		// Hold the reader briefly so the other goroutine has a
		// wall-clock chance to collide if the lock is missing.
		_, _ = io.Copy(io.Discard, r)
		time.Sleep(25 * time.Millisecond)
		mu.Lock()
		seen = append(seen, idx)
		mu.Unlock()
		return nil
	})
	n.mu.Lock()
	n.currentTerm = 1
	n.mu.Unlock()

	send := func(idx uint64, data string) *RPCMessage {
		req := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: idx, LastIncludedTerm: 1,
			Offset: 0, Done: true, Data: []byte(data),
		}
		payload, _ := json.Marshal(req)
		return n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})
	}

	done := make(chan *RPCMessage, 2)
	go func() { done <- send(7, "seven") }()
	go func() { done <- send(9, "nine") }()

	// Both RPCs must ACK Success=true: either the install ran, or
	// the monotonicity check idempotently accepted the lower-index
	// call. Either way the leader is told "you're caught up".
	for i := 0; i < 2; i++ {
		resp := <-done
		var out InstallSnapshotResponse
		if err := json.Unmarshal(resp.Payload, &out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if !out.Success {
			t.Fatalf("concurrent install %d returned Success=false: %+v", i, out)
		}
	}

	// Final raft.snapshot MUST reflect the higher of the two
	// indices. A torn-mix failure would either pick idx=7 (rollback)
	// or yield a corrupt file.
	snap, err := LoadSnapshot(dir)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if snap.Meta.LastIncludedIndex != 9 {
		t.Fatalf("final on-disk LastIncludedIndex = %d, want 9 (strictly higher)",
			snap.Meta.LastIncludedIndex)
	}
	// The installer MUST have run at least once; it MUST NOT have
	// observed an impossible order (7 after 9, which would indicate
	// a post-rollback install slipped through).
	mu.Lock()
	defer mu.Unlock()
	if len(seen) == 0 {
		t.Fatalf("installer must run at least once across two concurrent calls")
	}
	if len(seen) >= 2 && seen[0] == 9 && seen[1] == 7 {
		t.Fatalf("installer ran 9 then 7; monotonicity check failed to skip the stale call")
	}
}
