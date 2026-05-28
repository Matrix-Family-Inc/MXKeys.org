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

// testWALKey is a fixed HMAC key used across WAL tests. Any non-empty
// value would do; using a fixed secret keeps golden-like reasoning
// simple when debugging a failing test.
var testWALKey = []byte("test-wal-hmac-key-32-bytes-or-so!")

func newTestWAL(t *testing.T) (*WAL, string) {
	t.Helper()
	dir := t.TempDir()
	w, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
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

	w1, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if err != nil {
		t.Fatalf("OpenWAL: %v", err)
	}
	mustAppend(t, w1, LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"a"`)})
	mustAppend(t, w1, LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"b"`)})
	if err := w1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
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

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
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
	// After the walMagicSize-byte magic, each record is walHeaderSize
	// bytes of header + payload. Skip magic + first record header to
	// read first length, then advance past the first record and the
	// second record's header to land inside the second record's
	// payload.
	firstLen := binary.LittleEndian.Uint32(raw[walMagicSize : walMagicSize+4])
	tamperOffset := int(walMagicSize + walHeaderSize + firstLen + walHeaderSize)
	raw[tamperOffset] ^= 0xFF
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
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

// TestWALDetectsHMACTampering verifies the security property of the v3
// format: an attacker who rewrites a record's payload (preserving the
// CRC by also rewriting it) is caught by the HMAC layer. The caller
// sees ErrWALTampered, a stronger signal than generic corruption.
func TestWALDetectsHMACTampering(t *testing.T) {
	w, dir := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1, Command: json.RawMessage(`"good"`)},
		LogEntry{Index: 2, Term: 1, Command: json.RawMessage(`"target"`)},
	)
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	path := filepath.Join(dir, walFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// Plan: rewrite the second record's payload to a different
	// syntactically-valid JSON entry, recompute the CRC accordingly,
	// but leave the HMAC alone. The reader must detect the mismatch.
	firstLen := binary.LittleEndian.Uint32(raw[walMagicSize : walMagicSize+4])
	secondHdr := walMagicSize + walHeaderSize + int(firstLen)
	secondLen := binary.LittleEndian.Uint32(raw[secondHdr : secondHdr+4])
	secondPayloadStart := secondHdr + walHeaderSize
	secondPayloadEnd := secondPayloadStart + int(secondLen)

	// Replace the payload with a synthetic LogEntry of the same length.
	replacement := make([]byte, secondLen)
	copy(replacement, []byte(`{"Index":99,"Term":99,"Command":"x"}`))
	// Pad if replacement is shorter than original by repeating its bytes.
	for i := len(`{"Index":99,"Term":99,"Command":"x"}`); i < int(secondLen); i++ {
		replacement[i] = ' '
	}
	copy(raw[secondPayloadStart:secondPayloadEnd], replacement)

	// Rewrite the CRC to match the forged payload so the fast-path
	// CRC check passes; only the HMAC can catch this.
	forgedCRC := crc32.Checksum(raw[secondPayloadStart:secondPayloadEnd], walCRC)
	binary.LittleEndian.PutUint32(raw[secondHdr+4:secondHdr+8], forgedCRC)

	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	w2, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer w2.Close()

	got, rerr := w2.ReadAll()
	if !errors.Is(rerr, ErrWALTampered) {
		t.Fatalf("expected ErrWALTampered, got %v", rerr)
	}
	if len(got) != 1 || got[0].Index != 1 {
		t.Fatalf("expected only the first well-formed entry, got %+v", got)
	}
}

// TestWALRejectsLegacyV2 proves the upgrade gate: a file that starts
// with the v2 magic is refused with the specific sentinel error, which
// drives the operator to the documented upgrade path.
func TestWALRejectsLegacyV2(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, walFileName)
	if err := os.WriteFile(path, walMagicV2[:], 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey})
	if !errors.Is(err, ErrWALLegacyFormat) {
		t.Fatalf("expected ErrWALLegacyFormat, got %v", err)
	}
}

// TestWALRejectsMissingHMACKey verifies the constructor's key-required
// precondition.
func TestWALRejectsMissingHMACKey(t *testing.T) {
	_, err := OpenWAL(WALOptions{Dir: t.TempDir(), SyncOnAppend: true})
	if err == nil {
		t.Fatal("expected OpenWAL to reject missing HMAC key")
	}
}

// TestWALRejectsUnknownMagic verifies that opening a WAL file with an
// incompatible magic prefix is a hard error rather than a silent parse of
// potentially foreign data.
func TestWALRejectsUnknownMagic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, walFileName)
	// Write a fake magic that matches the version-v1 shape (never shipped
	// but a reader that defensive-checks the magic should reject it).
	bogus := []byte{'M', 'X', 'K', 'S', '_', 'W', 'A', 'L', '_', 'v', '1', 0}
	if err := os.WriteFile(path, bogus, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := OpenWAL(WALOptions{Dir: dir, SyncOnAppend: true, HMACKey: testWALKey}); err == nil {
		t.Fatal("expected OpenWAL to reject unknown magic")
	}
}

// Group-commit, durability, truncation, close-semantics, size-limit,
// and CRC-init tests live in wal_durability_test.go and
// wal_truncate_test.go to keep this file under the ADR-0010 size cap.
