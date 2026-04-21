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
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// seedValidSnapshot writes a well-formed raft.snapshot into dir
// and returns its on-disk path. Used by every integrity test
// below as the "good" baseline before corruption is introduced.
func seedValidSnapshot(t *testing.T, dir string, data []byte) string {
	t.Helper()
	err := SaveSnapshot(dir, Snapshot{
		Meta: SnapshotMeta{LastIncludedIndex: 10, LastIncludedTerm: 1},
		Data: data,
	})
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}
	return filepath.Join(dir, snapshotFileName)
}

// flipByte XORs 0xFF into the file byte at the given offset to
// simulate a single-bit-rot event. Bounded, deterministic, and
// visible both to CRC and to any structural parser.
func flipByte(t *testing.T, path string, offset int) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if offset >= len(raw) {
		t.Fatalf("offset %d out of range for file size %d", offset, len(raw))
	}
	raw[offset] ^= 0xFF
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

// TestLoadSnapshotReaderRejectsPayloadCorruption pins the
// integrity invariant that the streaming install path must catch
// on-disk corruption before feeding the state machine. Previously
// LoadSnapshotReader only verified magic + header and trusted the
// payload; a flipped byte inside the data portion would slip
// through to the installer.
func TestLoadSnapshotReaderRejectsPayloadCorruption(t *testing.T) {
	dir := t.TempDir()
	path := seedValidSnapshot(t, dir, []byte("hello-raft-snapshot-payload"))

	// Sanity: the unmodified file loads cleanly.
	f, _, err := LoadSnapshotReader(dir)
	if err != nil {
		t.Fatalf("LoadSnapshotReader on clean file: %v", err)
	}
	_ = f.Close()

	// Flip a byte inside the data portion (offset >= header size).
	flipByte(t, path, snapshotHeaderSize+3)

	// The corrupted file must be rejected with ErrSnapshotCorrupt.
	_, _, err = LoadSnapshotReader(dir)
	if !errors.Is(err, ErrSnapshotCorrupt) {
		t.Fatalf("expected ErrSnapshotCorrupt on payload corruption, got %v", err)
	}
}

// TestLoadSnapshotReaderRejectsTruncatedPayload covers the other
// common failure mode: the on-disk length in the header says N
// bytes but the file is shorter. verifySnapshotPayloadCRC's
// io.CopyN(size) must error out instead of silently succeeding
// with a partial payload.
func TestLoadSnapshotReaderRejectsTruncatedPayload(t *testing.T) {
	dir := t.TempDir()
	path := seedValidSnapshot(t, dir, []byte("this-payload-will-be-truncated"))

	// Truncate the file by four bytes so the header still claims
	// the full payload length but the file ends early.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if err := os.Truncate(path, info.Size()-4); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	_, _, err = LoadSnapshotReader(dir)
	if !errors.Is(err, ErrSnapshotCorrupt) {
		t.Fatalf("expected ErrSnapshotCorrupt on truncation, got %v", err)
	}
}

// TestLoadSnapshotReaderRejectsTamperedCRCField locks in the
// symmetric case: the header's CRC field is flipped while the
// payload is intact. The pre-verify must compute the real CRC
// from the data and see it no longer matches the header.
func TestLoadSnapshotReaderRejectsTamperedCRCField(t *testing.T) {
	dir := t.TempDir()
	path := seedValidSnapshot(t, dir, []byte("intact-data-but-stale-crc-field"))

	// snapshotHeaderCRCOffset is the first byte of the data_crc
	// field. Flipping any of its four bytes produces a CRC that
	// cannot match the intact payload's real checksum.
	flipByte(t, path, snapshotHeaderCRCOffset)

	_, _, err := LoadSnapshotReader(dir)
	if !errors.Is(err, ErrSnapshotCorrupt) {
		t.Fatalf("expected ErrSnapshotCorrupt on tampered CRC field, got %v", err)
	}
}

// TestLoadSnapshotReaderAcceptsValidFile is the positive-case
// guard: the pre-verify path must still let a clean file through
// with the reader seeked to the data portion and meta populated.
func TestLoadSnapshotReaderAcceptsValidFile(t *testing.T) {
	dir := t.TempDir()
	data := []byte("valid-payload-for-integrity-test")
	_ = seedValidSnapshot(t, dir, data)

	f, meta, err := LoadSnapshotReader(dir)
	if err != nil {
		t.Fatalf("LoadSnapshotReader on valid file: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	if meta.LastIncludedIndex != 10 || meta.LastIncludedTerm != 1 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	if meta.Size != int64(len(data)) {
		t.Fatalf("meta.Size = %d, want %d", meta.Size, len(data))
	}

	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read data: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("data mismatch: got %q want %q", got, data)
	}
}

// TestCountingReaderCountsExactBytes documents the observability
// contract used by LoadFromDisk and handleInstallSnapshot to log
// an advisory warning when an installer stops short of size.
// A counter that over- or under-reports by even a byte would
// produce noisy false warnings in production logs.
func TestCountingReaderCountsExactBytes(t *testing.T) {
	payload := []byte("0123456789")
	cr := &countingReader{r: &sliceReader{b: payload}}

	// Pull the stream out in three chunks of irregular size.
	for _, want := range []int{3, 4, 3} {
		buf := make([]byte, want)
		if _, err := io.ReadFull(cr, buf); err != nil {
			t.Fatalf("ReadFull: %v", err)
		}
	}
	if cr.count != int64(len(payload)) {
		t.Fatalf("counter = %d, want %d", cr.count, len(payload))
	}
}

// sliceReader turns a []byte into an io.Reader without depending
// on the bytes package. Kept local to this file.
type sliceReader struct{ b []byte }

func (s *sliceReader) Read(p []byte) (int, error) {
	if len(s.b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.b)
	s.b = s.b[n:]
	return n, nil
}
