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
	"testing"
	"time"
)

// TestHandlePreVoteDoesNotMutateState is the core property of pre-vote:
// an RPC round-trip must leave currentTerm, votedFor, and state
// untouched, so that a partitioned or misbehaving peer's probe cannot
// disrupt a healthy cluster.
func TestHandlePreVoteDoesNotMutateState(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "n1",
		ElectionTimeout: 300 * time.Millisecond,
	})
	n.currentTerm = 5
	n.votedFor = "someone"
	n.state = Follower
	n.leaderId = "leader"
	n.lastContact = time.Now().Add(-1 * time.Hour) // long gap => grant path candidate

	payload, _ := json.Marshal(PreVoteRequest{
		Term:         10, // hypothetical higher term
		CandidateId:  "n2",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})

	_ = n.handlePreVote(&RPCMessage{Type: MsgPreVote, Payload: payload})

	if n.currentTerm != 5 {
		t.Errorf("currentTerm mutated: got %d, want 5", n.currentTerm)
	}
	if n.votedFor != "someone" {
		t.Errorf("votedFor mutated: got %q, want 'someone'", n.votedFor)
	}
	if n.state != Follower {
		t.Errorf("state mutated: got %v, want Follower", n.state)
	}
}

// TestHandlePreVoteRefusesWhileLeaderReachable enforces the anti-
// disruption property: a node that just heard from the leader must
// refuse to grant pre-votes, otherwise a flapping peer could still
// force real elections.
func TestHandlePreVoteRefusesWhileLeaderReachable(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "n1",
		ElectionTimeout: 300 * time.Millisecond,
	})
	n.currentTerm = 3
	n.leaderId = "leader1"
	n.lastContact = time.Now() // right now -> leader is fresh

	payload, _ := json.Marshal(PreVoteRequest{
		Term:         4,
		CandidateId:  "n2",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	resp := n.handlePreVote(&RPCMessage{Type: MsgPreVote, Payload: payload})
	var v PreVoteResponse
	if err := json.Unmarshal(resp.Payload, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.VoteGranted {
		t.Fatal("pre-vote must be refused while leader is reachable")
	}
}

// TestHandlePreVoteGrantsWithFreshLog validates the affirmative path:
// no recent leader contact, up-to-date log => grant.
func TestHandlePreVoteGrantsWithFreshLog(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "n1",
		ElectionTimeout: 300 * time.Millisecond,
	})
	n.currentTerm = 2
	n.leaderId = "" // no known leader
	n.lastContact = time.Now().Add(-1 * time.Hour)

	payload, _ := json.Marshal(PreVoteRequest{
		Term:         3,
		CandidateId:  "n2",
		LastLogIndex: 0,
		LastLogTerm:  0,
	})
	resp := n.handlePreVote(&RPCMessage{Type: MsgPreVote, Payload: payload})
	var v PreVoteResponse
	if err := json.Unmarshal(resp.Payload, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !v.VoteGranted {
		t.Fatal("pre-vote should be granted with stale leader and up-to-date log")
	}
}

// TestRunPreVoteSoloNodeSucceeds: a single-node "cluster" needs no
// peers to win pre-vote.
func TestRunPreVoteSoloNodeSucceeds(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "solo",
		ElectionTimeout: 100 * time.Millisecond,
	})
	if !n.runPreVote() {
		t.Fatal("solo node must win pre-vote without peers")
	}
}
