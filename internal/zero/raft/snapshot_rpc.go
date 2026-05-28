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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"

	"mxkeys/internal/zero/log"
)

// handleInstallSnapshot processes an InstallSnapshot RPC. Follower-side.
//
// Flow: term check; tuple reset on Offset==0 or tuple mismatch;
// append chunk to spill file (disk) or in-memory buffer; on
// Done=true finalise header, run streaming installer over data
// portion, atomic-rename spill into raft.snapshot, advance
// snapshotIndex / commitIndex / lastApplied.
//
// Memory: stateDir != "" streams to raft.snapshot.recv so peak
// RAM is O(snapshotChunkSize) both on accumulation and install.
//
// Concurrency: n.snapMu is held for the whole handler, so no
// parallel handleInstallSnapshot or CompactLog can mutate the
// spill, raft.snapshot, or snapshotIndex/term while we run.
//
// Success flag contract:
//   - Success=true: non-Done chunk accepted, or Done chunk fully
//     installed and renamed into place.
//   - Success=false: stale term, missing installer, offset gap,
//     overflow, installer error, finalise error, rename error.
//     The leader MUST NOT advance nextIndex/matchIndex on
//     Success=false; it retries from offset 0.
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

	// Serialise with CompactLog and other InstallSnapshot handlers
	// for the full duration. No concurrent writer to raft.snapshot,
	// the spill file, or the snapshotIndex/term memory slot can
	// interleave while we hold snapMu.
	n.snapMu.Lock()
	defer n.snapMu.Unlock()

	n.mu.Lock()
	response := InstallSnapshotResponse{Term: n.currentTerm, Success: false}

	if req.Term < n.currentTerm {
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}

	// Reject outright without an installer: advancing
	// snapshotIndex without applying the payload would desync the
	// state machine. Operator must register SnapshotInstaller.
	if n.snapshotInstaller == nil {
		n.mu.Unlock()
		log.Warn("Raft InstallSnapshot rejected: no SnapshotInstaller registered",
			"last_index", req.LastIncludedIndex, "last_term", req.LastIncludedTerm)
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.votedFor = ""
	}
	n.state = Follower
	n.leaderId = req.LeaderID
	// Never overwrite a known leaderAddr with an empty one; see
	// handleAppendEntries for the rationale.
	if req.LeaderAddress != "" {
		n.leaderAddr = req.LeaderAddress
	}
	n.lastContact = time.Now()

	// Monotonicity: never roll snapshotIndex backwards. A
	// duplicate retry or straggler from a stale leader is ACKed
	// idempotently with Success=true without running the installer
	// or touching the spill file.
	if req.LastIncludedIndex <= n.snapshotIndex && n.snapshotIndex > 0 {
		response.Term = n.currentTerm
		response.Success = true
		response.BytesStored = 0
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}

	// Reset the transfer on Offset==0 OR when the leader has moved to
	// a newer snapshot while we were mid-stream. beginPendingSnapshot
	// closes and truncates the spill file and writes a fresh header
	// so no stale bytes carry over.
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
		n.releasePendingSnapshot()
		response.Term = n.currentTerm
		response.BytesStored = 0
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	if err := n.appendPendingSnapshot(req.Data); err != nil {
		// Overflow past maxSnapshotSize or a spill-file write failure.
		// Either way the transfer is unrecoverable; release and reject.
		log.Warn("Raft pending snapshot append failed",
			"last_index", req.LastIncludedIndex,
			"error", err,
		)
		n.releasePendingSnapshot()
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

	// Final chunk: finalise the spill file (patch header, fsync,
	// seek to data), run the installer against it, and on success
	// atomically rename the spill into raft.snapshot. Everything
	// below runs outside n.mu so the installer is free to take the
	// application's own locks; snapMu still excludes other snapshot
	// writers.
	installer := n.snapshotInstaller
	stateDir := n.stateDir
	var (
		f       *os.File
		path    string
		size    int64
		memData []byte
	)
	if stateDir != "" {
		var ferr error
		f, path, size, ferr = n.finalizePendingSnapshot()
		if ferr != nil {
			log.Warn("Raft finalize pending snapshot failed",
				"last_index", req.LastIncludedIndex,
				"error", ferr,
			)
			response.Term = n.currentTerm
			n.mu.Unlock()
			return n.wrapResponse(MsgInstallSnapshotRes, response)
		}
	} else {
		memData = n.drainPendingSnapshotInMemory()
		size = int64(len(memData))
	}
	n.mu.Unlock()

	// Build the installer reader. CRC integrity is already
	// established by finalizePendingSnapshot; the counting wrap
	// exists only to emit an advisory short-read warning below.
	var baseReader io.Reader
	if f != nil {
		baseReader = f
	} else {
		baseReader = bytes.NewReader(memData)
	}
	cr := &countingReader{r: baseReader}

	if installer != nil {
		if err := installer(cr, size, req.LastIncludedIndex, req.LastIncludedTerm); err != nil {
			log.Warn("Raft snapshot installer rejected payload",
				"last_index", req.LastIncludedIndex,
				"last_term", req.LastIncludedTerm,
				"bytes", size,
				"error", err,
			)
			if f != nil {
				_ = f.Close()
				_ = os.Remove(path)
			}
			n.mu.Lock()
			response.Term = n.currentTerm
			n.mu.Unlock()
			return n.wrapResponse(MsgInstallSnapshotRes, response)
		}
	}

	// Short read violates the SnapshotInstaller contract. Reject
	// with Success=false and drop the spill; the idempotent
	// installer will converge (or surface its bug) on retry.
	if cr.count != size {
		log.Warn("Raft install handler: installer did not drain full snapshot",
			"read", cr.count, "expected", size)
		if f != nil {
			_ = f.Close()
			_ = os.Remove(path)
		}
		n.mu.Lock()
		response.Term = n.currentTerm
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}

	// For the disk-backed path the spill file IS the snapshot in
	// its final byte layout; just close the handle and atomically
	// rename it into place. That single rename replaces the
	// previous SaveSnapshot pass, so the peak memory of this whole
	// critical section is O(chunk) + the installer's own decoding.
	if f != nil {
		_ = f.Close()
		if err := finalizeAndRenamePendingSnapshot(stateDir, path); err != nil {
			log.Warn("Raft snapshot rename failed", "error", err)
			n.mu.Lock()
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
	response.Term = n.currentTerm
	response.Success = true
	response.BytesStored = uint64(size)
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
