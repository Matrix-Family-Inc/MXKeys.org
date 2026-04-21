package raft

import (
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

// RPC signing, verification, and handler tests live in
// raft_protocol_test.go to keep this file under the ADR-0010 cap.

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
	defer func() { _ = node.Stop() }()

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
