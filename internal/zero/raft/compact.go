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
	"fmt"

	"mxkeys/internal/zero/log"
)

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
