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
// expected CRC span, and returns a reader positioned at the start
// of the data portion along with the snapshot metadata. Caller
// MUST close the returned file when done.
//
// Unlike LoadSnapshot, no byte of the payload is buffered: peak
// memory is one header read. Callers who need cryptographic CRC
// verification of the data can wrap the returned reader with a
// tee into crc32.New(walCRC) and compare at EOF; LoadFromDisk
// trusts the rename+fsync durability invariant of the filesystem
// because the file was written by SaveSnapshot, which verifies
// the CRC matches the data before renaming the tmp into place.
func LoadSnapshotReader(dir string) (*os.File, SnapshotMeta, error) {
	path := snapshotFilePath(dir)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, SnapshotMeta{}, ErrNoSnapshot
		}
		return nil, SnapshotMeta{}, fmt.Errorf("raft snapshot: open: %w", err)
	}
	meta, _, err := readSnapshotHeader(f)
	if err != nil {
		_ = f.Close()
		return nil, SnapshotMeta{}, err
	}
	return f, meta, nil
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
