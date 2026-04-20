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

	"mxkeys/internal/zero/log"
)

// SetStateDir attaches a persistent state directory to the node. The same
// directory holds the WAL and the snapshot file. Must be called before
// Start; subsequent calls are no-ops to keep the invariant that persistence
// is bound once per node lifetime.
//
// When stateDir is empty the node runs in the legacy in-memory mode and no
// durability guarantees apply.
func (n *Node) SetStateDir(stateDir string, syncOnAppend bool) error {
	if stateDir == "" {
		return fmt.Errorf("raft: state dir is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.wal != nil {
		return nil
	}

	w, err := OpenWAL(WALOptions{Dir: stateDir, SyncOnAppend: syncOnAppend})
	if err != nil {
		return err
	}
	n.wal = w
	n.stateDir = stateDir
	return nil
}

// SetSnapshotProvider registers the state-machine snapshot callback used
// when CompactLog is invoked. See SnapshotProvider.
func (n *Node) SetSnapshotProvider(fn SnapshotProvider) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.snapshotProvider = fn
}

// SetSnapshotInstaller registers the state-machine snapshot-install callback
// used on startup and when a leader sends InstallSnapshot. See
// SnapshotInstaller.
func (n *Node) SetSnapshotInstaller(fn SnapshotInstaller) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.snapshotInstaller = fn
}

// LoadFromDisk restores node state from the persisted snapshot + WAL.
// Safe to call before Start; a no-op when SetStateDir was not called.
//
// Recovery semantics:
//
//  1. If a snapshot exists, install it (via snapshotInstaller if set) and
//     set snapshotIndex/snapshotTerm/commitIndex/lastApplied.
//  2. Replay the WAL: every record whose Index > snapshotIndex is appended
//     to n.log. Corrupt tails truncate the replay (ErrWALCorrupt).
//  3. currentTerm is advanced to max(term in replayed entries,
//     snapshotTerm) so a future election sees a term at least as high as
//     anything already durable.
func (n *Node) LoadFromDisk() error {
	if n.stateDir == "" {
		return nil
	}

	snap, err := LoadSnapshot(n.stateDir)
	switch {
	case errors.Is(err, ErrNoSnapshot):
		// Nothing persisted yet. Carry on to WAL replay.
	case err != nil:
		return fmt.Errorf("raft: load snapshot: %w", err)
	default:
		if n.snapshotInstaller != nil {
			if err := n.snapshotInstaller(snap.Data, snap.Meta.LastIncludedIndex, snap.Meta.LastIncludedTerm); err != nil {
				return fmt.Errorf("raft: install snapshot: %w", err)
			}
		}
		n.mu.Lock()
		n.snapshotIndex = snap.Meta.LastIncludedIndex
		n.snapshotTerm = snap.Meta.LastIncludedTerm
		n.commitIndex = snap.Meta.LastIncludedIndex
		n.lastApplied = snap.Meta.LastIncludedIndex
		if snap.Meta.LastIncludedTerm > n.currentTerm {
			n.currentTerm = snap.Meta.LastIncludedTerm
		}
		n.mu.Unlock()
		log.Info("Raft snapshot loaded",
			"last_included_index", snap.Meta.LastIncludedIndex,
			"last_included_term", snap.Meta.LastIncludedTerm,
			"size", snap.Meta.Size,
		)
	}

	if n.wal == nil {
		return nil
	}
	entries, werr := n.wal.ReadAll()
	if werr != nil && !errors.Is(werr, ErrWALCorrupt) {
		return fmt.Errorf("raft: read wal: %w", werr)
	}
	if errors.Is(werr, ErrWALCorrupt) {
		log.Warn("Raft WAL tail corrupt; truncating to last well-formed record",
			"kept_entries", len(entries),
		)
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	for _, e := range entries {
		if e.Index <= n.snapshotIndex {
			// Already captured by the snapshot; ignore.
			continue
		}
		n.log = append(n.log, e)
		if e.Term > n.currentTerm {
			n.currentTerm = e.Term
		}
	}
	return nil
}

// persistEntry durably stores a single appended log entry.
// Caller may hold n.mu; WAL has its own mutex.
func (n *Node) persistEntry(entry LogEntry) error {
	if n.wal == nil {
		return nil
	}
	return n.wal.Append(entry)
}

// persistEntries durably stores a slice of entries in order.
func (n *Node) persistEntries(entries []LogEntry) error {
	if n.wal == nil {
		return nil
	}
	for _, e := range entries {
		if err := n.wal.Append(e); err != nil {
			return err
		}
	}
	return nil
}

// truncateLogAfter drops log entries with Index > lastKeepIndex from both
// the in-memory slice and the WAL. Used when a follower's tail conflicts
// with the leader's view of history.
//
// Caller must hold n.mu.
func (n *Node) truncateLogAfter(lastKeepIndex uint64) error {
	if uint64(len(n.log)) > lastKeepIndex {
		// Log is 1-indexed via Entry.Index; the in-memory slice is 0-indexed.
		n.log = n.log[:lastKeepIndex]
	}
	if n.wal == nil {
		return nil
	}
	return n.wal.TruncateAfter(lastKeepIndex)
}

// CompactLog produces a new on-disk snapshot at the current applied index
// and truncates the WAL prefix. Requires a registered SnapshotProvider.
//
// The in-memory log slice is intentionally not truncated here; the snapshot
// is for catch-up (InstallSnapshot RPC) and WAL-size containment, not for
// in-process memory reclaim. Callers that need memory reclaim must manage
// it at a higher layer.
func (n *Node) CompactLog() error {
	if n.stateDir == "" || n.snapshotProvider == nil {
		return fmt.Errorf("raft: compaction requires state dir and snapshot provider")
	}

	n.mu.RLock()
	lastIdx := n.lastApplied
	var lastTerm uint64
	if lastIdx > 0 && int(lastIdx) <= len(n.log) {
		lastTerm = n.log[lastIdx-1].Term
	} else {
		lastTerm = n.snapshotTerm
	}
	n.mu.RUnlock()

	if lastIdx == 0 {
		return fmt.Errorf("raft: nothing to compact")
	}

	data, err := n.snapshotProvider()
	if err != nil {
		return fmt.Errorf("raft: snapshot provider: %w", err)
	}

	snap := Snapshot{
		Meta: SnapshotMeta{
			LastIncludedIndex: lastIdx,
			LastIncludedTerm:  lastTerm,
			Size:              int64(len(data)),
		},
		Data: data,
	}
	if err := SaveSnapshot(n.stateDir, snap); err != nil {
		return err
	}

	n.mu.Lock()
	n.snapshotIndex = lastIdx
	n.snapshotTerm = lastTerm
	n.mu.Unlock()

	if n.wal != nil {
		if err := n.wal.TruncateBefore(lastIdx + 1); err != nil {
			return fmt.Errorf("raft: compact wal: %w", err)
		}
	}

	log.Info("Raft log compacted",
		"snapshot_index", lastIdx,
		"snapshot_term", lastTerm,
		"snapshot_bytes", len(data),
	)
	return nil
}
