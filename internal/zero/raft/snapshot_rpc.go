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
//  3. Accumulate req.Data into pendingSnapshot (chunking). A chunk with
//     req.Offset == 0 resets the buffer so a retry or a new leader
//     snapshots cleanly. A chunk whose (LastIncludedIndex, Term) tuple
//     differs from the current buffered transfer also resets.
//  4. When req.Done == true the accumulated buffer is installed:
//     invoke the registered SnapshotInstaller, persist to disk,
//     truncate any local log entries whose Index <= LastIncludedIndex,
//     and advance commitIndex / lastApplied.
//
// Success flag contract:
//   - Success=true only when the chunk was accepted (non-Done) or
//     fully installed and persisted (Done).
//   - Success=false on stale term, offset gap, installer error, or
//     snapshot save error. The leader MUST treat Success=false as a
//     rejection and NOT advance nextIndex/matchIndex; it should retry
//     from offset 0.
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
	n.leaderAddr = req.LeaderAddress
	n.lastContact = time.Now()

	// Reset buffer on Offset==0 OR when the leader has moved to a
	// newer snapshot while we were mid-stream.
	newTransfer := req.Offset == 0 ||
		req.LastIncludedIndex != n.pendingSnapshotIndex ||
		req.LastIncludedTerm != n.pendingSnapshotTerm
	if newTransfer {
		n.pendingSnapshot = n.pendingSnapshot[:0]
		n.pendingSnapshotIndex = req.LastIncludedIndex
		n.pendingSnapshotTerm = req.LastIncludedTerm
		n.pendingSnapshotExpected = 0
	}

	// Enforce monotonically-advancing offset. Any gap or backtrack is
	// treated as a retransmit need; we discard the partial buffer and
	// surface our current (fresh) term so the leader resets at 0.
	if req.Offset != n.pendingSnapshotExpected {
		n.pendingSnapshot = n.pendingSnapshot[:0]
		n.pendingSnapshotExpected = 0
		response.Term = n.currentTerm
		response.BytesStored = 0
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	n.pendingSnapshot = append(n.pendingSnapshot, req.Data...)
	n.pendingSnapshotExpected += uint64(len(req.Data))

	// Mid-stream: ACK with Success=true so the leader continues sending
	// the next chunk.
	if !req.Done {
		response.Term = n.currentTerm
		response.Success = true
		response.BytesStored = n.pendingSnapshotExpected
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}

	// Final chunk: capture what we need, then drop n.mu to run the
	// installer callback (which may take its own locks).
	data := append([]byte(nil), n.pendingSnapshot...)
	n.pendingSnapshot = n.pendingSnapshot[:0]
	n.pendingSnapshotExpected = 0
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
			n.mu.RLock()
			response.Term = n.currentTerm
			response.Success = false
			n.mu.RUnlock()
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
			n.mu.RLock()
			response.Term = n.currentTerm
			response.Success = false
			n.mu.RUnlock()
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
	response.Term = n.currentTerm
	response.Success = true
	response.BytesStored = uint64(len(data))
	n.mu.Unlock()

	if n.wal != nil {
		_ = n.wal.TruncateBefore(req.LastIncludedIndex + 1)
	}
	return n.wrapResponse(MsgInstallSnapshotRes, response)
}

// The leader-side InstallSnapshot flow (SendInstallSnapshot +
// exchangeSnapshotChunk + snapshotChunkSize + ErrSnapshotRejected)
// lives in snapshot_send.go to keep this file focused on the
// follower-side handler.
