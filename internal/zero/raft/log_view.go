/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package raft

// Log accessors that translate between absolute 1-based Raft log indices and
// the in-memory slice, which may have a logOffset due to compaction.
//
// All accessors require n.mu to be held in the appropriate mode by the
// caller. They do not lock internally: the Raft state machine takes the
// lock at higher granularity and holds it across multiple related reads
// (e.g. checking consistency, then appending).

// logLen returns the logical log length, i.e. the absolute index of the
// last entry known to this node (including compacted ones). Returns 0
// when the log has no entries at all.
func (n *Node) logLen() uint64 {
	return n.logOffset + uint64(len(n.log))
}

// sliceIndex maps an absolute Raft log index to the offset inside n.log.
// Returns (-1, false) when the index is outside the in-memory window: either
// below the compaction boundary (already in a snapshot) or past the tail.
// Valid absolute indices are in [logOffset+1, logOffset+len(log)].
func (n *Node) sliceIndex(absoluteIndex uint64) (int, bool) {
	return offsetToSlot(absoluteIndex, n.logOffset, len(n.log))
}

// entryAt returns the log entry at the given absolute index. Returns false
// when the entry is not available in memory (covered by snapshot or past
// the tail); callers must handle the snapshot case via snapshotIndex/Term.
func (n *Node) entryAt(absoluteIndex uint64) (LogEntry, bool) {
	slot, ok := n.sliceIndex(absoluteIndex)
	if !ok {
		return LogEntry{}, false
	}
	return n.log[slot], true
}

// termAt returns the term of the entry at absoluteIndex, resolving through
// the snapshot metadata when the entry has been compacted. Returns (0, true)
// when absoluteIndex == 0 (the "before any entry" sentinel used by
// AppendEntries for the very first record in the log).
func (n *Node) termAt(absoluteIndex uint64) (uint64, bool) {
	if absoluteIndex == 0 {
		return 0, true
	}
	if absoluteIndex == n.snapshotIndex {
		return n.snapshotTerm, true
	}
	e, ok := n.entryAt(absoluteIndex)
	if !ok {
		return 0, false
	}
	return e.Term, true
}

// lastLogIndexTerm returns the (index, term) of the highest known entry,
// consulting the snapshot when the in-memory slice is empty after
// compaction. Used by election eligibility checks.
func (n *Node) lastLogIndexTerm() (uint64, uint64) {
	if len(n.log) == 0 {
		return n.snapshotIndex, n.snapshotTerm
	}
	last := n.log[len(n.log)-1]
	return last.Index, last.Term
}

// truncateSliceAfter drops in-memory entries with Index > lastKeepIndex.
// A negative argument is a programmer error; callers pass absolute indices
// already validated against logOffset.
// Caller must hold n.mu for writes.
func (n *Node) truncateSliceAfter(lastKeepIndex uint64) {
	if lastKeepIndex <= n.logOffset {
		// Everything in memory would be dropped.
		n.log = n.log[:0]
		return
	}
	// Route the uint64 -> int narrowing through offsetToSlot so the
	// conversion happens at exactly one site in the package. The
	// "end-exclusive" semantics of slice[:n] map to "slot for
	// lastKeepIndex + 1" returned as the keep count.
	slot, ok := offsetToSlot(lastKeepIndex+1, n.logOffset, len(n.log))
	if !ok {
		return
	}
	n.log = n.log[:slot]
}
