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
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// snapshotChunkSize is the maximum payload size per InstallSnapshot
// RPC. 512 KiB keeps each RPC well inside the 8 MiB walMaxRecord
// budget, leaves headroom for JSON framing, and limits the amount of
// data that needs to be retransmitted after a network hiccup.
const snapshotChunkSize = 512 * 1024

// ErrSnapshotRejected indicates the follower refused (or failed to
// install) a chunk of an InstallSnapshot stream. The replication loop
// must treat this as "retry the whole transfer from offset 0 on the
// next pass" and MUST NOT advance nextIndex/matchIndex for the peer.
var ErrSnapshotRejected = errors.New("raft: follower rejected install_snapshot")

// SendInstallSnapshot is a leader-side helper that pushes the current
// on-disk snapshot to a lagging peer. The snapshot is streamed in
// snapshotChunkSize-byte chunks; each chunk carries the full
// (LastIncludedIndex, LastIncludedTerm) tuple so the follower can
// detect leader changes mid-stream. Only the final chunk sets
// Done=true; the follower applies the accumulated buffer only after
// seeing Done.
//
// Returns early when there is no snapshot on disk or the node has no
// state directory. Returns on the first RPC failure or a follower
// rejection (ErrSnapshotRejected); the replication loop will retry.
//
// Invariant: nextIndex/matchIndex for the peer are advanced if and
// only if the follower ACKed the Done chunk with Success=true. Every
// other outcome (network error, decode error, Success=false ACK)
// leaves the peer bookkeeping untouched so the next replication pass
// will either resend log entries from the current nextIndex or
// restart the snapshot from offset 0.
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

	leaderAddr := n.advertiseAddr()

	data := []byte(snap.Data)
	total := uint64(len(data))

	// Edge case: empty snapshot. Still emit a single Done=true RPC so
	// the follower's state machine can reset cleanly.
	if total == 0 {
		req := InstallSnapshotRequest{
			Term:              term,
			LeaderID:          leaderID,
			LeaderAddress:     leaderAddr,
			LastIncludedIndex: snap.Meta.LastIncludedIndex,
			LastIncludedTerm:  snap.Meta.LastIncludedTerm,
			Offset:            0,
			Done:              true,
			Data:              nil,
		}
		ok, err := n.exchangeSnapshotChunk(peer, &req)
		if err != nil {
			return err
		}
		if !ok {
			return ErrSnapshotRejected
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
				LeaderAddress:     leaderAddr,
				LastIncludedIndex: snap.Meta.LastIncludedIndex,
				LastIncludedTerm:  snap.Meta.LastIncludedTerm,
				Offset:            offset,
				Done:              end == total,
				Data:              append([]byte(nil), data[offset:end]...),
			}
			ok, err := n.exchangeSnapshotChunk(peer, &req)
			if err != nil {
				return err
			}
			if !ok {
				// Follower rejected this chunk (gap, installer failure,
				// save failure, or stale term). Abort the transfer; a
				// future replication pass will restart from offset 0.
				return ErrSnapshotRejected
			}
			offset = end
		}
	}

	// Peer acknowledged the Done chunk with Success=true. Only now do
	// we advance bookkeeping; prior to this point every return path
	// left nextIndex/matchIndex untouched.
	n.mu.Lock()
	n.nextIndex[peer] = snap.Meta.LastIncludedIndex + 1
	n.matchIndex[peer] = snap.Meta.LastIncludedIndex
	n.mu.Unlock()
	return nil
}

// exchangeSnapshotChunk sends a single chunk and processes the reply.
// Returns (ok, err): ok reflects the follower's Success flag; err is
// a transport/decoding error. A caller that sees ok=false on the
// final chunk must NOT treat the peer as having received the
// snapshot.
func (n *Node) exchangeSnapshotChunk(peer string, req *InstallSnapshotRequest) (bool, error) {
	resp, err := n.sendRPC(peer, MsgInstallSnapshot, req)
	if err != nil {
		return false, err
	}
	var out InstallSnapshotResponse
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		return false, fmt.Errorf("raft: decode install snapshot response: %w", err)
	}
	n.mu.Lock()
	if out.Term > n.currentTerm {
		n.currentTerm = out.Term
		n.state = Follower
		n.votedFor = ""
	}
	n.mu.Unlock()
	return out.Success, nil
}
