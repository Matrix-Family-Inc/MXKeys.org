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
	"fmt"
	"time"
)

const (
	MsgForwardProposal    MessageType = "forward_proposal"
	MsgForwardProposalRes MessageType = "forward_proposal_response"
)

// ForwardProposalRequest is sent by a follower to the current leader
// to propose a command for replication on behalf of a client that
// happened to hit the follower. The leader decodes Command, calls
// Submit locally, and returns the outcome in ForwardProposalResponse.
// Authentication reuses the SharedSecret-signed RPC envelope used by
// every other Raft RPC; any peer that can sign Raft traffic can also
// forward.
type ForwardProposalRequest struct {
	Command []byte `json:"command"`
}

// ForwardProposalResponse carries the leader's verdict on a
// forwarded proposal. Error is empty on success and otherwise
// contains the Submit error message (ErrNotLeader, ErrTimeout,
// persist failure, etc.). Wire-level transport errors are surfaced
// to the caller as sendRPC errors, not via this field.
type ForwardProposalResponse struct {
	Error string `json:"error,omitempty"`
}

// Propose submits a command for replication, transparently forwarding
// to the current leader when this node is a follower.
//
// Behaviour:
//
//   - Leader: equivalent to Submit. Returns Submit's error on failure.
//   - Follower with a known leader address (learned from
//     AppendEntries / InstallSnapshot): sends a MsgForwardProposal
//     RPC to the leader and returns the leader's verdict. Transport
//     errors are wrapped; a non-empty Error in the response is
//     returned as a descriptive error.
//   - Follower with no known leader: returns ErrNoLeader immediately.
//
// This is the right entry point for application-level writes in raft
// mode. Submit alone would leave follower-originated writes invisible
// to the cluster because non-leader callers get ErrNotLeader with no
// fallback.
func (n *Node) Propose(ctx context.Context, command []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	n.mu.RLock()
	isLeader := n.state == Leader
	leaderAddr := n.leaderAddr
	n.mu.RUnlock()

	if isLeader {
		return n.Submit(ctx, command)
	}
	if leaderAddr == "" {
		return ErrNoLeader
	}

	req := ForwardProposalRequest{Command: append([]byte(nil), command...)}
	// ctx is threaded into sendRPCCtx so Cluster.Stop() (via
	// proposeCtx → stopCh cancel) or any other caller-initiated
	// cancel tears the forwarded RPC down immediately instead of
	// waiting out the net.Conn deadline.
	resp, err := n.sendRPCCtx(ctx, leaderAddr, MsgForwardProposal, req)
	if err != nil {
		return fmt.Errorf("raft: forward proposal to %s: %w", leaderAddr, err)
	}
	var out ForwardProposalResponse
	if err := json.Unmarshal(resp.Payload, &out); err != nil {
		return fmt.Errorf("raft: decode forward proposal response: %w", err)
	}
	if out.Error != "" {
		return fmt.Errorf("raft: leader rejected forwarded proposal: %s", out.Error)
	}
	return nil
}

// handleForwardProposal is the leader-side handler for
// MsgForwardProposal. A non-leader that receives this RPC returns
// ErrNotLeader in the response Error field so the follower caller
// can surface a clear failure without silently dropping the write.
//
// Timeout: the leader bounds Submit by CommitTimeout (with a 5 s
// floor). The follower's own context deadline is not carried over
// the wire; a follower that needs its own deadline should cancel
// the caller's context and it will propagate via the RPC read path
// (sendRPC dial timeout plus the inbound handler's Submit timeout).
func (n *Node) handleForwardProposal(msg *RPCMessage) *RPCMessage {
	var req ForwardProposalRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return n.wrapResponse(MsgForwardProposalRes, ForwardProposalResponse{
			Error: fmt.Sprintf("decode: %v", err),
		})
	}
	if len(req.Command) == 0 {
		return n.wrapResponse(MsgForwardProposalRes, ForwardProposalResponse{
			Error: "empty command",
		})
	}

	n.mu.RLock()
	isLeader := n.state == Leader
	n.mu.RUnlock()
	if !isLeader {
		return n.wrapResponse(MsgForwardProposalRes, ForwardProposalResponse{
			Error: ErrNotLeader.Error(),
		})
	}

	timeout := n.config.CommitTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	// ctxWithStop tears the pending Submit down the moment
	// Node.Stop() is invoked on the leader, instead of letting the
	// forwarded proposal burn through the full CommitTimeout on
	// shutdown.
	ctx, cancel := n.ctxWithStop(context.Background(), timeout)
	defer cancel()

	if err := n.Submit(ctx, req.Command); err != nil {
		return n.wrapResponse(MsgForwardProposalRes, ForwardProposalResponse{
			Error: err.Error(),
		})
	}
	return n.wrapResponse(MsgForwardProposalRes, ForwardProposalResponse{})
}
