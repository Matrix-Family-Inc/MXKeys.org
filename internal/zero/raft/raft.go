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
	"fmt"
	"hash"
	"net"
	"os"
	"sync"
	"time"

	"mxkeys/internal/zero/nettls"
)

var (
	ErrNotLeader = errors.New("not the leader")
	ErrNoQuorum  = errors.New("no quorum")
	ErrShutdown  = errors.New("node is shutting down")
	ErrTimeout   = errors.New("operation timeout")
	// ErrNoLeader: Propose on a follower with no known leader
	// (e.g. mid-election). Callers back off and retry.
	ErrNoLeader = errors.New("no leader known")
	// ErrSnapshotInstallerRequired: snapshot arrives (LoadFromDisk
	// or InstallSnapshot) but no SnapshotInstaller is registered.
	// Raft refuses rather than desync the state machine.
	ErrSnapshotInstallerRequired = errors.New("raft: SetSnapshotInstaller is required to process a persisted or incoming snapshot")
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

	// AdvertiseAddr is the dialable "host:port" peers use to reach
	// this node. Leader embeds it in every AE / InstallSnapshot so
	// followers learn a concrete forward endpoint (see Propose).
	// Empty falls back to BindAddress:BindPort; that only works
	// when BindAddress is a real interface, not a wildcard.
	AdvertiseAddr string

	// TLS configures transport encryption and mutual auth for Raft
	// peer traffic. TLS.Enabled=false keeps plain TCP (backward-
	// compatible). Production SHOULD enable TLS with mutual auth.
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

	// logOffset is the count of entries logically present but
	// absent from the in-memory slice. Invariant:
	//   n.log[i].Index == n.logOffset + uint64(i) + 1
	// Grows monotonically via CompactLog/InstallSnapshot once the
	// covering snapshot is persisted. Equals snapshotIndex when
	// non-zero.
	logOffset uint64

	// snapshotIndex is the highest log index reflected in the
	// latest persisted snapshot; entries with Index <= snapshotIndex
	// may be absent from n.log and are served from disk.
	snapshotIndex uint64
	snapshotTerm  uint64

	// pendingSnapshot* hold the in-flight InstallSnapshot transfer.
	// stateDir != "" streams chunks to raft.snapshot.recv for
	// O(chunk) RAM; stateDir=="" buffers in pendingSnapshot. See
	// pending_snapshot.go for the lifecycle.
	pendingSnapshot         []byte
	pendingSnapshotFile     *os.File
	pendingSnapshotPath     string
	pendingSnapshotIndex    uint64
	pendingSnapshotTerm     uint64
	pendingSnapshotExpected uint64      // next expected Offset = bytes accumulated so far
	pendingSnapshotCRC      hash.Hash32 // incremental Castagnoli CRC over data chunks

	// Leader state
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	// Volatile state
	leaderId    string
	leaderAddr  string
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

	// snapMu serialises every writer of raft.snapshot on disk and
	// the matching in-memory (snapshotIndex, snapshotTerm,
	// logOffset) bookkeeping: CompactLog and every
	// handleInstallSnapshot. Without it two snapshot writers could
	// roll the persisted state backwards or trash the spill file.
	// Lock order: snapMu before n.mu; never held while taking an
	// application-level lock.
	snapMu sync.Mutex

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

// MsgForwardProposal, ForwardProposalRequest and ForwardProposalResponse
// are declared in propose.go alongside the Propose/handleForwardProposal
// implementation.

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

// AppendEntriesRequest is sent by leaders to replicate log entries.
//
// LeaderAddress is the leader's dialable "host:port" and is used by
// followers to forward client proposals (see Propose). Empty when
// the leader was built without an AdvertiseAddr; in that case
// followers cannot forward and Propose returns ErrNoLeader.
type AppendEntriesRequest struct {
	Term          uint64     `json:"term"`
	LeaderId      string     `json:"leader_id"`
	LeaderAddress string     `json:"leader_address,omitempty"`
	PrevLogIndex  uint64     `json:"prev_log_index"`
	PrevLogTerm   uint64     `json:"prev_log_term"`
	Entries       []LogEntry `json:"entries"`
	LeaderCommit  uint64     `json:"leader_commit"`
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

// advertiseAddr returns the dialable "host:port" peers should use to
// reach this node. Falls back to the bind address when AdvertiseAddr
// is unset. Returns "" only when neither is usable, in which case
// leader-forward cannot work for this node.
func (n *Node) advertiseAddr() string {
	if s := n.config.AdvertiseAddr; s != "" {
		return s
	}
	if n.config.BindAddress != "" && n.config.BindPort > 0 {
		return fmt.Sprintf("%s:%d", n.config.BindAddress, n.config.BindPort)
	}
	return ""
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
