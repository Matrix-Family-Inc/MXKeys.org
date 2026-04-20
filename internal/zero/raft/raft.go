/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 */

package raft

import (
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"

	"mxkeys/internal/zero/nettls"
)

var (
	ErrNotLeader = errors.New("not the leader")
	ErrNoQuorum  = errors.New("no quorum")
	ErrShutdown  = errors.New("node is shutting down")
	ErrTimeout   = errors.New("operation timeout")
)

// State represents the Raft node state
type State int

const (
	Follower State = iota
	Candidate
	Leader
)

func (s State) String() string {
	switch s {
	case Follower:
		return "follower"
	case Candidate:
		return "candidate"
	case Leader:
		return "leader"
	default:
		return "unknown"
	}
}

// LogEntry represents an entry in the Raft log
type LogEntry struct {
	Index   uint64          `json:"index"`
	Term    uint64          `json:"term"`
	Command json.RawMessage `json:"command"`
}

// Config holds Raft configuration
type Config struct {
	NodeID            string
	BindAddress       string
	BindPort          int
	Peers             []string
	ElectionTimeout   time.Duration
	HeartbeatInterval time.Duration
	CommitTimeout     time.Duration
	SharedSecret      string

	// TLS configures transport-level encryption and mutual
	// authentication for Raft peer traffic. When TLS.Enabled is false
	// Raft uses plain TCP (backward-compatible default). Operators
	// SHOULD enable TLS with mutual auth in every production cluster.
	TLS nettls.Config
}

// Node represents a Raft node
type Node struct {
	config Config

	mu          sync.RWMutex
	state       State
	currentTerm uint64
	votedFor    string
	log         []LogEntry
	commitIndex uint64
	lastApplied uint64

	// logOffset is the number of entries logically present in the log but
	// absent from the in-memory slice. Invariant:
	//     n.log[i].Index == n.logOffset + uint64(i) + 1
	// Grows monotonically only via CompactLog/InstallSnapshot after those
	// operations successfully persist a snapshot that covers the dropped
	// prefix. Always equals snapshotIndex when non-zero.
	logOffset uint64

	// snapshotIndex is the highest Raft log index reflected in the latest
	// persisted snapshot. Entries with Index <= snapshotIndex may be absent
	// from n.log (compacted) and must be served from disk via InstallSnapshot.
	snapshotIndex uint64
	snapshotTerm  uint64

	// pendingSnapshot buffers InstallSnapshot chunks from the current
	// leader while they arrive. Reset whenever a chunk arrives with
	// Offset==0 (new transfer starts) or a different
	// (LastIncludedIndex, LastIncludedTerm) tuple (leader moved on).
	pendingSnapshot         []byte
	pendingSnapshotIndex    uint64
	pendingSnapshotTerm     uint64
	pendingSnapshotExpected uint64 // next expected Offset

	// Leader state
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	// Volatile state
	leaderId    string
	lastContact time.Time

	// Channels
	applyCh  chan LogEntry
	stopCh   chan struct{}
	stopOnce sync.Once
	replayMu sync.Mutex
	seenRPCs map[string]time.Time

	// Network
	listener net.Listener

	// Persistence
	wal      *WAL
	stateDir string

	// Callbacks
	onStateChange     func(State)
	onApply           func(LogEntry)
	snapshotProvider  SnapshotProvider
	snapshotInstaller SnapshotInstaller

	wg sync.WaitGroup
}

// RPC message types
type MessageType string

const (
	MsgPreVote        MessageType = "pre_vote"
	MsgPreVoteRes     MessageType = "pre_vote_response"
	MsgRequestVote    MessageType = "request_vote"
	MsgRequestVoteRes MessageType = "request_vote_response"
	MsgAppendEntries  MessageType = "append_entries"
	MsgAppendRes      MessageType = "append_entries_response"
)

// RequestVoteRequest is sent by candidates to gather votes
type RequestVoteRequest struct {
	Term         uint64 `json:"term"`
	CandidateId  string `json:"candidate_id"`
	LastLogIndex uint64 `json:"last_log_index"`
	LastLogTerm  uint64 `json:"last_log_term"`
}

// RequestVoteResponse is the response to a vote request
type RequestVoteResponse struct {
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"vote_granted"`
}

// PreVoteRequest is the pre-vote extension (Ongaro thesis, 9.6). The
// candidate probes peers without incrementing its own term so that a
// partitioned or flapping node cannot disrupt a stable leader by
// triggering an election that bumps everyone else's term. Only if a
// majority of peers would grant a real vote does the candidate proceed
// to startElection.
//
// Shape matches RequestVote exactly: peers compute the same "would I
// vote yes?" condition in both cases, but the pre-vote path does not
// mutate server state.
type PreVoteRequest struct {
	Term         uint64 `json:"term"`
	CandidateId  string `json:"candidate_id"`
	LastLogIndex uint64 `json:"last_log_index"`
	LastLogTerm  uint64 `json:"last_log_term"`
}

// PreVoteResponse is the response to a pre-vote probe.
type PreVoteResponse struct {
	Term        uint64 `json:"term"`
	VoteGranted bool   `json:"vote_granted"`
}

// AppendEntriesRequest is sent by leaders to replicate log entries
type AppendEntriesRequest struct {
	Term         uint64     `json:"term"`
	LeaderId     string     `json:"leader_id"`
	PrevLogIndex uint64     `json:"prev_log_index"`
	PrevLogTerm  uint64     `json:"prev_log_term"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit uint64     `json:"leader_commit"`
}

// AppendEntriesResponse is the response to append entries
type AppendEntriesResponse struct {
	Term    uint64 `json:"term"`
	Success bool   `json:"success"`
}

// RPCMessage wraps all RPC messages
type RPCMessage struct {
	Type      MessageType     `json:"type"`
	From      string          `json:"from"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature,omitempty"`
}

// NewNode creates a new Raft node
func NewNode(cfg Config) *Node {
	if cfg.ElectionTimeout == 0 {
		cfg.ElectionTimeout = 1 * time.Second
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = cfg.ElectionTimeout / 4
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 250 * time.Millisecond
	}
	if cfg.HeartbeatInterval >= cfg.ElectionTimeout {
		cfg.HeartbeatInterval = cfg.ElectionTimeout / 2
	}
	if cfg.CommitTimeout == 0 {
		cfg.CommitTimeout = 5 * time.Second
	}

	return &Node{
		config:      cfg,
		state:       Follower,
		log:         make([]LogEntry, 0),
		nextIndex:   make(map[string]uint64),
		matchIndex:  make(map[string]uint64),
		seenRPCs:    make(map[string]time.Time),
		applyCh:     make(chan LogEntry, 100),
		stopCh:      make(chan struct{}),
		lastContact: time.Now(),
	}
}
