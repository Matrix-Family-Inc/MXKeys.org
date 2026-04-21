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
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
)

// snapshotFileName is the current-generation snapshot on disk. Snapshot
// writes go to a .tmp sibling and atomically rename on success.
const snapshotFileName = "raft.snapshot"

// snapshotMagic prefixes every snapshot file so a stray misaligned read
// fails fast rather than parsing garbage as state.
var snapshotMagic = [4]byte{'M', 'X', 'K', 'S'}

// maxSnapshotSize caps the size of an on-disk snapshot. At 256 MiB
// we comfortably fit any practical notary state while keeping the
// length field in the header bounded to uint32.
const maxSnapshotSize = 256 << 20

// SnapshotMeta describes the point in the replicated log that a snapshot
// captures.
type SnapshotMeta struct {
	// LastIncludedIndex is the highest Raft log index reflected in the
	// snapshot state; the log prefix up to and including this index has been
	// applied to the state machine.
	LastIncludedIndex uint64 `json:"last_included_index"`
	// LastIncludedTerm is the term of LastIncludedIndex (required to keep
	// log-consistency invariants after compaction).
	LastIncludedTerm uint64 `json:"last_included_term"`
	// Size is the payload length in bytes; stored for logging/metrics.
	Size int64 `json:"size"`
}

// Snapshot bundles metadata with the opaque state-machine bytes.
type Snapshot struct {
	Meta SnapshotMeta
	Data []byte
}

// ErrNoSnapshot is returned by LoadSnapshot when the snapshot file is
// missing. Callers should treat this as "no durable snapshot yet" and
// replay the full WAL.
var ErrNoSnapshot = errors.New("raft snapshot: no snapshot")

// ErrSnapshotCorrupt indicates a malformed snapshot file: bad magic, bad
// header CRC, bad payload CRC, or truncated data. On production startup this
// is fatal; on tests it is exercised explicitly.
var ErrSnapshotCorrupt = errors.New("raft snapshot: corrupt file")

// snapshotFilePath returns the canonical path within dir.
func snapshotFilePath(dir string) string {
	return filepath.Join(dir, snapshotFileName)
}

// SaveSnapshot writes meta+data atomically to dir/raft.snapshot.
// Format (little-endian):
//
//	[4 magic][8 last_index][8 last_term][4 data_len][4 data_crc][N data]
func SaveSnapshot(dir string, s Snapshot) error {
	if dir == "" {
		return fmt.Errorf("raft snapshot: dir is required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("raft snapshot: mkdir: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("raft snapshot: chmod: %w", err)
	}

	tmpPath := snapshotFilePath(dir) + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("raft snapshot: open tmp: %w", err)
	}

	if _, err := f.Write(snapshotMagic[:]); err != nil {
		closeRemove(f, tmpPath)
		return fmt.Errorf("raft snapshot: write magic: %w", err)
	}

	var hdr [20]byte
	binary.LittleEndian.PutUint64(hdr[0:8], s.Meta.LastIncludedIndex)
	binary.LittleEndian.PutUint64(hdr[8:16], s.Meta.LastIncludedTerm)
	dataLen, err := lenUint32("raft snapshot body", s.Data, maxSnapshotSize)
	if err != nil {
		closeRemove(f, tmpPath)
		return err
	}
	binary.LittleEndian.PutUint32(hdr[16:20], dataLen)
	if _, err := f.Write(hdr[:]); err != nil {
		closeRemove(f, tmpPath)
		return fmt.Errorf("raft snapshot: write header: %w", err)
	}

	var crcBuf [4]byte
	binary.LittleEndian.PutUint32(crcBuf[:], crc32.Checksum(s.Data, walCRC))
	if _, err := f.Write(crcBuf[:]); err != nil {
		closeRemove(f, tmpPath)
		return fmt.Errorf("raft snapshot: write crc: %w", err)
	}

	if _, err := f.Write(s.Data); err != nil {
		closeRemove(f, tmpPath)
		return fmt.Errorf("raft snapshot: write data: %w", err)
	}

	if err := f.Sync(); err != nil {
		closeRemove(f, tmpPath)
		return fmt.Errorf("raft snapshot: fsync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("raft snapshot: close tmp: %w", err)
	}

	if err := os.Rename(tmpPath, snapshotFilePath(dir)); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("raft snapshot: rename: %w", err)
	}
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
}

