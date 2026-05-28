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
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// TruncateAfter rewrites the WAL, dropping every entry whose Index is
// strictly greater than lastKeepIndex. Pass 0 to wipe the log entirely.
//
// Implementation uses temp-file + rename so a crash mid-truncate leaves the
// previous WAL intact. Callers must hold the Raft node lock to ensure no
// concurrent Append races a rewrite.
func (w *WAL) TruncateAfter(lastKeepIndex uint64) error {
	return w.filterRewrite(func(e LogEntry) bool {
		return e.Index <= lastKeepIndex
	})
}

// TruncateBefore rewrites the WAL, dropping every entry whose Index is
// strictly less than firstKeepIndex. Used during snapshot compaction: after
// a snapshot at index N the log can safely forget everything at or below N.
func (w *WAL) TruncateBefore(firstKeepIndex uint64) error {
	return w.filterRewrite(func(e LogEntry) bool {
		return e.Index >= firstKeepIndex
	})
}

// filterRewrite reads the current WAL, keeps entries that pass predicate, and
// atomically rewrites the file. Corrupt tails are treated as truncated: the
// well-formed prefix is preserved, the bad suffix is dropped.
func (w *WAL) filterRewrite(keep func(LogEntry) bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entries, err := w.readAllLocked()
	if err != nil && !errors.Is(err, ErrWALCorrupt) {
		return fmt.Errorf("raft wal: truncate read: %w", err)
	}

	filtered := entries[:0]
	for _, e := range entries {
		if keep(e) {
			filtered = append(filtered, e)
		}
	}
	return w.rewriteLocked(filtered)
}

// rewriteLocked atomically replaces the WAL with a new file containing keep.
// Requires w.mu held.
//
// The replacement file is written with the current walMagic prefix so a
// rewritten WAL is indistinguishable from a freshly-created one. This keeps
// the format version monotonic: a partially-rewritten state cannot appear
// to be an older version to a future reader.
func (w *WAL) rewriteLocked(keep []LogEntry) error {
	tmpPath := filepath.Join(w.dir, walFileName+".rewrite")
	tmp, err := os.OpenFile(tmpPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("raft wal: open tmp: %w", err)
	}

	if _, err := tmp.Write(walMagic[:]); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("raft wal: write tmp magic: %w", err)
	}

	// Temporarily swap w.file to the tmp handle so appendLocked can reuse the
	// same write path. Restored on any failure below.
	original := w.file
	w.file = tmp
	for _, e := range keep {
		if err := w.appendLocked(e); err != nil {
			w.file = original
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
			return err
		}
	}
	w.file = original

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("raft wal: fsync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("raft wal: close tmp: %w", err)
	}

	// Close the live file before rename (some platforms require this).
	if w.file != nil {
		_ = w.file.Close()
	}
	if err := os.Rename(tmpPath, w.path); err != nil {
		return fmt.Errorf("raft wal: rename: %w", err)
	}

	// fsync the directory so the rename itself is durable on power loss.
	if d, err := os.Open(w.dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}

	reopened, err := os.OpenFile(w.path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("raft wal: reopen: %w", err)
	}
	w.file = reopened
	return nil
}
