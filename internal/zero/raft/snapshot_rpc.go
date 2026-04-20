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
//  3. Install the snapshot via the registered SnapshotInstaller if present.
//  4. Truncate any local log entries whose Index is <= LastIncludedIndex
//     (they are subsumed by the snapshot).
//  5. Persist the snapshot to disk so a restart does not require another
//     InstallSnapshot.
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
	installer := n.snapshotInstaller
	stateDir := n.stateDir
	n.mu.Unlock()

	// Apply to state machine first; if the installer rejects, do not mutate
	// log/commit state. Running installer without holding n.mu lets
	// implementations freely take their own locks.
	if installer != nil {
		if err := installer(req.Data, req.LastIncludedIndex, req.LastIncludedTerm); err != nil {
			n.mu.RLock()
			response.Term = n.currentTerm
			n.mu.RUnlock()
			return n.wrapResponse(MsgInstallSnapshotRes, response)
		}
	}

	// Persist the snapshot to disk (best-effort; log the failure but do not
	// fail the RPC, the leader can retry if we never catch up).
	if stateDir != "" {
		snap := Snapshot{
			Meta: SnapshotMeta{
				LastIncludedIndex: req.LastIncludedIndex,
				LastIncludedTerm:  req.LastIncludedTerm,
				Size:              int64(len(req.Data)),
			},
			Data: req.Data,
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
	// Drop log entries fully covered by the snapshot; preserve any tail
	// whose Index > LastIncludedIndex to avoid re-fetching already-replicated
	// work from the leader. Advance logOffset so the invariant
	// n.log[i].Index == n.logOffset + i + 1 holds after the rewrite.
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

	// Reflect compaction on disk too.
	if n.wal != nil {
		_ = n.wal.TruncateBefore(req.LastIncludedIndex + 1)
	}
	return n.wrapResponse(MsgInstallSnapshotRes, response)
}

// SendInstallSnapshot is a leader-side helper that pushes the current
// on-disk snapshot to a lagging peer. Returns early when there is no
// snapshot on disk or the node has no state directory.
func (n *Node) SendInstallSnapshot(ctx context.Context, peer string) error {
	if n.stateDir == "" {
		return fmt.Errorf("raft: no state dir")
	}
	snap, err := LoadSnapshot(n.stateDir)
	if err != nil {
		return fmt.Errorf("raft: load snapshot: %w", err)
	}

	n.mu.RLock()
	req := InstallSnapshotRequest{
		Term:              n.currentTerm,
		LeaderID:          n.config.NodeID,
		LastIncludedIndex: snap.Meta.LastIncludedIndex,
		LastIncludedTerm:  snap.Meta.LastIncludedTerm,
		Data:              snap.Data,
	}
	n.mu.RUnlock()

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
	} else {
		// Peer caught up to our snapshot; advance nextIndex/matchIndex.
		n.nextIndex[peer] = snap.Meta.LastIncludedIndex + 1
		n.matchIndex[peer] = snap.Meta.LastIncludedIndex
	}
	n.mu.Unlock()
	return nil
}
