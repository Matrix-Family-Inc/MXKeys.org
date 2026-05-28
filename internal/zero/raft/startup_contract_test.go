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
	"io"
	"os"
	"path/filepath"
	"testing"
)

const startupContractSecret = "startup-contract-hmac-32-bytes-minimum-padding"

// seedWALAndSnapshot writes a full, self-consistent on-disk state:
// a snapshot file at (idx, term) plus a WAL holding entries above
// the snapshot boundary. Returns the state dir.
func seedWALAndSnapshot(t *testing.T, snapIdx, snapTerm uint64, walEntries []LogEntry) string {
	t.Helper()
	dir := t.TempDir()

	err := SaveSnapshot(dir, Snapshot{
		Meta: SnapshotMeta{LastIncludedIndex: snapIdx, LastIncludedTerm: snapTerm},
		Data: []byte(`{"version":1}`),
	})
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	w, err := OpenWAL(WALOptions{
		Dir:          dir,
		SyncOnAppend: true,
		HMACKey:      []byte(startupContractSecret),
	})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	for _, e := range walEntries {
		if err := w.Append(e); err != nil {
			t.Fatalf("WAL append: %v", err)
		}
	}
	return dir
}

// TestLoadFromDiskRefusesSnapshotWithoutInstaller pins the
// configuration-contract invariant added in this commit: a Node
// with a state directory that holds a snapshot MUST have a
// SnapshotInstaller registered or LoadFromDisk fails clean
// BEFORE touching any Node field. Advancing snapshotIndex past
// a payload that was never applied to a state machine would
// silently lie about node progress.
func TestLoadFromDiskRefusesSnapshotWithoutInstaller(t *testing.T) {
	dir := seedWALAndSnapshot(t, 5, 2, nil)

	n := NewNode(Config{NodeID: "n", SharedSecret: startupContractSecret})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })

	err := n.LoadFromDisk()
	if !errors.Is(err, ErrSnapshotInstallerRequired) {
		t.Fatalf("expected ErrSnapshotInstallerRequired, got %v", err)
	}

	// Assert the Node remained fully unmodified: no field touched
	// by the aborted snapshot install leaked into memory.
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.snapshotIndex != 0 || n.snapshotTerm != 0 || n.logOffset != 0 {
		t.Fatalf("snapshotIndex/snapshotTerm/logOffset must stay zero on rejection, got (%d, %d, %d)",
			n.snapshotIndex, n.snapshotTerm, n.logOffset)
	}
	if n.commitIndex != 0 || n.lastApplied != 0 {
		t.Fatalf("commitIndex/lastApplied must stay zero on rejection, got (%d, %d)",
			n.commitIndex, n.lastApplied)
	}
}

// TestLoadFromDiskDoesNotPartiallyMutateWhenInstallerFails proves
// the phased-apply invariant: if the installer returns an error,
// the Node's snapshotIndex/term/logOffset MUST remain zero. A
// partial mutation here would leave the caller in the "Start()
// failed but Node fields are set" trap the audit flagged.
func TestLoadFromDiskDoesNotPartiallyMutateWhenInstallerFails(t *testing.T) {
	dir := seedWALAndSnapshot(t, 7, 3, nil)

	n := NewNode(Config{NodeID: "n", SharedSecret: startupContractSecret})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })

	installerErr := errors.New("installer refused")
	n.SetSnapshotInstaller(func(r io.Reader, _ int64, _, _ uint64) error {
		_, _ = io.Copy(io.Discard, r)
		return installerErr
	})

	err := n.LoadFromDisk()
	if !errors.Is(err, installerErr) {
		t.Fatalf("expected installer error to bubble up, got %v", err)
	}

	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.snapshotIndex != 0 || n.logOffset != 0 || n.commitIndex != 0 {
		t.Fatalf("Node must not have been mutated when installer fails: idx=%d offset=%d commit=%d",
			n.snapshotIndex, n.logOffset, n.commitIndex)
	}
}

