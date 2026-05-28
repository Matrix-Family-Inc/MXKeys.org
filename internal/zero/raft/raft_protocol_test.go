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
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRPCSigning(t *testing.T) {
	node := NewNode(Config{NodeID: "n1", SharedSecret: testRaftSecret})
	msg := &RPCMessage{
		Type:    MsgRequestVote,
		From:    "n1",
		Payload: json.RawMessage(`{"term":1}`),
	}
	if err := node.signRPC(msg); err != nil {
		t.Fatalf("signRPC() error = %v", err)
	}
	if err := node.verifyRPC(msg); err != nil {
		t.Fatalf("verifyRPC() error = %v", err)
	}
	msg.Payload = json.RawMessage(`{"term":2}`)
	if err := node.verifyRPC(msg); err == nil {
		t.Fatal("expected tampered RPC to fail verification")
	}
}

func TestVerifyRPCRejectsReplay(t *testing.T) {
	node := NewNode(Config{NodeID: "n1", SharedSecret: testRaftSecret})
	msg := &RPCMessage{
		Type:      MsgRequestVote,
		From:      "n1",
		Timestamp: time.Now(),
		Payload:   json.RawMessage(`{"term":1}`),
	}
	if err := node.signRPC(msg); err != nil {
		t.Fatalf("signRPC() error = %v", err)
	}
	if err := node.verifyRPC(msg); err != nil {
		t.Fatalf("verifyRPC() first pass error = %v", err)
	}
	if err := node.verifyRPC(msg); err == nil {
		t.Fatal("expected replayed RPC to be rejected")
	}
}

func TestVerifyRPCRejectsStaleTimestamp(t *testing.T) {
	node := NewNode(Config{NodeID: "n1", SharedSecret: testRaftSecret})
	msg := &RPCMessage{
		Type:      MsgRequestVote,
		From:      "n1",
		Timestamp: time.Now().Add(-maxRPCSkew - time.Second),
		Payload:   json.RawMessage(`{"term":1}`),
	}
	if err := node.signRPC(msg); err != nil {
		t.Fatalf("signRPC() error = %v", err)
	}
	if err := node.verifyRPC(msg); err == nil {
		t.Fatal("expected stale RPC timestamp to fail verification")
	}
}

func TestReadBoundedRPCRejectsOversizedPayload(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxRaftMessageSize+1)
	if _, err := readBoundedRPC(bytes.NewReader(oversized), maxRaftMessageSize); err == nil {
		t.Fatal("expected oversized raft payload to be rejected")
	}
}


func TestHandleRequestVoteTermAndLogRules(t *testing.T) {
	node := NewNode(Config{NodeID: "n-local"})
	node.currentTerm = 3
	node.log = []LogEntry{{Index: 1, Term: 3}}

	// Lower term is always rejected.
	lowerReq := RequestVoteRequest{Term: 2, CandidateId: "cand", LastLogIndex: 1, LastLogTerm: 3}
	lowerPayload, _ := json.Marshal(lowerReq)
	lowerResp := decodeVoteResponse(t, node.handleRequestVote(&RPCMessage{Type: MsgRequestVote, Payload: lowerPayload}))
	if lowerResp.VoteGranted {
		t.Fatalf("lower-term candidate must be rejected")
	}
	if lowerResp.Term != 3 {
		t.Fatalf("response term = %d, want 3", lowerResp.Term)
	}

	// Higher term but stale log must be rejected.
	staleReq := RequestVoteRequest{Term: 4, CandidateId: "cand", LastLogIndex: 1, LastLogTerm: 2}
	stalePayload, _ := json.Marshal(staleReq)
	staleResp := decodeVoteResponse(t, node.handleRequestVote(&RPCMessage{Type: MsgRequestVote, Payload: stalePayload}))
	if staleResp.VoteGranted {
		t.Fatalf("stale candidate log must be rejected")
	}
	if node.currentTerm != 4 {
		t.Fatalf("currentTerm = %d, want 4 after higher-term request", node.currentTerm)
	}

	// Higher term and up-to-date log must be accepted.
	freshReq := RequestVoteRequest{Term: 5, CandidateId: "cand", LastLogIndex: 1, LastLogTerm: 3}
	freshPayload, _ := json.Marshal(freshReq)
	freshResp := decodeVoteResponse(t, node.handleRequestVote(&RPCMessage{Type: MsgRequestVote, Payload: freshPayload}))
	if !freshResp.VoteGranted {
		t.Fatalf("up-to-date candidate should receive vote")
	}
	if node.votedFor != "cand" {
		t.Fatalf("votedFor = %q, want cand", node.votedFor)
	}
}

