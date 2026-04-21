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
	"encoding/json"
	"time"

	"mxkeys/internal/zero/log"
)

// handleInstallSnapshot processes an InstallSnapshot RPC. Follower-side only.
//
// Semantics:
//
//  1. Reject when the leader's term is stale.
//  2. Refresh currentTerm and fall back to follower on higher leader term.
//  3. Accumulate req.Data into the active transfer (chunking). A chunk
//     with req.Offset == 0 resets the buffer so a retry or a new leader
//     snapshots cleanly. A chunk whose (LastIncludedIndex, Term) tuple
//     differs from the current buffered transfer also resets.
//  4. When req.Done == true the accumulated payload is installed:
//     invoke the registered SnapshotInstaller, persist to disk,
//     truncate any local log entries whose Index <= LastIncludedIndex,
//     and advance commitIndex / lastApplied.
//
// Memory bound: when stateDir != "" the in-flight payload is streamed
// to stateDir/raft.snapshot.recv so the follower holds only one
// chunk in memory at a time regardless of total snapshot size. See
// pending_snapshot.go for the spill lifecycle and ErrPendingSnapshotOverflow
// for the maxSnapshotSize cap.
//
// Success flag contract:
//   - Success=true only when the chunk was accepted (non-Done) or
//     fully installed and persisted (Done).
//   - Success=false on stale term, offset gap, overflow, installer
//     error, or snapshot save error. The leader MUST treat
//     Success=false as a rejection and NOT advance
//     nextIndex/matchIndex; it should retry from offset 0.
func (n *Node) handleInstallSnapshot(msg *RPCMessage) *RPCMessage {
	var req InstallSnapshotRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		n.mu.RLock()
		term := n.currentTerm
		n.mu.RUnlock()
		return n.wrapResponse(MsgInstallSnapshotRes, InstallSnapshotResponse{
			Term:    term,
			Success: false,
		})
	}

	n.mu.Lock()
	response := InstallSnapshotResponse{Term: n.currentTerm, Success: false}

	if req.Term < n.currentTerm {
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.votedFor = ""
	}
	n.state = Follower
	n.leaderId = req.LeaderID
	// Never overwrite a known leaderAddr with an empty one. See the
	// matching comment in handleAppendEntries for the rationale.
	if req.LeaderAddress != "" {
		n.leaderAddr = req.LeaderAddress
	}
	n.lastContact = time.Now()

	// Reset the transfer on Offset==0 OR when the leader has moved to
	// a newer snapshot while we were mid-stream. beginPendingSnapshot
	// closes and truncates the spill file so no stale bytes carry
	// over.
	newTransfer := req.Offset == 0 ||
		req.LastIncludedIndex != n.pendingSnapshotIndex ||
		req.LastIncludedTerm != n.pendingSnapshotTerm
	if newTransfer {
		if err := n.beginPendingSnapshot(req.LastIncludedIndex, req.LastIncludedTerm); err != nil {
			log.Warn("Raft begin pending snapshot failed",
				"last_index", req.LastIncludedIndex,
				"error", err,
			)
			response.Term = n.currentTerm
			n.mu.Unlock()
			return n.wrapResponse(MsgInstallSnapshotRes, response)
		}
	}

	// Enforce monotonically-advancing offset. Any gap or backtrack is
	// treated as a retransmit need; we discard the partial transfer
	// and surface our current (fresh) term so the leader resets at 0.
	if req.Offset != n.pendingSnapshotExpected {
		n.resetPendingSnapshot()
		response.Term = n.currentTerm
		response.BytesStored = 0
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	if err := n.appendPendingSnapshot(req.Data); err != nil {
		// Overflow past maxSnapshotSize or a spill-file write failure.
		// Either way the transfer is unrecoverable; reset and reject.
		log.Warn("Raft pending snapshot append failed",
			"last_index", req.LastIncludedIndex,
			"error", err,
		)
		n.resetPendingSnapshot()
		response.Term = n.currentTerm
		response.BytesStored = 0
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}

	// Mid-stream: ACK with Success=true so the leader continues sending
	// the next chunk.
	if !req.Done {
		response.Term = n.currentTerm
		response.Success = true
		response.BytesStored = n.pendingSnapshotExpected
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}

	// Final chunk: drain the transfer into a transient buffer, then
	// drop n.mu to run the installer callback (which may take its
	// own locks).
	data, err := n.drainPendingSnapshot()
	if err != nil {
		log.Warn("Raft drain pending snapshot failed",
			"last_index", req.LastIncludedIndex,
			"error", err,
		)
		n.resetPendingSnapshot()
		response.Term = n.currentTerm
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	installer := n.snapshotInstaller
	stateDir := n.stateDir
	n.mu.Unlock()

	if installer != nil {
		if err := installer(data, req.LastIncludedIndex, req.LastIncludedTerm); err != nil {
			log.Warn("Raft snapshot installer rejected payload",
				"last_index", req.LastIncludedIndex,
				"last_term", req.LastIncludedTerm,
				"bytes", len(data),
				"error", err,
			)
			n.mu.Lock()
			n.resetPendingSnapshot()
			response.Term = n.currentTerm
			n.mu.Unlock()
			return n.wrapResponse(MsgInstallSnapshotRes, response)
		}
	}

	if stateDir != "" {
		snap := Snapshot{
			Meta: SnapshotMeta{
				LastIncludedIndex: req.LastIncludedIndex,
				LastIncludedTerm:  req.LastIncludedTerm,
				Size:              int64(len(data)),
			},
			Data: data,
		}
		if err := SaveSnapshot(stateDir, snap); err != nil {
			// Persist failure means we cannot guarantee the snapshot will
			// survive a crash; reject so the leader retries rather than
			// advancing match/next on a best-effort install.
			log.Warn("Raft snapshot save failed", "error", err)
			n.mu.Lock()
			n.resetPendingSnapshot()
			response.Term = n.currentTerm
			n.mu.Unlock()
			return n.wrapResponse(MsgInstallSnapshotRes, response)
		}
	}

	n.mu.Lock()
	n.snapshotIndex = req.LastIncludedIndex
	n.snapshotTerm = req.LastIncludedTerm
	if req.LastIncludedIndex > n.commitIndex {
		n.commitIndex = req.LastIncludedIndex
	}
	if req.LastIncludedIndex > n.lastApplied {
		n.lastApplied = req.LastIncludedIndex
	}
	var keep []LogEntry
	for _, e := range n.log {
		if e.Index > req.LastIncludedIndex {
			keep = append(keep, e)
		}
	}
	n.log = keep
	if req.LastIncludedIndex > n.logOffset {
		n.logOffset = req.LastIncludedIndex
	}
	// Successful install: drop the spill file now that raft.snapshot
	// is the authoritative copy.
	n.resetPendingSnapshot()
	response.Term = n.currentTerm
	response.Success = true
	response.BytesStored = uint64(len(data))
	n.mu.Unlock()

	if n.wal != nil {
		// WAL prefix truncation is bookkeeping: the snapshot
		// supersedes every entry up to LastIncludedIndex so any
		// left-behind WAL record below that boundary is redundant
		// but not incorrect (LoadFromDisk skips Index <=
		// snapshotIndex during replay). A failure here does NOT
		// invalidate the install we just persisted; log at Warn so
		// disk pressure or permission issues are visible to
		// operators rather than silenced.
		if err := n.wal.TruncateBefore(req.LastIncludedIndex + 1); err != nil {
			log.Warn("Raft WAL truncate after InstallSnapshot failed",
				"snapshot_index", req.LastIncludedIndex,
				"error", err,
			)
		}
	}
	return n.wrapResponse(MsgInstallSnapshotRes, response)
}

// The leader-side InstallSnapshot flow (SendInstallSnapshot +
// exchangeSnapshotChunk + snapshotChunkSize + ErrSnapshotRejected)
// lives in snapshot_send.go to keep this file focused on the
// follower-side handler.
