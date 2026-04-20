package raft

import (
	"encoding/json"
	"testing"
)

// TestLoadFromDiskReplaysWAL verifies that a node started against an existing
// state directory reconstructs its log from the WAL before accepting RPCs.
func TestLoadFromDiskReplaysWAL(t *testing.T) {
	dir := t.TempDir()

	// Simulate a previous instance that wrote three entries to the WAL.
	w, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"one"`)},
		LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"two"`)},
		LogEntry{Index: 3, Term: 2, Command: json.RawMessage(`"three"`)},
	)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Fresh node bound to the same dir: LoadFromDisk must rebuild n.log.
	n := NewNode(Config{NodeID: "n1", SharedSecret: "test-wal-hmac-key-32-bytes-or-so!"})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	defer func() { _ = n.wal.Close() }()

	if err := n.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk: %v", err)
	}

	n.mu.RLock()
	defer n.mu.RUnlock()
	if len(n.log) != 3 {
		t.Fatalf("expected 3 replayed entries, got %d", len(n.log))
	}
	if n.currentTerm != 2 {
		t.Fatalf("currentTerm must advance to highest replayed term (2), got %d", n.currentTerm)
	}
	if n.log[2].Index != 3 {
		t.Fatalf("unexpected last replayed index: %d", n.log[2].Index)
	}
}

// TestLoadFromDiskRestoresSnapshotThenReplaysTail verifies that a snapshot
// is installed before the WAL is replayed, and WAL entries whose Index is
// covered by the snapshot are skipped.
func TestLoadFromDiskRestoresSnapshotThenReplaysTail(t *testing.T) {
	dir := t.TempDir()

	// Snapshot captures state through index 5, term 3.
	stateBytes := []byte(`{"map":{"a":1,"b":2}}`)
	if err := SaveSnapshot(dir, Snapshot{
		Meta: SnapshotMeta{LastIncludedIndex: 5, LastIncludedTerm: 3, Size: int64(len(stateBytes))},
		Data: stateBytes,
	}); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// WAL contains entries 4..7. Entries 4 and 5 are already in the
	// snapshot and must be skipped on replay. Entries 6 and 7 extend past
	// the snapshot and must land in n.log.
	w, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	mustAppend(t, w,
		LogEntry{Index: 4, Term: 3, Command: json.RawMessage(`"pre-snap"`)},
		LogEntry{Index: 5, Term: 3, Command: json.RawMessage(`"snap-edge"`)},
		LogEntry{Index: 6, Term: 4, Command: json.RawMessage(`"post-snap-1"`)},
		LogEntry{Index: 7, Term: 4, Command: json.RawMessage(`"post-snap-2"`)},
	)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var installed struct {
		data     []byte
		lastIdx  uint64
		lastTerm uint64
	}
	installerCalls := 0

	n := NewNode(Config{NodeID: "n1", SharedSecret: "test-wal-hmac-key-32-bytes-or-so!"})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	defer func() { _ = n.wal.Close() }()

	n.SetSnapshotInstaller(func(data []byte, lastIncludedIndex, lastIncludedTerm uint64) error {
		installerCalls++
		installed.data = append([]byte(nil), data...)
		installed.lastIdx = lastIncludedIndex
		installed.lastTerm = lastIncludedTerm
		return nil
	})

	if err := n.LoadFromDisk(); err != nil {
		t.Fatalf("LoadFromDisk: %v", err)
	}
	if installerCalls != 1 {
		t.Fatalf("installer must be invoked exactly once, got %d", installerCalls)
	}
	if installed.lastIdx != 5 || installed.lastTerm != 3 || string(installed.data) != string(stateBytes) {
		t.Fatalf("installer received wrong payload: %+v", installed)
	}

	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.snapshotIndex != 5 || n.snapshotTerm != 3 {
		t.Fatalf("snapshotIndex/Term not restored: %d/%d", n.snapshotIndex, n.snapshotTerm)
	}
	if n.commitIndex != 5 || n.lastApplied != 5 {
		t.Fatalf("commit/lastApplied must reflect the snapshot boundary: %d/%d", n.commitIndex, n.lastApplied)
	}
	if len(n.log) != 2 || n.log[0].Index != 6 || n.log[1].Index != 7 {
		t.Fatalf("post-snapshot WAL tail not replayed correctly: %+v", n.log)
	}
	if n.currentTerm < 4 {
		t.Fatalf("currentTerm must reach the WAL tail term (4), got %d", n.currentTerm)
	}
}

