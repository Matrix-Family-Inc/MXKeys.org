/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Updated
 */

package raft

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"

	"mxkeys/internal/zero/log"
)

// pendingSnapshotFileName is the spill file used while an incoming
// InstallSnapshot transfer is in flight. Once finalised it has the
// exact byte layout of a raft.snapshot file, so the Done=true
// terminus can atomically rename the spill into place without a
// second disk pass through SaveSnapshot. A crash mid-transfer
// leaves the file behind; LoadFromDisk removes any stale copy on
// the next start.
const pendingSnapshotFileName = "raft.snapshot.recv"

// ErrPendingSnapshotOverflow is returned when a chunk append would
// push the accumulated transfer past maxSnapshotSize. The handler
// surfaces this as an explicit Success=false rejection so the
// leader aborts the stream rather than continuing to consume
// follower memory or disk.
var ErrPendingSnapshotOverflow = errors.New("raft: pending snapshot exceeds maxSnapshotSize")

// pendingSnapshotPathFor returns the spill file path for a given
// state directory. Shared between the handler, LoadFromDisk
// cleanup, and tests.
func pendingSnapshotPathFor(stateDir string) string {
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, pendingSnapshotFileName)
}

// cleanupStalePendingSnapshot removes any leftover spill file from
// a previous crashed transfer. Safe to call before Start; no-op
// when the file is absent. Errors are logged at Warn level rather
// than returned because they do not compromise correctness: a
// subsequent transfer will truncate the file on first write.
func cleanupStalePendingSnapshot(stateDir string) {
	path := pendingSnapshotPathFor(stateDir)
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warn("Raft stale pending snapshot cleanup failed",
			"path", path,
			"error", err,
		)
	}
}

// releasePendingSnapshot discards any in-memory or on-disk state
// left from a prior or aborted transfer. Caller must hold n.mu.
// Idempotent; safe to call from every failure branch.
func (n *Node) releasePendingSnapshot() {
	n.pendingSnapshot = n.pendingSnapshot[:0]
	n.pendingSnapshotExpected = 0
	n.pendingSnapshotIndex = 0
	n.pendingSnapshotTerm = 0
	n.pendingSnapshotCRC = nil
	if n.pendingSnapshotFile != nil {
		_ = n.pendingSnapshotFile.Close()
		n.pendingSnapshotFile = nil
	}
	if n.pendingSnapshotPath != "" {
		if err := os.Remove(n.pendingSnapshotPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Warn("Raft pending snapshot cleanup failed",
				"path", n.pendingSnapshotPath,
				"error", err,
			)
		}
		n.pendingSnapshotPath = ""
	}
}

// beginPendingSnapshot initialises a fresh transfer. In the
// disk-backed mode it opens the spill file and writes the
// SaveSnapshot header with placeholder length/CRC fields; the
// placeholders are replaced by finalizePendingSnapshot at the end
// of the transfer. This keeps the spill file's byte layout
// identical to a final raft.snapshot so the terminus can rename
// the spill into place atomically without a second-pass write.
//
// Caller must hold n.mu.
func (n *Node) beginPendingSnapshot(lastIncludedIndex, lastIncludedTerm uint64) error {
	n.releasePendingSnapshot()
	n.pendingSnapshotIndex = lastIncludedIndex
	n.pendingSnapshotTerm = lastIncludedTerm

	if n.stateDir == "" {
		return nil
	}

	if err := os.MkdirAll(n.stateDir, 0o700); err != nil {
		return fmt.Errorf("raft: mkdir state dir: %w", err)
	}
	path := pendingSnapshotPathFor(n.stateDir)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("raft: open pending snapshot: %w", err)
	}

	if err := writePendingSnapshotHeader(f, lastIncludedIndex, lastIncludedTerm); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return err
	}

	n.pendingSnapshotFile = f
	n.pendingSnapshotPath = path
	n.pendingSnapshotCRC = crc32.New(walCRC)
	return nil
}

// writePendingSnapshotHeader writes the snapshot magic and the
// header block with placeholder data_len (zero) and data_crc
// (zero). The placeholders are patched in by
// finalizePendingSnapshot once every chunk has been appended and
// the running CRC is complete.
func writePendingSnapshotHeader(f *os.File, lastIncludedIndex, lastIncludedTerm uint64) error {
	if _, err := f.Write(snapshotMagic[:]); err != nil {
		return fmt.Errorf("raft: write pending snapshot magic: %w", err)
	}
	var hdr [20]byte
	binary.LittleEndian.PutUint64(hdr[0:8], lastIncludedIndex)
	binary.LittleEndian.PutUint64(hdr[8:16], lastIncludedTerm)
	// hdr[16:20] is data_len: leave zero until finalise.
	if _, err := f.Write(hdr[:]); err != nil {
		return fmt.Errorf("raft: write pending snapshot header: %w", err)
	}
	var crcPlaceholder [4]byte
	if _, err := f.Write(crcPlaceholder[:]); err != nil {
		return fmt.Errorf("raft: write pending snapshot crc placeholder: %w", err)
	}
	return nil
}

