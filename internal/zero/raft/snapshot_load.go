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
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// LoadSnapshot reads dir/raft.snapshot. Returns ErrNoSnapshot when the file
// is missing, ErrSnapshotCorrupt when its CRC or magic fails.
//
// LoadSnapshot buffers the full data portion in memory. Callers on
// the install path (LoadFromDisk) should prefer LoadSnapshotReader
// so peak memory stays O(chunk) rather than O(snapshot size).
func LoadSnapshot(dir string) (*Snapshot, error) {
	path := snapshotFilePath(dir)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoSnapshot
		}
		return nil, fmt.Errorf("raft snapshot: open: %w", err)
	}
	defer f.Close()

	meta, expectedCRC, err := readSnapshotHeader(f)
	if err != nil {
		return nil, err
	}

	data := make([]byte, meta.Size)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, ErrSnapshotCorrupt
	}

	if crc32.Checksum(data, walCRC) != expectedCRC {
		return nil, ErrSnapshotCorrupt
	}

	return &Snapshot{
		Meta: SnapshotMeta{
			LastIncludedIndex: meta.LastIncludedIndex,
			LastIncludedTerm:  meta.LastIncludedTerm,
			Size:              meta.Size,
		},
		Data: data,
	}, nil
}

// LoadSnapshotReader opens dir/raft.snapshot, verifies magic +
// header + payload CRC, and returns a reader positioned at the
// start of the data portion along with the snapshot metadata.
// Caller MUST close the returned file when done.
//
// Integrity guarantee: the full payload is streamed through a
// Castagnoli hasher before the reader is handed to the caller.
// An on-disk corruption (bit rot, partial write, truncation, or
// header/data mismatch) is rejected here with ErrSnapshotCorrupt
// so a startup or mid-stream install never feeds a silently
// damaged payload to the state machine. The pre-verify buffer
// size stays small (streamBufSize) so peak RAM remains O(chunk)
// even for a 256 MiB snapshot.
//
// Cost: one full sequential read of the file before install,
// then a second sequential read by the installer. Both passes
// are cheap in practice because the OS page cache keeps the
// bytes hot.
func LoadSnapshotReader(dir string) (*os.File, SnapshotMeta, error) {
	path := snapshotFilePath(dir)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, SnapshotMeta{}, ErrNoSnapshot
		}
		return nil, SnapshotMeta{}, fmt.Errorf("raft snapshot: open: %w", err)
	}
	meta, expectedCRC, err := readSnapshotHeader(f)
	if err != nil {
		_ = f.Close()
		return nil, SnapshotMeta{}, err
	}
	if err := verifySnapshotPayloadCRC(f, meta.Size, expectedCRC); err != nil {
		_ = f.Close()
		return nil, SnapshotMeta{}, err
	}
	// Rewind to the start of the data portion so the installer
	// sees the complete payload.
	if _, err := f.Seek(int64(snapshotHeaderSize), io.SeekStart); err != nil {
		_ = f.Close()
		return nil, SnapshotMeta{}, fmt.Errorf("raft snapshot: seek after verify: %w", err)
	}
	return f, meta, nil
}

// streamBufSize is the chunk used while streaming a snapshot file
// through a CRC hasher. 64 KiB matches typical OS page-cache read
// granularity and keeps the hasher's working set in L1.
const streamBufSize = 64 * 1024

// verifySnapshotPayloadCRC reads exactly size bytes from r through
// a Castagnoli hasher and compares the result against expected.
// Returns ErrSnapshotCorrupt for any mismatch, short read, or
// read error. The reader is consumed; callers that want to pass
// the same stream to a subsequent consumer must seek or reopen.
func verifySnapshotPayloadCRC(r io.Reader, size int64, expected uint32) error {
	h := crc32.New(walCRC)
	if _, err := io.CopyN(h, r, size); err != nil {
		return ErrSnapshotCorrupt
	}
	if h.Sum32() != expected {
		return ErrSnapshotCorrupt
	}
	return nil
}

// countingReader wraps an io.Reader so the call site can observe
// how many bytes the installer actually consumed. Used by install
// paths to log an advisory warning when the installer stopped
// short of the snapshot's declared size; integrity is already
// established by the CRC pre-verify, this is strictly an
// observability hook.
type countingReader struct {
	r     io.Reader
	count int64
}

// Read delegates to the wrapped reader and advances count by the
// number of bytes returned. The error value is forwarded as-is so
// the caller can distinguish short-read EOFs from underlying
// failures.
func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.count += int64(n)
	return n, err
}

// readSnapshotHeader advances r past the magic + header + CRC
// prefix and returns the embedded metadata. The returned reader
// position is at the start of the data payload. The second
// return value is the expected CRC32C of the payload, exposed so
// streaming callers can verify it on their own if they want.
func readSnapshotHeader(r io.Reader) (SnapshotMeta, uint32, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return SnapshotMeta{}, 0, ErrSnapshotCorrupt
	}
	if magic != snapshotMagic {
		return SnapshotMeta{}, 0, ErrSnapshotCorrupt
	}

	var hdr [20]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return SnapshotMeta{}, 0, ErrSnapshotCorrupt
	}
	lastIdx := binary.LittleEndian.Uint64(hdr[0:8])
	lastTerm := binary.LittleEndian.Uint64(hdr[8:16])
	dataLen := binary.LittleEndian.Uint32(hdr[16:20])

	var crcBuf [4]byte
	if _, err := io.ReadFull(r, crcBuf[:]); err != nil {
		return SnapshotMeta{}, 0, ErrSnapshotCorrupt
	}
	expectedCRC := binary.LittleEndian.Uint32(crcBuf[:])

	return SnapshotMeta{
		LastIncludedIndex: lastIdx,
		LastIncludedTerm:  lastTerm,
		Size:              int64(dataLen),
	}, expectedCRC, nil
}
