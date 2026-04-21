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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"mxkeys/internal/zero/log"
)

// pendingSnapshotFileName is the spill file used while an incoming
// InstallSnapshot transfer is in flight. It is rewritten from the
// beginning on every new transfer and removed on completion; a
// crash mid-transfer leaves it behind, and LoadFromDisk removes
// any stale copy before the next transfer can start.
const pendingSnapshotFileName = "raft.snapshot.recv"

// ErrPendingSnapshotOverflow is returned when a chunk append would
// push the accumulated transfer past maxSnapshotSize. The handler
// surfaces this as an explicit Success=false rejection so the
// leader aborts the stream rather than continuing to consume
// follower memory.
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

// resetPendingSnapshot discards any in-memory or on-disk state
// left from a prior or aborted transfer. Caller must hold n.mu.
//
// For the disk-backed path the file is closed and removed; for the
// in-memory path the buffer is truncated in place. The error is
// logged but not returned because every caller is already on a
// rejection or new-transfer branch that cannot meaningfully act on
// a cleanup failure.
func (n *Node) resetPendingSnapshot() {
	n.pendingSnapshot = n.pendingSnapshot[:0]
	n.pendingSnapshotExpected = 0
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
// disk-backed mode it opens the spill file with O_TRUNC so any
// stale tail is discarded. Caller must hold n.mu.
func (n *Node) beginPendingSnapshot(lastIncludedIndex, lastIncludedTerm uint64) error {
	n.resetPendingSnapshot()
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
	n.pendingSnapshotFile = f
	n.pendingSnapshotPath = path
	return nil
}

// appendPendingSnapshot writes chunk bytes into the active
// transfer, enforcing maxSnapshotSize as a hard cap before any
// write hits disk or RAM. Returns ErrPendingSnapshotOverflow when
// the cap would be exceeded; the caller must translate this into a
// Success=false response and reset the transfer. Caller must hold
// n.mu.
func (n *Node) appendPendingSnapshot(data []byte) error {
	newSize := n.pendingSnapshotExpected + uint64(len(data))
	if newSize > maxSnapshotSize {
		return ErrPendingSnapshotOverflow
	}
	if n.pendingSnapshotFile != nil {
		if _, err := n.pendingSnapshotFile.Write(data); err != nil {
			return fmt.Errorf("raft: write pending snapshot chunk: %w", err)
		}
	} else {
		n.pendingSnapshot = append(n.pendingSnapshot, data...)
	}
	n.pendingSnapshotExpected = newSize
	return nil
}

// drainPendingSnapshot returns the fully-accumulated payload for
// installer and SaveSnapshot, and clears the in-memory counters
// so the next transfer starts from zero. For the disk-backed path
// the spill file is fsync'd and then read in full; the file
// itself is left on disk for finalizePendingSnapshotFile to
// either rename or remove based on installer outcome. Caller
// must hold n.mu.
func (n *Node) drainPendingSnapshot() ([]byte, error) {
	var data []byte
	if n.pendingSnapshotFile != nil {
		if err := n.pendingSnapshotFile.Sync(); err != nil {
			return nil, fmt.Errorf("raft: sync pending snapshot: %w", err)
		}
		if _, err := n.pendingSnapshotFile.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("raft: seek pending snapshot: %w", err)
		}
		buf := make([]byte, n.pendingSnapshotExpected)
		if _, err := io.ReadFull(n.pendingSnapshotFile, buf); err != nil {
			return nil, fmt.Errorf("raft: read pending snapshot: %w", err)
		}
		data = buf
	} else {
		data = append([]byte(nil), n.pendingSnapshot...)
		n.pendingSnapshot = n.pendingSnapshot[:0]
	}
	n.pendingSnapshotExpected = 0
	return data, nil
}
