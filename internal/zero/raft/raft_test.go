package raft

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9000,
	}

	node := NewNode(cfg)
	if node == nil {
		t.Fatal("NewNode returned nil")
	}

	if node.State() != Follower {
		t.Errorf("initial state = %v, want Follower", node.State())
	}

	if node.Term() != 0 {
		t.Errorf("initial term = %d, want 0", node.Term())
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{Follower, "follower"},
		{Candidate, "candidate"},
		{Leader, "leader"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestNodeStats(t *testing.T) {
	cfg := Config{
		NodeID:      "stats-node",
		BindAddress: "127.0.0.1",
		BindPort:    9001,
	}

	node := NewNode(cfg)
	stats := node.Stats()

	if stats["node_id"] != "stats-node" {
		t.Errorf("node_id = %v, want stats-node", stats["node_id"])
	}

	if stats["state"] != "follower" {
		t.Errorf("state = %v, want follower", stats["state"])
	}

	if stats["term"].(uint64) != 0 {
		t.Errorf("term = %v, want 0", stats["term"])
	}
}

func TestLogEntry(t *testing.T) {
	entry := LogEntry{
		Index:   1,
		Term:    1,
		Command: json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed LogEntry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Index != 1 {
		t.Errorf("Index = %d, want 1", parsed.Index)
	}

	if parsed.Term != 1 {
		t.Errorf("Term = %d, want 1", parsed.Term)
	}
}

func TestRequestVoteRequest(t *testing.T) {
	req := RequestVoteRequest{
		Term:         5,
		CandidateId:  "node1",
		LastLogIndex: 10,
		LastLogTerm:  4,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal RequestVoteRequest: %v", err)
	}

	var parsed RequestVoteRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal RequestVoteRequest: %v", err)
	}

	if parsed.Term != 5 {
		t.Errorf("Term = %d, want 5", parsed.Term)
	}

	if parsed.CandidateId != "node1" {
		t.Errorf("CandidateId = %q, want node1", parsed.CandidateId)
	}
}

func TestAppendEntriesRequest(t *testing.T) {
	req := AppendEntriesRequest{
		Term:         5,
		LeaderId:     "leader1",
		PrevLogIndex: 10,
		PrevLogTerm:  4,
		Entries: []LogEntry{
			{Index: 11, Term: 5, Command: json.RawMessage(`{}`)},
		},
		LeaderCommit: 10,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal AppendEntriesRequest: %v", err)
	}

	var parsed AppendEntriesRequest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal AppendEntriesRequest: %v", err)
	}

	if parsed.LeaderId != "leader1" {
		t.Errorf("LeaderId = %q, want leader1", parsed.LeaderId)
	}

	if len(parsed.Entries) != 1 {
		t.Errorf("Entries length = %d, want 1", len(parsed.Entries))
	}
}

func TestRPCMessage(t *testing.T) {
	payload, err := json.Marshal(RequestVoteRequest{Term: 1})
	if err != nil {
		t.Fatalf("failed to marshal RequestVoteRequest payload: %v", err)
	}
	msg := RPCMessage{
		Type:    MsgRequestVote,
		From:    "node1",
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal RPCMessage: %v", err)
	}

	var parsed RPCMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal RPCMessage: %v", err)
	}

	if parsed.Type != MsgRequestVote {
		t.Errorf("Type = %v, want request_vote", parsed.Type)
	}

	if parsed.From != "node1" {
		t.Errorf("From = %q, want node1", parsed.From)
	}
}

func TestIsLeader(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9002,
	}

	node := NewNode(cfg)

	if node.IsLeader() {
		t.Error("new node should not be leader")
	}
}

func TestLeaderID(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9003,
	}

	node := NewNode(cfg)

	if node.LeaderID() != "" {
		t.Errorf("initial leaderID = %q, want empty", node.LeaderID())
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9004,
	}

	node := NewNode(cfg)

	if node.config.ElectionTimeout == 0 {
		t.Error("ElectionTimeout should have default")
	}

	if node.config.HeartbeatInterval == 0 {
		t.Error("HeartbeatInterval should have default")
	}

	if node.config.CommitTimeout == 0 {
		t.Error("CommitTimeout should have default")
	}
}

