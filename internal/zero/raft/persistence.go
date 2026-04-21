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

	if n.config.SharedSecret == "" {
		return fmt.Errorf("raft: SharedSecret is required to open the WAL (used to derive the WAL HMAC key)")
	}
	w, err := OpenWAL(WALOptions{
		Dir:          stateDir,
		SyncOnAppend: syncOnAppend,
		HMACKey:      []byte(n.config.SharedSecret),
	})
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
// Recovery:
//  1. If a snapshot exists, install it (via snapshotInstaller if set)
//     and set snapshotIndex/snapshotTerm/commitIndex/lastApplied.
//  2. Replay WAL records with Index > snapshotIndex. Corrupt tails
//     truncate the replay (ErrWALCorrupt).
//  3. currentTerm advances to max(replayed-term, snapshotTerm).
func (n *Node) LoadFromDisk() error {
	if n.stateDir == "" {
		return nil
	}

	// A previous transfer that crashed mid-stream may have left a
	// spill file behind. Remove it before the node accepts new
	// InstallSnapshot chunks so the next transfer starts from a
	// known-empty state instead of inheriting partial bytes.
	cleanupStalePendingSnapshot(n.stateDir)

	f, meta, err := LoadSnapshotReader(n.stateDir)
	switch {
	case errors.Is(err, ErrNoSnapshot):
		// Nothing persisted yet. Carry on to WAL replay.
	case err != nil:
		return fmt.Errorf("raft: load snapshot: %w", err)
	default:
		if n.snapshotInstaller != nil {
			// CRC already verified by LoadSnapshotReader; the
			// counter is for advisory short-read warnings only.
			cr := &countingReader{r: f}
			if ierr := n.snapshotInstaller(cr, meta.Size, meta.LastIncludedIndex, meta.LastIncludedTerm); ierr != nil {
				_ = f.Close()
				return fmt.Errorf("raft: install snapshot: %w", ierr)
			}
			if cr.count < meta.Size {
				log.Warn("Raft startup installer did not drain full snapshot",
					"read", cr.count,
					"expected", meta.Size,
				)
			}
		}
		_ = f.Close()
		n.mu.Lock()
		n.snapshotIndex = meta.LastIncludedIndex
		n.snapshotTerm = meta.LastIncludedTerm
		// logOffset starts at the snapshot boundary so subsequent WAL
		// entries (Index > snapshotIndex) index correctly into n.log.
		n.logOffset = meta.LastIncludedIndex
		n.commitIndex = meta.LastIncludedIndex
		n.lastApplied = meta.LastIncludedIndex
		if meta.LastIncludedTerm > n.currentTerm {
			n.currentTerm = meta.LastIncludedTerm
		}
		n.mu.Unlock()
		log.Info("Raft snapshot loaded",
			"last_included_index", meta.LastIncludedIndex,
			"last_included_term", meta.LastIncludedTerm,
			"size", meta.Size,
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

// truncateLogAfter drops log entries with Index > lastKeepIndex from both
// the in-memory slice and the WAL. Used when a follower's tail conflicts
// with the leader's view of history.
//
// Caller must hold n.mu.
func (n *Node) truncateLogAfter(lastKeepIndex uint64) error {
	n.truncateSliceAfter(lastKeepIndex)
	if n.wal == nil {
		return nil
	}
	return n.wal.TruncateAfter(lastKeepIndex)
}

// CompactLog persists a new on-disk snapshot at the index the
// state machine reached, truncates the WAL prefix, and drops the
// covered in-memory log prefix. Requires a registered
// SnapshotProvider.
//
// Atomicity: LastIncludedIndex is whatever the provider returned
// (captured by the application under its own lock together with
// the payload). Two replicas at the same applied prefix therefore
// produce byte-identical files at the same LastIncludedIndex.
//
// Concurrency: n.snapMu serialises this method with every
// handleInstallSnapshot, so no other snapshot writer can interleave
// its SaveSnapshot between our validation and persist; raft.snapshot
// on disk never rolls back. The defensive post-persist re-check is
// a belt-and-suspenders guard for future snapMu-exempt writers.
//
// Post-compact: in-memory log holds only Index > snapshotIndex,
// logOffset advances to snapshotIndex.
func (n *Node) CompactLog() error {
	if n.stateDir == "" || n.snapshotProvider == nil {
		return fmt.Errorf("raft: compaction requires state dir and snapshot provider")
	}

	// Serialise with any concurrent handleInstallSnapshot BEFORE we
	// call the application's provider. The provider captures state
	// under its own locks; if we did not hold snapMu here, an
	// InstallSnapshot handler could complete its full install (and
	// persist raft.snapshot with a strictly newer LastIncludedIndex)
	// while we were still inside the provider. Our later SaveSnapshot
	// would then overwrite the newer on-disk file with our older
	// payload.
	n.snapMu.Lock()
	defer n.snapMu.Unlock()

	data, appliedIdx, err := n.snapshotProvider()
	if err != nil {
		return fmt.Errorf("raft: snapshot provider: %w", err)
	}
	if appliedIdx == 0 {
		return fmt.Errorf("raft: nothing to compact")
	}

	// Validate the reported index against the node's current view
	// before we commit the snapshot to disk. Under snapMu no other
	// writer can move snapshotIndex/commitIndex between this check
	// and the persist below.
	n.mu.Lock()
	if appliedIdx <= n.snapshotIndex {
		n.mu.Unlock()
		return fmt.Errorf("raft: provider index %d not above current snapshotIndex %d", appliedIdx, n.snapshotIndex)
	}
	if appliedIdx > n.commitIndex {
		n.mu.Unlock()
		return fmt.Errorf("raft: provider index %d above commitIndex %d", appliedIdx, n.commitIndex)
	}
	lastTerm, ok := n.termAt(appliedIdx)
	if !ok {
		n.mu.Unlock()
		return fmt.Errorf("raft: provider index %d outside log view [offset=%d, len=%d]", appliedIdx, n.logOffset, len(n.log))
	}
	n.mu.Unlock()

	snap := Snapshot{
		Meta: SnapshotMeta{
			LastIncludedIndex: appliedIdx,
			LastIncludedTerm:  lastTerm,
			Size:              int64(len(data)),
		},
		Data: data,
	}
	if err := SaveSnapshot(n.stateDir, snap); err != nil {
		return err
	}

	n.mu.Lock()
	// snapMu excludes every other snapshot writer, so this check
	// should always hold after a successful SaveSnapshot. The guard
	// remains as a defensive assertion: if some future change
	// introduces another writer that bypasses snapMu, we must not
	// roll raft.snapshot back in memory.
	if appliedIdx <= n.snapshotIndex {
		n.mu.Unlock()
		log.Info("Raft compaction superseded by concurrent install",
			"provider_index", appliedIdx,
			"snapshot_index", n.snapshotIndex,
		)
		return nil
	}
	// Drop the in-memory prefix covered by the snapshot. logOffset
	// becomes appliedIdx so Index/offset arithmetic stays consistent.
	if appliedIdx > n.logOffset {
		drop := len(n.log)
		if slot, ok := offsetToSlot(appliedIdx+1, n.logOffset, len(n.log)); ok {
			drop = slot
		}
		// Preserve the post-snapshot tail in a fresh backing slice so
		// the old array with compacted entries becomes eligible for GC.
		rest := append([]LogEntry(nil), n.log[drop:]...)
		n.log = rest
	}
	n.logOffset = appliedIdx
	n.snapshotIndex = appliedIdx
	n.snapshotTerm = lastTerm
	n.mu.Unlock()

	if n.wal != nil {
		if err := n.wal.TruncateBefore(appliedIdx + 1); err != nil {
			return fmt.Errorf("raft: compact wal: %w", err)
		}
	}

	log.Info("Raft log compacted",
		"snapshot_index", appliedIdx,
		"snapshot_term", lastTerm,
		"snapshot_bytes", len(data),
	)
	return nil
}
