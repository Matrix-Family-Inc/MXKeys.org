package raft

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

func newTestWAL(t *testing.T) (*WAL, string) {
	t.Helper()
	dir := t.TempDir()
	w, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w, dir
}

func mustAppend(t *testing.T, w *WAL, entries ...LogEntry) {
	t.Helper()
	for _, e := range entries {
		if err := w.Append(e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
}

func TestWALRoundTrip(t *testing.T) {
	w, _ := newTestWAL(t)

	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"hello"`)},
		LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"world"`)},
		LogEntry{Index: 3, Term: 2, Command: json.RawMessage(`{"k":"v"}`)},
	)

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].Index != 1 || got[2].Index != 3 {
		t.Fatalf("unexpected indices: %v", got)
	}
}

func TestWALRoundTripAcrossReopen(t *testing.T) {
	dir := t.TempDir()

	w1, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	mustAppend(t, w1, LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"a"`)})
	mustAppend(t, w1, LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"b"`)})
	if err := w1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()

	got, err := w2.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries after reopen, got %d", len(got))
	}

	// New appends land after the reopened prefix.
	mustAppend(t, w2, LogEntry{Index: 3, Term: 2, Command: json.RawMessage(`"c"`)})

	got, err = w2.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries after second append, got %d", len(got))
	}
}

func TestWALDetectsCorruptTail(t *testing.T) {
	w, dir := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"a"`)},
		LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"b"`)},
	)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Append a torn header (only 5 bytes instead of 8) to simulate a crash.
	path := filepath.Join(dir, walFileName)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := f.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x05}); err != nil {
		t.Fatalf("write torn tail: %v", err)
	}
	_ = f.Close()

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()

	got, rerr := w2.ReadAll()
	if !errors.Is(rerr, ErrWALCorrupt) {
		t.Fatalf("expected ErrWALCorrupt, got %v", rerr)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 well-formed entries before the torn tail, got %d", len(got))
	}
}

func TestWALDetectsCRCTampering(t *testing.T) {
	w, dir := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"good"`)},
		LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"tampered"`)},
	)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Flip a payload byte in the second record. CRC validation should reject
	// it and ReadAll should return only the first entry.
	path := filepath.Join(dir, walFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// First record: 8 bytes header + len bytes. Advance past it to find the
	// payload of record 2.
	firstLen := binary.LittleEndian.Uint32(raw[0:4])
	tamperOffset := int(8 + firstLen + 8) // skip first record + second header
	raw[tamperOffset] ^= 0xFF
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()

	got, rerr := w2.ReadAll()
	if !errors.Is(rerr, ErrWALCorrupt) {
		t.Fatalf("expected ErrWALCorrupt, got %v", rerr)
	}
	if len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("expected only the first well-formed entry, got %+v", got)
	}
}

func TestWALRejectsOversizedRecord(t *testing.T) {
	w, _ := newTestWAL(t)
	huge := make([]byte, walMaxRecord+1)
	for i := range huge {
		huge[i] = 'A'
	}
	err := w.Append(LogEntry{Index: 1, Term: 1, Command: json.RawMessage(huge)})
	if err == nil {
		t.Fatal("expected oversized record to be rejected")
	}
}

func TestWALTruncateAfter(t *testing.T) {
	w, _ := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1},
		LogEntry{Index: 2, Term: 1},
		LogEntry{Index: 3, Term: 2},
		LogEntry{Index: 4, Term: 2},
		LogEntry{Index: 5, Term: 3},
	)

	if err := w.TruncateAfter(3); err != nil {
		t.Fatalf("TruncateAfter: %v", err)
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries after truncate, got %d", len(got))
	}
	if got[2].Index != 3 {
		t.Fatalf("expected last kept index 3, got %d", got[2].Index)
	}

	// Confirm that Append still lands at the end after truncation.
	mustAppend(t, w, LogEntry{Index: 4, Term: 4})
	got, err = w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 4 || got[3].Term != 4 {
		t.Fatalf("unexpected post-truncate append: %+v", got)
	}
}

func TestWALTruncateBefore(t *testing.T) {
	w, _ := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1},
		LogEntry{Index: 2, Term: 1},
		LogEntry{Index: 3, Term: 2},
		LogEntry{Index: 4, Term: 2},
	)

	if err := w.TruncateBefore(3); err != nil {
		t.Fatalf("TruncateBefore: %v", err)
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries after truncate-before, got %d", len(got))
	}
	if got[0].Index != 3 || got[1].Index != 4 {
		t.Fatalf("unexpected entries after truncate-before: %+v", got)
	}
}

// sanity test that crc32 IEEE table resolves at package init. Protects against
// future code-motion that might drop the var.
func TestWALCRCTableInitialized(t *testing.T) {
	if walCRC == nil {
		t.Fatal("walCRC must be initialized at package load")
	}
	sum := crc32.Checksum([]byte("abc"), walCRC)
	if sum == 0 {
		t.Fatal("crc32 IEEE of 'abc' must be non-zero")
	}
}