func TestSubmitNotLeader(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9005,
	}

	node := NewNode(cfg)

	ctx := context.Background()
	err := node.Submit(ctx, []byte("command"))

	if err != ErrNotLeader {
		t.Errorf("Submit on follower should return ErrNotLeader, got %v", err)
	}
}

func TestRandomElectionTimeout(t *testing.T) {
	cfg := Config{
		NodeID:          "node1",
		BindAddress:     "127.0.0.1",
		BindPort:        9006,
		ElectionTimeout: 100 * time.Millisecond,
	}

	node := NewNode(cfg)

	timeouts := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		timeout := node.randomElectionTimeout()
		timeouts[timeout] = true

		if timeout < cfg.ElectionTimeout {
			t.Errorf("timeout %v is less than base %v", timeout, cfg.ElectionTimeout)
		}

		if timeout > 2*cfg.ElectionTimeout {
			t.Errorf("timeout %v is more than 2x base", timeout)
		}
	}

	if len(timeouts) < 2 {
		t.Error("randomElectionTimeout should produce varied values")
	}
}

func TestIsLogUpToDate(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9007,
	}

	node := NewNode(cfg)

	// Empty log - any log is up to date
	if !node.isLogUpToDate(0, 0) {
		t.Error("empty log should consider any log up to date")
	}

	// Add entry
	node.log = append(node.log, LogEntry{Index: 1, Term: 1})

	// Same term, same index
	if !node.isLogUpToDate(1, 1) {
		t.Error("same log should be up to date")
	}

	// Higher term
	if !node.isLogUpToDate(0, 2) {
		t.Error("higher term should be up to date")
	}

	// Lower term
	if node.isLogUpToDate(10, 0) {
		t.Error("lower term should not be up to date")
	}
}

func TestWrapResponse(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9008,
	}

	node := NewNode(cfg)

	resp := RequestVoteResponse{
		Term:        5,
		VoteGranted: true,
	}

	msg := node.wrapResponse(MsgRequestVoteRes, resp)

	if msg.Type != MsgRequestVoteRes {
		t.Errorf("Type = %v, want request_vote_response", msg.Type)
	}

	if msg.From != "node1" {
		t.Errorf("From = %q, want node1", msg.From)
	}

	var parsed RequestVoteResponse
	json.Unmarshal(msg.Payload, &parsed)

	if !parsed.VoteGranted {
		t.Error("VoteGranted should be true")
	}
}

func TestSetCallbacks(t *testing.T) {
	cfg := Config{
		NodeID:      "node1",
		BindAddress: "127.0.0.1",
		BindPort:    9009,
	}

	node := NewNode(cfg)

	stateChangeCalled := false
	applyCalled := false

	node.SetOnStateChange(func(s State) {
		stateChangeCalled = true
	})

	node.SetOnApply(func(e LogEntry) {
		applyCalled = true
	})

	if node.onStateChange == nil {
		t.Error("onStateChange should be set")
	}

	if node.onApply == nil {
		t.Error("onApply should be set")
	}

	_ = stateChangeCalled
	_ = applyCalled
}

func TestMessageTypes(t *testing.T) {
	types := []MessageType{
		MsgRequestVote,
		MsgRequestVoteRes,
		MsgAppendEntries,
		MsgAppendRes,
	}

	for _, msgType := range types {
		if msgType == "" {
			t.Error("message type should not be empty")
		}
	}
}

func TestErrors(t *testing.T) {
	errors := []error{
		ErrNotLeader,
		ErrNoQuorum,
		ErrShutdown,
		ErrTimeout,
	}

	for _, err := range errors {
		if err.Error() == "" {
			t.Error("error should have message")
		}
	}
}
