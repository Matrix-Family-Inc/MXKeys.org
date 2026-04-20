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
	"context"
	"encoding/json"
	"fmt"
	"time"
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
// Non-terminal chunks simply acknowledge reception via the normal
// response. Out-of-order offsets cause an immediate reset on the
// follower; the leader will retry from offset 0.
func (n *Node) handleInstallSnapshot(msg *RPCMessage) *RPCMessage {
	var req InstallSnapshotRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		n.mu.RLock()
		term := n.currentTerm
		n.mu.RUnlock()
		return n.wrapResponse(MsgInstallSnapshotRes, InstallSnapshotResponse{Term: term})
	}

	n.mu.Lock()
	response := InstallSnapshotResponse{Term: n.currentTerm}

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
		n.mu.Unlock()
		return n.wrapResponse(MsgInstallSnapshotRes, response)
	}
	n.pendingSnapshot = append(n.pendingSnapshot, req.Data...)
	n.pendingSnapshotExpected += uint64(len(req.Data))

	// Mid-stream: just ACK, keep buffering.
	if !req.Done {
		response.Term = n.currentTerm
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
			n.mu.RLock()
			response.Term = n.currentTerm
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
		_ = SaveSnapshot(stateDir, snap)
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
	n.mu.Unlock()

	if n.wal != nil {
		_ = n.wal.TruncateBefore(req.LastIncludedIndex + 1)
	}
	return n.wrapResponse(MsgInstallSnapshotRes, response)
}

// snapshotChunkSize is the maximum payload size per InstallSnapshot
// RPC. 512 KiB keeps each RPC well inside the 8 MiB walMaxRecord
// budget, leaves headroom for JSON framing, and limits the amount of
// data that needs to be retransmitted after a network hiccup.
const snapshotChunkSize = 512 * 1024

// SendInstallSnapshot is a leader-side helper that pushes the current
// on-disk snapshot to a lagging peer. The snapshot is streamed in
// snapshotChunkSize-byte chunks; each chunk carries the full
// (LastIncludedIndex, LastIncludedTerm) tuple so the follower can
// detect leader changes mid-stream. Only the final chunk sets
// Done=true; the follower applies the accumulated buffer only after
// seeing Done.
//
// Returns early when there is no snapshot on disk or the node has no
// state directory. Returns on the first RPC failure; the replication
// loop will retry.
func (n *Node) SendInstallSnapshot(ctx context.Context, peer string) error {
	if n.stateDir == "" {
		return fmt.Errorf("raft: no state dir")
	}
	snap, err := LoadSnapshot(n.stateDir)
	if err != nil {
		return fmt.Errorf("raft: load snapshot: %w", err)
	}

	n.mu.RLock()
	term := n.currentTerm
	leaderID := n.config.NodeID
	n.mu.RUnlock()

	data := []byte(snap.Data)
	total := uint64(len(data))

	// Edge case: empty snapshot. Still emit a single Done=true RPC so
	// the follower's state machine can reset cleanly.
	if total == 0 {
		req := InstallSnapshotRequest{
			Term:              term,
			LeaderID:          leaderID,
			LastIncludedIndex: snap.Meta.LastIncludedIndex,
			LastIncludedTerm:  snap.Meta.LastIncludedTerm,
			Offset:            0,
			Done:              true,
			Data:              nil,
		}
		if err := n.exchangeSnapshotChunk(peer, &req); err != nil {
			return err
		}
	} else {
		var offset uint64
		for offset < total {
			if err := ctx.Err(); err != nil {
				return err
			}
			end := offset + snapshotChunkSize
			if end > total {
				end = total
			}
			req := InstallSnapshotRequest{
				Term:              term,
				LeaderID:          leaderID,
				LastIncludedIndex: snap.Meta.LastIncludedIndex,
				LastIncludedTerm:  snap.Meta.LastIncludedTerm,
				Offset:            offset,
				Done:              end == total,
				Data:              append([]byte(nil), data[offset:end]...),
			}
			if err := n.exchangeSnapshotChunk(peer, &req); err != nil {
				return err
			}
			offset = end
		}
	}

	// Peer acknowledged the last chunk; advance bookkeeping.
	n.mu.Lock()
	n.nextIndex[peer] = snap.Meta.LastIncludedIndex + 1
	n.matchIndex[peer] = snap.Meta.LastIncludedIndex
	n.mu.Unlock()
	return nil
}

// exchangeSnapshotChunk sends a single chunk and processes the reply's
// term field; the caller is the overall SendInstallSnapshot loop.
func (n *Node) exchangeSnapshotChunk(peer string, req *InstallSnapshotRequest) error {
	resp, err := n.sendRPC(peer, MsgInstallSnapshot, req)
	if err != nil {
		return err
	}
	var out InstallSnapshotResponse
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		return fmt.Errorf("raft: decode install snapshot response: %w", err)
	}
	n.mu.Lock()
	if out.Term > n.currentTerm {
		n.currentTerm = out.Term
		n.state = Follower
		n.votedFor = ""
	}
	n.mu.Unlock()
	return nil
}