// appendPendingSnapshot writes chunk bytes into the active transfer
// and updates the running CRC. Enforces maxSnapshotSize as a hard
// cap before any write hits disk or RAM. Caller must hold n.mu.
func (n *Node) appendPendingSnapshot(data []byte) error {
	newSize := n.pendingSnapshotExpected + uint64(len(data))
	if newSize > maxSnapshotSize {
		return ErrPendingSnapshotOverflow
	}
	if n.pendingSnapshotFile != nil {
		if _, err := n.pendingSnapshotFile.Write(data); err != nil {
			return fmt.Errorf("raft: write pending snapshot chunk: %w", err)
		}
		// hash.Hash32.Write never returns a non-nil error; ignore.
		_, _ = n.pendingSnapshotCRC.Write(data)
	} else {
		n.pendingSnapshot = append(n.pendingSnapshot, data...)
	}
	n.pendingSnapshotExpected = newSize
	return nil
}

// finalizePendingSnapshot patches the real data_len and data_crc
// into the spill file's header, fsyncs, seeks the reader to the
// start of the data portion, and transfers ownership of the file
// handle and path to the caller. Every pendingSnapshot* field on
// the node is cleared so concurrent handlers cannot trip over a
// half-owned transfer.
//
// Returns (file, path, dataSize, err). Ownership semantics:
//
//   - The caller MUST close the returned file.
//   - On installer success the caller MUST rename path →
//     raft.snapshot (atomic) and fsync the directory.
//   - On installer failure the caller MUST os.Remove(path).
//
// Disk-backed mode only. For the in-memory fallback use
// drainPendingSnapshotInMemory.
//
// Caller must hold n.mu.
func (n *Node) finalizePendingSnapshot() (*os.File, string, int64, error) {
	if n.pendingSnapshotFile == nil {
		return nil, "", 0, fmt.Errorf("raft: no disk-backed pending snapshot to finalise")
	}
	f := n.pendingSnapshotFile
	path := n.pendingSnapshotPath
	size := n.pendingSnapshotExpected
	crc := n.pendingSnapshotCRC

	// Detach ownership BEFORE the potentially-fallible I/O below so
	// a partial failure leaves Node.pending* cleanly empty and any
	// concurrent handler starting a new transfer cannot inherit a
	// dangling pointer to a file we are about to close.
	n.pendingSnapshotFile = nil
	n.pendingSnapshotPath = ""
	n.pendingSnapshotExpected = 0
	n.pendingSnapshotIndex = 0
	n.pendingSnapshotTerm = 0
	n.pendingSnapshotCRC = nil

	if size > uint64(^uint32(0)) {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, "", 0, fmt.Errorf("raft: pending snapshot size %d overflows uint32 header field", size)
	}

	var buf [8]byte
	binary.LittleEndian.PutUint32(buf[0:4], uint32(size))
	binary.LittleEndian.PutUint32(buf[4:8], crcSum(crc))
	if _, err := f.WriteAt(buf[:], snapshotHeaderLenOffset); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, "", 0, fmt.Errorf("raft: patch pending snapshot header: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, "", 0, fmt.Errorf("raft: fsync pending snapshot: %w", err)
	}
	if _, err := f.Seek(int64(snapshotHeaderSize), io.SeekStart); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return nil, "", 0, fmt.Errorf("raft: seek pending snapshot to data: %w", err)
	}
	return f, path, int64(size), nil
}

// crcSum is a shim around hash.Hash32.Sum32 so callers don't need to
// import hash themselves when all they want is the checksum.
func crcSum(h hash.Hash32) uint32 {
	if h == nil {
		return 0
	}
	return h.Sum32()
}

// drainPendingSnapshotInMemory is the in-memory fallback used when
// the node has no state directory (tests). Returns the accumulated
// bytes and clears every pendingSnapshot* field. Caller must hold
// n.mu.
func (n *Node) drainPendingSnapshotInMemory() []byte {
	data := append([]byte(nil), n.pendingSnapshot...)
	n.pendingSnapshot = n.pendingSnapshot[:0]
	n.pendingSnapshotExpected = 0
	n.pendingSnapshotIndex = 0
	n.pendingSnapshotTerm = 0
	return data
}

// finalizeAndRenamePendingSnapshot is the success-path helper used
// by handleInstallSnapshot. It renames the finalised spill file to
// the canonical snapshot path and fsyncs the directory so the new
// snapshot is durable. The file handle MUST already be closed by
// the caller (installer returned) before this rename runs; Linux
// tolerates rename with an open fd but closing is cheaper and
// removes the Windows caveat if the project ever targets it.
func finalizeAndRenamePendingSnapshot(stateDir, spillPath string) error {
	finalPath := snapshotFilePath(stateDir)
	if err := os.Rename(spillPath, finalPath); err != nil {
		_ = os.Remove(spillPath)
		return fmt.Errorf("raft: rename pending snapshot: %w", err)
	}
	syncSnapshotDir(stateDir)
	return nil
}