// TestLoadFromDiskDoesNotMutateWhenWALReadFatallyFails covers the
// WAL-first ordering: even when the snapshot is perfectly valid,
// a non-ErrWALCorrupt WAL read error MUST abort before the
// installer runs, so neither the state machine nor the Node
// fields are touched. Reproduces the WAL-read-permission failure
// by chmod-ing the WAL file to unreadable.
func TestLoadFromDiskDoesNotMutateWhenWALReadFatallyFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod-based permission trick is ineffective")
	}

	dir := seedWALAndSnapshot(t, 5, 1, []LogEntry{
		{Index: 6, Term: 1, Command: json.RawMessage(`"first"`)},
	})

	// Build the Node (which opens the WAL). Then revoke read on
	// the WAL file so ReadAll hits EACCES rather than ErrWALCorrupt.
	n := NewNode(Config{NodeID: "n", SharedSecret: startupContractSecret})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	// Close the node's handle so the chmod takes effect at read time.
	_ = n.wal.Close()

	walPath := filepath.Join(dir, "raft.wal")
	if err := os.Chmod(walPath, 0o000); err != nil {
		t.Fatalf("chmod WAL: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(walPath, 0o600) })

	// Re-open a Node and re-attach the state dir to get a fresh WAL handle
	// using the same directory.
	n2 := NewNode(Config{NodeID: "n2", SharedSecret: startupContractSecret})
	if err := n2.SetStateDir(dir, true); err != nil {
		// SetStateDir may fail because OpenWAL can't read the file;
		// that's equivalent to the same class of failure we want to
		// exercise. In that case there's nothing to test; skip.
		t.Skipf("SetStateDir with unreadable WAL already fails here: %v", err)
	}
	t.Cleanup(func() { _ = n2.wal.Close() })

	installerCalled := false
	n2.SetSnapshotInstaller(func(r io.Reader, _ int64, _, _ uint64) error {
		installerCalled = true
		_, _ = io.Copy(io.Discard, r)
		return nil
	})

	err := n2.LoadFromDisk()
	if err == nil {
		t.Fatalf("LoadFromDisk must fail when WAL cannot be read")
	}
	if installerCalled {
		t.Fatalf("installer must NOT run when WAL read fails; got called")
	}

	n2.mu.RLock()
	defer n2.mu.RUnlock()
	if n2.snapshotIndex != 0 || n2.logOffset != 0 || n2.commitIndex != 0 {
		t.Fatalf("Node must remain unmutated on WAL read failure: idx=%d offset=%d commit=%d",
			n2.snapshotIndex, n2.logOffset, n2.commitIndex)
	}
}

// TestHandleInstallSnapshotRejectsWithoutInstaller mirrors the
// startup-path rule on the RPC side. A Done chunk must not
// advance snapshotIndex (or write anything to raft.snapshot)
// when no installer is registered; instead the handler returns
// Success=false so the leader retries.
func TestHandleInstallSnapshotRejectsWithoutInstaller(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{
		NodeID:       "n",
		SharedSecret: startupContractSecret,
	})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	t.Cleanup(func() { _ = n.wal.Close() })
	n.currentTerm = 1

	req := InstallSnapshotRequest{
		Term: 1, LeaderID: "L", LastIncludedIndex: 10, LastIncludedTerm: 1,
		Offset: 0, Done: true, Data: []byte("would-be-state"),
	}
	payload, _ := json.Marshal(req)
	resp := n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})
	var out InstallSnapshotResponse
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Success {
		t.Fatalf("expected Success=false when no installer registered, got %+v", out)
	}

	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.snapshotIndex != 0 {
		t.Fatalf("snapshotIndex must not advance on rejected install, got %d", n.snapshotIndex)
	}
	// The spill file must not exist either: the handler must reject
	// before touching any on-disk state.
	if _, err := os.Stat(filepath.Join(dir, pendingSnapshotFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("spill file must not have been created, stat err = %v", err)
	}
}