// LoadSnapshot reads dir/raft.snapshot. Returns ErrNoSnapshot when the file
// is missing, ErrSnapshotCorrupt when its CRC or magic fails.
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

	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, ErrSnapshotCorrupt
	}
	if magic != snapshotMagic {
		return nil, ErrSnapshotCorrupt
	}

	var hdr [20]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return nil, ErrSnapshotCorrupt
	}
	lastIdx := binary.LittleEndian.Uint64(hdr[0:8])
	lastTerm := binary.LittleEndian.Uint64(hdr[8:16])
	dataLen := binary.LittleEndian.Uint32(hdr[16:20])

	var crcBuf [4]byte
	if _, err := io.ReadFull(f, crcBuf[:]); err != nil {
		return nil, ErrSnapshotCorrupt
	}
	expectedCRC := binary.LittleEndian.Uint32(crcBuf[:])

	data := make([]byte, dataLen)
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, ErrSnapshotCorrupt
	}

	if crc32.Checksum(data, walCRC) != expectedCRC {
		return nil, ErrSnapshotCorrupt
	}

	return &Snapshot{
		Meta: SnapshotMeta{
			LastIncludedIndex: lastIdx,
			LastIncludedTerm:  lastTerm,
			Size:              int64(dataLen),
		},
		Data: data,
	}, nil
}

// closeRemove is a cleanup helper used by SaveSnapshot failure paths.
func closeRemove(f *os.File, path string) {
	_ = f.Close()
	_ = os.Remove(path)
}

// --- State-machine snapshot callbacks -----------------------------------

// SnapshotProvider produces the state-machine bytes to persist in a new
// snapshot. It is invoked by the Raft node when it decides to compact the
// log; the caller is the application layer (e.g. CRDT state holder).
type SnapshotProvider func() ([]byte, error)

// SnapshotInstaller installs state-machine bytes received via
// InstallSnapshot RPC or loaded from disk during startup. Must be idempotent
// for the same (lastIndex, lastTerm) pair.
type SnapshotInstaller func(data []byte, lastIncludedIndex, lastIncludedTerm uint64) error

// InstallSnapshotRequest is sent by a leader to fast-forward a follower
// that lags past the leader's truncated log prefix.
//
// Chunked transport: snapshots larger than snapshotChunkSize are split
// into multiple RPCs. All RPCs in a run share the same (Term,
// LastIncludedIndex, LastIncludedTerm) tuple. Offset advances
// monotonically from 0 in units of Data length; the follower
// concatenates Data in order. Done=true signals the last chunk, at
// which point the follower applies the full buffer to its state
// machine and persists it.
//
// Single-shot RPCs (Offset=0, Done=true) remain fully supported for
// small snapshots and for backward compatibility with earlier builds.
type InstallSnapshotRequest struct {
	Term              uint64 `json:"term"`
	LeaderID          string `json:"leader_id"`
	LastIncludedIndex uint64 `json:"last_included_index"`
	LastIncludedTerm  uint64 `json:"last_included_term"`
	Offset            uint64 `json:"offset"`
	Done              bool   `json:"done"`
	// Data is a chunk of the snapshot payload. Binary-safe: marshaled
	// as base64 by encoding/json, which is why []byte and not
	// json.RawMessage is the right type here because snapshots need not be
	// valid JSON.
	Data []byte `json:"data"`
}

// InstallSnapshotResponse acknowledges the install and returns the
// follower's current term.
//
// Success semantics (production Raft contract):
//
//   - A non-Done chunk ACKs with Success=true only when the follower
//     accepted the chunk into its pending-snapshot buffer (no term,
//     offset, or tuple mismatch).
//   - A Done chunk ACKs with Success=true only when the follower also
//     successfully applied the reassembled payload via the registered
//     SnapshotInstaller and persisted it to disk.
//   - Success=false on any rejection (stale term, offset gap, decode
//     error, installer error, save error). The leader MUST NOT advance
//     nextIndex/matchIndex for the peer on Success=false; it should
//     retry the snapshot from offset 0 on the next replication pass.
//
// BytesStored is the number of payload bytes the follower currently
// has buffered for the (LastIncludedIndex, LastIncludedTerm) tuple
// after processing this chunk. The leader uses it to detect silent
// buffer resets (e.g. after the follower observed a gap and dropped
// the partial transfer) and restart from offset 0.
type InstallSnapshotResponse struct {
	Term        uint64 `json:"term"`
	Success     bool   `json:"success"`
	BytesStored uint64 `json:"bytes_stored,omitempty"`
}

// MsgInstallSnapshot is the RPC type for InstallSnapshot.
const (
	MsgInstallSnapshot    MessageType = "install_snapshot"
	MsgInstallSnapshotRes MessageType = "install_snapshot_response"
)
