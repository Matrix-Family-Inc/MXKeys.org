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
	"testing"
	"time"
)

// TestProposeRespectsCancelledContextImmediately locks in the
// contract that Propose terminates on ctx cancellation before
// touching the network. Without this guard a caller who signalled
// shutdown via the context would still wait out the forwarded RPC's
// TCP deadline.
func TestProposeRespectsCancelledContextImmediately(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "f",
		ElectionTimeout: 200 * time.Millisecond,
	})
	n.leaderAddr = "127.0.0.1:0" // dialable-looking but never used
	n.state = Follower

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := n.Propose(ctx, []byte("payload"))
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled on already-cancelled ctx, got %v", err)
	}
	// The connection default deadline is seconds; a terminating
	// Propose should return effectively instantly.
	if elapsed > 100*time.Millisecond {
		t.Fatalf("Propose took %v; must return immediately on cancelled ctx", elapsed)
	}
}

// TestProposeOnLeaderReturnsNoLeaderErrorWhenNotLeaderAndNoAddr
// verifies the other terminal branch: a non-leader with no known
// leader address must fail fast with ErrNoLeader rather than
// attempting to dial or returning a wrapped context error.
func TestProposeReturnsErrNoLeaderWhenAddrMissing(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "f",
		ElectionTimeout: 200 * time.Millisecond,
	})
	n.state = Follower
	n.leaderAddr = ""

	err := n.Propose(context.Background(), []byte("payload"))
	if !errors.Is(err, ErrNoLeader) {
		t.Fatalf("expected ErrNoLeader, got %v", err)
	}
}

// TestHandleAppendEntriesDoesNotClobberLeaderAddrWithEmpty asserts
// that a well-formed AppendEntries from a leader that did not
// populate LeaderAddress (e.g. a mixed-version peer) never strips
// the follower's known forwarding endpoint. Previously this was a
// latent Propose-returns-ErrNoLeader trap.
func TestHandleAppendEntriesDoesNotClobberLeaderAddrWithEmpty(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "f",
		ElectionTimeout: 200 * time.Millisecond,
	})
	n.currentTerm = 1
	n.leaderAddr = "10.0.0.5:7000"

	req := AppendEntriesRequest{
		Term:          1,
		LeaderId:      "leader",
		LeaderAddress: "",
		PrevLogIndex:  0,
		PrevLogTerm:   0,
		Entries:       nil,
		LeaderCommit:  0,
	}
	payload, _ := json.Marshal(req)
	_ = n.handleAppendEntries(&RPCMessage{Type: MsgAppendEntries, Payload: payload})

	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.leaderAddr != "10.0.0.5:7000" {
		t.Fatalf("leaderAddr must be preserved on empty LeaderAddress, got %q", n.leaderAddr)
	}
	if n.leaderId != "leader" {
		t.Fatalf("leaderId should update even when address is absent, got %q", n.leaderId)
	}
}

// TestHandleAppendEntriesAcceptsUpdatedLeaderAddr asserts the
// symmetric case: when a leader arrives with a populated address,
// the follower must overwrite any stale value so forwarding targets
// a live endpoint.
func TestHandleAppendEntriesAcceptsUpdatedLeaderAddr(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "f",
		ElectionTimeout: 200 * time.Millisecond,
	})
	n.currentTerm = 1
	n.leaderAddr = "10.0.0.5:7000"

	req := AppendEntriesRequest{
		Term:          1,
		LeaderId:      "new-leader",
		LeaderAddress: "10.0.0.9:8000",
	}
	payload, _ := json.Marshal(req)
	_ = n.handleAppendEntries(&RPCMessage{Type: MsgAppendEntries, Payload: payload})

	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.leaderAddr != "10.0.0.9:8000" {
		t.Fatalf("leaderAddr must update when new LeaderAddress is non-empty, got %q", n.leaderAddr)
	}
}

// TestHandleInstallSnapshotDoesNotClobberLeaderAddrWithEmpty
// mirrors the AE test above for the InstallSnapshot path. Both
// paths populate leaderAddr from the incoming RPC and must share
// the "never overwrite with empty" rule.
func TestHandleInstallSnapshotDoesNotClobberLeaderAddrWithEmpty(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "f",
		ElectionTimeout: 200 * time.Millisecond,
	})
	n.currentTerm = 1
	n.leaderAddr = "10.0.0.5:7000"

	req := InstallSnapshotRequest{
		Term:              1,
		LeaderID:          "leader",
		LeaderAddress:     "",
		LastIncludedIndex: 1,
		LastIncludedTerm:  1,
		Offset:            0,
		Done:              true,
		Data:              nil,
	}
	payload, _ := json.Marshal(req)
	_ = n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})

	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.leaderAddr != "10.0.0.5:7000" {
		t.Fatalf("leaderAddr must be preserved on empty LeaderAddress in InstallSnapshot, got %q", n.leaderAddr)
	}
}