func TestHandleAppendEntriesRejectsInconsistentPrevLog(t *testing.T) {
	node := NewNode(Config{NodeID: "n-local"})
	node.currentTerm = 2
	node.log = []LogEntry{{Index: 1, Term: 1}}

	tooFar := AppendEntriesRequest{
		Term:         2,
		LeaderId:     "leader",
		PrevLogIndex: 2,
		PrevLogTerm:  1,
	}
	p1, _ := json.Marshal(tooFar)
	resp1 := decodeAppendResponse(t, node.handleAppendEntries(&RPCMessage{Type: MsgAppendEntries, Payload: p1}))
	if resp1.Success {
		t.Fatalf("append must fail when PrevLogIndex exceeds local log length")
	}

	badTerm := AppendEntriesRequest{
		Term:         2,
		LeaderId:     "leader",
		PrevLogIndex: 1,
		PrevLogTerm:  99,
	}
	p2, _ := json.Marshal(badTerm)
	resp2 := decodeAppendResponse(t, node.handleAppendEntries(&RPCMessage{Type: MsgAppendEntries, Payload: p2}))
	if resp2.Success {
		t.Fatalf("append must fail on PrevLogTerm mismatch")
	}
}

func TestHandleAppendEntriesAppendsAndAdvancesCommit(t *testing.T) {
	node := NewNode(Config{NodeID: "n-local"})
	node.currentTerm = 1
	node.log = []LogEntry{{Index: 1, Term: 1, Command: json.RawMessage(`"a"`)}}

	req := AppendEntriesRequest{
		Term:         1,
		LeaderId:     "leader",
		PrevLogIndex: 1,
		PrevLogTerm:  1,
		Entries:      []LogEntry{{Index: 2, Term: 1, Command: json.RawMessage(`"b"`)}},
		LeaderCommit: 2,
	}
	payload, _ := json.Marshal(req)
	resp := decodeAppendResponse(t, node.handleAppendEntries(&RPCMessage{Type: MsgAppendEntries, Payload: payload}))
	if !resp.Success {
		t.Fatalf("append entries should succeed for consistent log")
	}
	if len(node.log) != 2 || node.log[1].Command[1] != 'b' {
		t.Fatalf("log was not appended correctly")
	}
	if node.commitIndex != 2 {
		t.Fatalf("commitIndex = %d, want 2", node.commitIndex)
	}
	if node.LeaderID() != "leader" {
		t.Fatalf("leaderId = %q, want leader", node.LeaderID())
	}
}

func TestUpdateCommitIndexRequiresMajorityInCurrentTerm(t *testing.T) {
	node := NewNode(Config{
		NodeID: "n1",
		Peers:  []string{"n2", "n3"},
	})
	node.state = Leader
	node.currentTerm = 2
	node.log = []LogEntry{
		{Index: 1, Term: 1},
		{Index: 2, Term: 2},
	}
	node.matchIndex["n2"] = 2
	node.matchIndex["n3"] = 1

	node.updateCommitIndex()
	if node.commitIndex != 2 {
		t.Fatalf("commitIndex = %d, want 2", node.commitIndex)
	}

	node.commitIndex = 0
	node.matchIndex["n2"] = 1
	node.matchIndex["n3"] = 1
	node.updateCommitIndex()
	if node.commitIndex != 0 {
		t.Fatalf("entry from old term must not be committed as leader term evidence")
	}
}

func TestReplicateEntryExitPaths(t *testing.T) {
	t.Run("context canceled", func(t *testing.T) {
		node := NewNode(Config{NodeID: "n1", CommitTimeout: 100 * time.Millisecond})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := node.replicateEntry(ctx, LogEntry{Index: 1})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	})

	t.Run("shutdown", func(t *testing.T) {
		node := NewNode(Config{NodeID: "n1", CommitTimeout: 100 * time.Millisecond})
		close(node.stopCh)
		err := node.replicateEntry(context.Background(), LogEntry{Index: 1})
		if !errors.Is(err, ErrShutdown) {
			t.Fatalf("error = %v, want ErrShutdown", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		node := NewNode(Config{NodeID: "n1", CommitTimeout: 40 * time.Millisecond})
		err := node.replicateEntry(context.Background(), LogEntry{Index: 1})
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("error = %v, want ErrTimeout", err)
		}
	})
}
