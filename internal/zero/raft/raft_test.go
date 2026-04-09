package raft

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"
)

const testRaftSecret = "raft-test-secret"

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func decodeVoteResponse(t *testing.T, msg *RPCMessage) RequestVoteResponse {
	t.Helper()
	if msg == nil {
		t.Fatalf("expected non-nil RPC response")
	}
	var out RequestVoteResponse
	if err := json.Unmarshal(msg.Payload, &out); err != nil {
		t.Fatalf("failed to decode vote response: %v", err)
	}
	return out
}

func decodeAppendResponse(t *testing.T, msg *RPCMessage) AppendEntriesResponse {
	t.Helper()
	if msg == nil {
		t.Fatalf("expected non-nil RPC response")
	}
	var out AppendEntriesResponse
	if err := json.Unmarshal(msg.Payload, &out); err != nil {
		t.Fatalf("failed to decode append response: %v", err)
	}
	return out
}

func TestNewNodeDefaultsAndInitialState(t *testing.T) {
	node := NewNode(Config{NodeID: "n1", BindAddress: "127.0.0.1", BindPort: freePort(t)})
	if node.State() != Follower {
		t.Fatalf("initial state = %s, want %s", node.State(), Follower)
	}
	if node.Term() != 0 {
		t.Fatalf("initial term = %d, want 0", node.Term())
	}
	if node.config.ElectionTimeout == 0 || node.config.HeartbeatInterval == 0 || node.config.CommitTimeout == 0 {
		t.Fatalf("default timeouts must be initialized")
	}
	if node.config.ElectionTimeout != time.Second {
		t.Fatalf("default election timeout = %v, want 1s", node.config.ElectionTimeout)
	}
	if node.config.HeartbeatInterval != 250*time.Millisecond {
		t.Fatalf("default heartbeat interval = %v, want 250ms", node.config.HeartbeatInterval)
	}
}

func TestStateString(t *testing.T) {
	cases := []struct {
		in   State
		want string
	}{
		{Follower, "follower"},
		{Candidate, "candidate"},
		{Leader, "leader"},
		{State(99), "unknown"},
	}
	for _, tc := range cases {
		if got := tc.in.String(); got != tc.want {
			t.Fatalf("State(%d).String() = %q, want %q", tc.in, got, tc.want)
		}
	}
}

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

func TestStartElectionSingleNodeBecomesLeader(t *testing.T) {
	node := NewNode(Config{NodeID: "n1", SharedSecret: testRaftSecret})

	node.startElection()

	if !node.IsLeader() {
		t.Fatalf("single-node election should become leader, got state %s", node.State())
	}
}

func TestStartElectionWithPeerVote(t *testing.T) {
	peerListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen for peer: %v", err)
	}
	defer peerListener.Close()

	peerNode := NewNode(Config{NodeID: "n2", SharedSecret: testRaftSecret})
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := peerListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		msg, err := peerNode.readRPC(conn)
		if err != nil {
			return
		}
		if err := peerNode.verifyRPC(msg); err != nil {
			return
		}
		resp := peerNode.handleRPC(msg)
		if resp == nil {
			return
		}
		if err := peerNode.signRPC(resp); err != nil {
			return
		}
		_ = peerNode.writeRPC(conn, resp)
	}()

	node := NewNode(Config{
		NodeID:          "n1",
		Peers:           []string{peerListener.Addr().String()},
		SharedSecret:    testRaftSecret,
		ElectionTimeout: 100 * time.Millisecond,
	})

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		node.startElection()
	}()

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("startElection appears to be blocked")
	}
	<-done

	if !node.IsLeader() {
		t.Fatalf("expected leader after receiving peer vote, got state %s", node.State())
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

func TestSubmitAsLeaderAppendsEntryAndReturnsReplicationResult(t *testing.T) {
	node := NewNode(Config{
		NodeID:        "n1",
		CommitTimeout: 30 * time.Millisecond,
	})
	node.state = Leader
	node.currentTerm = 7

	err := node.Submit(context.Background(), []byte(`{"op":"set"}`))
	if err != nil {
		t.Fatalf("Submit() error = %v, want nil for single-node leader", err)
	}
	if len(node.log) != 1 {
		t.Fatalf("log length = %d, want 1", len(node.log))
	}
	if node.log[0].Index != 1 || node.log[0].Term != 7 {
		t.Fatalf("unexpected log entry metadata: %+v", node.log[0])
	}
	if node.commitIndex != 1 {
		t.Fatalf("commitIndex = %d, want 1", node.commitIndex)
	}
}

func TestSubmitRejectsFollower(t *testing.T) {
	node := NewNode(Config{NodeID: "n1"})
	if err := node.Submit(context.Background(), []byte("cmd")); !errors.Is(err, ErrNotLeader) {
		t.Fatalf("Submit() error = %v, want ErrNotLeader", err)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	node := NewNode(Config{
		NodeID:            "node1",
		BindAddress:       "127.0.0.1",
		BindPort:          freePort(t),
		ElectionTimeout:   40 * time.Millisecond,
		HeartbeatInterval: 15 * time.Millisecond,
		CommitTimeout:     100 * time.Millisecond,
		SharedSecret:      testRaftSecret,
	})

	if err := node.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := node.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestRandomElectionTimeoutRange(t *testing.T) {
	node := NewNode(Config{NodeID: "n1", ElectionTimeout: 100 * time.Millisecond})

	observed := map[time.Duration]struct{}{}
	for i := 0; i < 20; i++ {
		v := node.randomElectionTimeout()
		observed[v] = struct{}{}
		if v < 100*time.Millisecond || v > 200*time.Millisecond {
			t.Fatalf("timeout %v outside expected range [100ms, 200ms]", v)
		}
	}
	if len(observed) < 2 {
		t.Fatalf("randomElectionTimeout does not appear to add jitter")
	}
}

func TestNewNodeClampsHeartbeatBelowElectionTimeout(t *testing.T) {
	node := NewNode(Config{
		NodeID:            "n1",
		ElectionTimeout:   200 * time.Millisecond,
		HeartbeatInterval: 500 * time.Millisecond,
	})

	if node.config.HeartbeatInterval >= node.config.ElectionTimeout {
		t.Fatalf("heartbeat interval must stay below election timeout, got heartbeat=%v election=%v", node.config.HeartbeatInterval, node.config.ElectionTimeout)
	}
}