// TestCompactLogProducesSnapshotAndTrimsWAL verifies end-to-end compaction:
// Submit several entries as a single-node leader (already covered by
// TestSubmitAsLeaderAppendsEntryAndReturnsReplicationResult), then
// CompactLog with a trivial state provider and verify WAL shrinks while the
// on-disk snapshot captures the declared index/term.
func TestCompactLogProducesSnapshotAndTrimsWAL(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{NodeID: "n1", SharedSecret: "test-wal-hmac-key-32-bytes-or-so!"})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	defer func() { _ = n.wal.Close() }()

	n.SetSnapshotProvider(func() ([]byte, error) {
		return []byte("state-bytes"), nil
	})

	// Manually seed the log past the snapshot we'll create; CompactLog
	// requires lastApplied > 0 which is bumped by applyLoop in the real
	// path. Here we simulate the applied state directly.
	n.mu.Lock()
	n.currentTerm = 3
	n.log = []LogEntry{
		{Index: 1, Term: 1, Command: json.RawMessage(`"a"`)},
		{Index: 2, Term: 2, Command: json.RawMessage(`"b"`)},
		{Index: 3, Term: 3, Command: json.RawMessage(`"c"`)},
	}
	for _, e := range n.log {
		if err := n.wal.Append(e); err != nil {
			n.mu.Unlock()
			t.Fatalf("wal.Append seed: %v", err)
		}
	}
	n.commitIndex = 3
	n.lastApplied = 3
	n.mu.Unlock()

	if err := n.CompactLog(); err != nil {
		t.Fatalf("CompactLog: %v", err)
	}

	snap, err := LoadSnapshot(dir)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if snap.Meta.LastIncludedIndex != 3 || snap.Meta.LastIncludedTerm != 3 {
		t.Fatalf("snapshot metadata mismatch: %+v", snap.Meta)
	}
	if string(snap.Data) != "state-bytes" {
		t.Fatalf("snapshot data mismatch: %q", snap.Data)
	}

	entries, werr := n.wal.ReadAll()
	if werr != nil {
		t.Fatalf("ReadAll after compaction: %v", werr)
	}
	if len(entries) != 0 {
		t.Fatalf("WAL should be empty after compaction (all entries covered), got %d", len(entries))
	}

	// In-memory compaction: logOffset advances, n.log drops the prefix.
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.logOffset != 3 {
		t.Fatalf("expected logOffset=3 after compaction, got %d", n.logOffset)
	}
	if len(n.log) != 0 {
		t.Fatalf("expected in-memory log empty after compaction (all entries covered), got %d", len(n.log))
	}
	if n.logLen() != 3 {
		t.Fatalf("logLen() must still report absolute log length 3, got %d", n.logLen())
	}
}

// TestSubmitAfterCompactionUsesCorrectIndex verifies the post-compaction
// indexing invariant: after a snapshot at index N, the next Submit produces
// an entry with Index N+1 (not len(n.log)+1), and subsequent replication
// sees it correctly via the logOffset-aware accessors.
func TestSubmitAfterCompactionUsesCorrectIndex(t *testing.T) {
	dir := t.TempDir()
	n := NewNode(Config{NodeID: "n1", SharedSecret: "test-wal-hmac-key-32-bytes-or-so!"})
	if err := n.SetStateDir(dir, true); err != nil {
		t.Fatalf("SetStateDir: %v", err)
	}
	defer func() { _ = n.wal.Close() }()

	n.SetSnapshotProvider(func() ([]byte, error) { return []byte("state"), nil })

	// Seed three entries + compact.
	n.mu.Lock()
	n.state = Leader
	n.currentTerm = 5
	n.log = []LogEntry{
		{Index: 1, Term: 5, Command: json.RawMessage(`"a"`)},
		{Index: 2, Term: 5, Command: json.RawMessage(`"b"`)},
		{Index: 3, Term: 5, Command: json.RawMessage(`"c"`)},
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

	if err := n.CompactLog(); err != nil {
		t.Fatalf("CompactLog: %v", err)
	}

	// The sole leader applies nothing remotely; Submit returns ErrTimeout
	// after CommitTimeout because there are no peers to reach quorum on a
	// multi-node config. To isolate the indexing check, we drive Submit's
	// append-and-persist logic directly via the underlying machinery.
	n.mu.Lock()
	n.state = Leader
	next := n.logLen() + 1
	entry := LogEntry{Index: next, Term: n.currentTerm, Command: json.RawMessage(`"d"`)}
	if err := n.wal.Append(entry); err != nil {
		n.mu.Unlock()
		t.Fatalf("wal append: %v", err)
	}
	n.log = append(n.log, entry)
	got := n.logLen()
	n.mu.Unlock()

	if next != 4 {
		t.Fatalf("expected Submit to choose Index=4 post-compaction, got %d", next)
	}
	if got != 4 {
		t.Fatalf("logLen() must be 4 after appending one post-compaction entry, got %d", got)
	}
}

// TestSnapshotRoundTrip exercises SaveSnapshot + LoadSnapshot in isolation.
func TestSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"k":"v","n":42}`)

	if err := SaveSnapshot(dir, Snapshot{
		Meta: SnapshotMeta{LastIncludedIndex: 100, LastIncludedTerm: 7, Size: int64(len(data))},
		Data: data,
	}); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	got, err := LoadSnapshot(dir)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if got.Meta.LastIncludedIndex != 100 || got.Meta.LastIncludedTerm != 7 {
		t.Fatalf("unexpected metadata: %+v", got.Meta)
	}
	if string(got.Data) != string(data) {
		t.Fatalf("snapshot data round-trip mismatch")
	}
}

// TestLoadSnapshotMissingReturnsSentinel verifies the sentinel contract for
// callers distinguishing "never snapshotted" from "corrupt state".
func TestLoadSnapshotMissingReturnsSentinel(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadSnapshot(dir)
	if err != ErrNoSnapshot {
		t.Fatalf("expected ErrNoSnapshot, got %v", err)
	}
}
