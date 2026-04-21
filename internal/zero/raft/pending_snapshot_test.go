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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// spillPath returns the canonical spill-file path for a state dir,
// mirroring what pending_snapshot.go uses at runtime.
func spillPath(stateDir string) string {
	return filepath.Join(stateDir, pendingSnapshotFileName)
}

// sendChunk is a local helper mirroring the integration harness:
// JSON-encode an InstallSnapshotRequest and feed it through
// handleInstallSnapshot. Returns the decoded response.
func sendChunk(t *testing.T, n *Node, req InstallSnapshotRequest) InstallSnapshotResponse {
	t.Helper()
	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal chunk: %v", err)
	}
	msg := n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})
	var resp InstallSnapshotResponse
	if err := json.Unmarshal(msg.Payload, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

// TestInstallSnapshotSpillsChunksToDisk locks in the production
// memory story: when stateDir is configured, each incoming chunk
// goes straight to raft.snapshot.recv on disk and the Node's
// in-memory buffer stays empty. Without the spill a follower
// receiving a 256 MiB snapshot would hold the whole payload in
// RAM for the duration of the transfer.
func TestInstallSnapshotSpillsChunksToDisk(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:          "f",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })
	n.SetSnapshotInstaller(func([]byte, uint64, uint64) error { return nil })
	n.currentTerm = 1

	// First non-final chunk opens the spill file and writes the
	// chunk body to disk. The in-memory buffer MUST stay empty.
	resp := sendChunk(t, n, InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 10, LastIncludedTerm: 1,
		Offset: 0, Done: false, Data: []byte("first-"),
	})
	if !resp.Success {
		t.Fatalf("first chunk rejected: %+v", resp)
	}

	n.mu.RLock()
	memLen := len(n.pendingSnapshot)
	file := n.pendingSnapshotFile
	path := n.pendingSnapshotPath
	n.mu.RUnlock()

	if memLen != 0 {
		t.Fatalf("pendingSnapshot in-memory buffer must be empty in disk-backed mode, got %d bytes", memLen)
	}
	if file == nil {
		t.Fatalf("pendingSnapshotFile must be open after first chunk")
	}
	if path != spillPath(dir) {
		t.Fatalf("spill path = %q, want %q", path, spillPath(dir))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat spill file: %v", err)
	}
	if info.Size() != int64(len("first-")) {
		t.Fatalf("spill file size = %d, want %d", info.Size(), len("first-"))
	}
}

// TestInstallSnapshotSpillFileRemovedOnSuccess ensures the spill
// file is deleted once raft.snapshot on disk has become the
// authoritative copy. Leaving raft.snapshot.recv around would
// confuse the next transfer's offset=0 reset into inheriting
// stale bytes.
func TestInstallSnapshotSpillFileRemovedOnSuccess(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:          "f",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })
	n.SetSnapshotInstaller(func([]byte, uint64, uint64) error { return nil })
	n.currentTerm = 1

	// Two chunks: second Done=true. After this the spill file must
	// be gone and raft.snapshot must exist.
	_ = sendChunk(t, n, InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 10, LastIncludedTerm: 1,
		Offset: 0, Done: false, Data: []byte("head-"),
	})
	resp := sendChunk(t, n, InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 10, LastIncludedTerm: 1,
		Offset: uint64(len("head-")), Done: true, Data: []byte("tail"),
	})
	if !resp.Success {
		t.Fatalf("final chunk rejected: %+v", resp)
	}

	if _, err := os.Stat(spillPath(dir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("spill file must be removed on successful install, stat err = %v", err)
	}
	if _, err := LoadSnapshot(dir); err != nil {
		t.Fatalf("raft.snapshot must exist after success, LoadSnapshot: %v", err)
	}
}

// TestInstallSnapshotSpillFileRemovedOnInstallerError verifies the
// cleanup happens on the rejection path too. A stale spill left
// behind here would corrupt the next transfer's initial state.
func TestInstallSnapshotSpillFileRemovedOnInstallerError(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:          "f",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })
	n.SetSnapshotInstaller(func([]byte, uint64, uint64) error {
		return errors.New("installer rejected")
	})
	n.currentTerm = 1

	resp := sendChunk(t, n, InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 10, LastIncludedTerm: 1,
		Offset: 0, Done: true, Data: []byte("only-chunk"),
	})
	if resp.Success {
		t.Fatalf("expected Success=false when installer errors")
	}
	if _, err := os.Stat(spillPath(dir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("spill file must be removed on installer error, stat err = %v", err)
	}
}

// TestInstallSnapshotRejectsOverflow asserts the follower refuses
// a chunk that would push the accumulated transfer past
// maxSnapshotSize. Without this guard a malicious or buggy leader
// could drive the follower into unbounded memory / disk consumption
// before SaveSnapshot's own size check ran.
func TestInstallSnapshotRejectsOverflow(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:          "f",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })
	n.SetSnapshotInstaller(func([]byte, uint64, uint64) error { return nil })
	n.currentTerm = 1

	// Start a transfer just short of the cap, then try to append a
	// chunk that crosses it. The second chunk must be rejected with
	// Success=false and the spill state must be cleared.
	firstSize := maxSnapshotSize - 8
	_ = sendChunk(t, n, InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 1, LastIncludedTerm: 1,
		Offset: 0, Done: false, Data: make([]byte, firstSize),
	})
	resp := sendChunk(t, n, InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 1, LastIncludedTerm: 1,
		Offset: uint64(firstSize), Done: false, Data: make([]byte, 16),
	})
	if resp.Success {
		t.Fatalf("expected Success=false on overflow append")
	}
	n.mu.RLock()
	expected := n.pendingSnapshotExpected
	n.mu.RUnlock()
	if expected != 0 {
		t.Fatalf("pendingSnapshotExpected must reset after overflow, got %d", expected)
	}
	if _, err := os.Stat(spillPath(dir)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("spill file must be removed after overflow rejection, stat err = %v", err)
	}
}

// TestLoadFromDiskCleansStalePendingSnapshot locks in the crash-
// recovery contract: a stale raft.snapshot.recv from a previous
// aborted transfer must be removed on the next Start so incoming
// chunks start from a known-empty spill.
func TestLoadFromDiskCleansStalePendingSnapshot(t *testing.T) {
	dir := t.TempDir()
	stale := spillPath(dir)
	if err := os.WriteFile(stale, []byte("leftover-from-crash"), 0o600); err != nil {
		t.Fatalf("seed stale spill: %v", err)
	}

	n := NewNode(Config{
		NodeID:          "f",
		SharedSecret:    "test-hmac-key-32-bytes-minimum-padding!",
		ElectionTimeout: 300 * time.Millisecond,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })

	if err := n.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk: %v", err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadFromDisk must remove stale pending snapshot, stat err = %v", err)
	}
}
