/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 */

package raft

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"
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

	// Leader state
	nextIndex  map[string]uint64
	matchIndex map[string]uint64

	// Volatile state
	leaderId    string
	lastContact time.Time

	// Channels
	applyCh chan LogEntry
	stopCh  chan struct{}

	// Network
	listener net.Listener
	peers    map[string]net.Conn

	// Callbacks
	onStateChange func(State)
	onApply       func(LogEntry)

	wg sync.WaitGroup
}

// RPC message types
type MessageType string

const (
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
	Type    MessageType     `json:"type"`
	From    string          `json:"from"`
	Payload json.RawMessage `json:"payload"`
}

// NewNode creates a new Raft node
func NewNode(cfg Config) *Node {
	if cfg.ElectionTimeout == 0 {
		cfg.ElectionTimeout = 150 * time.Millisecond
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 50 * time.Millisecond
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
		peers:       make(map[string]net.Conn),
		applyCh:     make(chan LogEntry, 100),
		stopCh:      make(chan struct{}),
		lastContact: time.Now(),
	}
}

// Start starts the Raft node
func (n *Node) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", n.config.BindAddress, n.config.BindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	n.listener = listener

	// Accept connections
	n.wg.Add(1)
	go n.acceptLoop()

	// Connect to peers
	n.wg.Add(1)
	go n.connectPeers()

	// Run election timer
	n.wg.Add(1)
	go n.runElectionTimer()

	// Apply committed entries
	n.wg.Add(1)
	go n.applyLoop()

	return nil
}

// Stop stops the Raft node
func (n *Node) Stop() error {
	close(n.stopCh)

	if n.listener != nil {
		n.listener.Close()
	}

	n.mu.Lock()
	for _, conn := range n.peers {
		conn.Close()
	}
	n.mu.Unlock()

	n.wg.Wait()
	return nil
}

// Submit submits a command to the Raft cluster
func (n *Node) Submit(ctx context.Context, command []byte) error {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return ErrNotLeader
	}

	entry := LogEntry{
		Index:   uint64(len(n.log)) + 1,
		Term:    n.currentTerm,
		Command: command,
	}
	n.log = append(n.log, entry)
	n.mu.Unlock()

	// Replicate to peers
	return n.replicateEntry(ctx, entry)
}

// State returns the current node state
func (n *Node) State() State {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

// IsLeader returns true if this node is the leader
func (n *Node) IsLeader() bool {
	return n.State() == Leader
}

// LeaderID returns the current leader ID
func (n *Node) LeaderID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.leaderId
}

// Term returns the current term
func (n *Node) Term() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.currentTerm
}

// SetOnStateChange sets callback for state changes
func (n *Node) SetOnStateChange(fn func(State)) {
	n.onStateChange = fn
}

// SetOnApply sets callback for applied entries
func (n *Node) SetOnApply(fn func(LogEntry)) {
	n.onApply = fn
}

// acceptLoop accepts incoming connections
func (n *Node) acceptLoop() {
	defer n.wg.Done()

	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		conn, err := n.listener.Accept()
		if err != nil {
			select {
			case <-n.stopCh:
				return
			default:
				continue
			}
		}

		go n.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection
func (n *Node) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var msg RPCMessage
		if err := decoder.Decode(&msg); err != nil {
			return
		}

		response := n.handleRPC(&msg)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				return
			}
		}
	}
}

// handleRPC processes incoming RPC messages
func (n *Node) handleRPC(msg *RPCMessage) *RPCMessage {
	switch msg.Type {
	case MsgRequestVote:
		return n.handleRequestVote(msg)
	case MsgAppendEntries:
		return n.handleAppendEntries(msg)
	default:
		return nil
	}
}

// handleRequestVote handles vote requests
func (n *Node) handleRequestVote(msg *RPCMessage) *RPCMessage {
	var req RequestVoteRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		n.mu.RLock()
		term := n.currentTerm
		n.mu.RUnlock()
		return n.wrapResponse(MsgRequestVoteRes, RequestVoteResponse{
			Term:        term,
			VoteGranted: false,
		})
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	response := RequestVoteResponse{
		Term:        n.currentTerm,
		VoteGranted: false,
	}

	// Reply false if term < currentTerm
	if req.Term < n.currentTerm {
		return n.wrapResponse(MsgRequestVoteRes, response)
	}

	// Update term if necessary
	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.state = Follower
		n.votedFor = ""
	}

	// Check if we can vote
	if (n.votedFor == "" || n.votedFor == req.CandidateId) && n.isLogUpToDate(req.LastLogIndex, req.LastLogTerm) {
		n.votedFor = req.CandidateId
		n.lastContact = time.Now()
		response.VoteGranted = true
	}

	response.Term = n.currentTerm
	return n.wrapResponse(MsgRequestVoteRes, response)
}

// handleAppendEntries handles append entries requests
func (n *Node) handleAppendEntries(msg *RPCMessage) *RPCMessage {
	var req AppendEntriesRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		n.mu.RLock()
		term := n.currentTerm
		n.mu.RUnlock()
		return n.wrapResponse(MsgAppendRes, AppendEntriesResponse{
			Term:    term,
			Success: false,
		})
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	response := AppendEntriesResponse{
		Term:    n.currentTerm,
		Success: false,
	}

	// Reply false if term < currentTerm
	if req.Term < n.currentTerm {
		return n.wrapResponse(MsgAppendRes, response)
	}

	// Update term and state
	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.votedFor = ""
	}

	n.state = Follower
	n.leaderId = req.LeaderId
	n.lastContact = time.Now()

	// Check log consistency
	if req.PrevLogIndex > 0 {
		if req.PrevLogIndex > uint64(len(n.log)) {
			return n.wrapResponse(MsgAppendRes, response)
		}
		if n.log[req.PrevLogIndex-1].Term != req.PrevLogTerm {
			return n.wrapResponse(MsgAppendRes, response)
		}
	}

	// Append new entries
	for i, entry := range req.Entries {
		idx := req.PrevLogIndex + uint64(i) + 1
		if idx <= uint64(len(n.log)) {
			if n.log[idx-1].Term != entry.Term {
				n.log = n.log[:idx-1]
				n.log = append(n.log, entry)
			}
		} else {
			n.log = append(n.log, entry)
		}
	}

	// Update commit index
	if req.LeaderCommit > n.commitIndex {
		lastNew := req.PrevLogIndex + uint64(len(req.Entries))
		if req.LeaderCommit < lastNew {
			n.commitIndex = req.LeaderCommit
		} else {
			n.commitIndex = lastNew
		}
	}

	response.Success = true
	response.Term = n.currentTerm
	return n.wrapResponse(MsgAppendRes, response)
}

// runElectionTimer runs the election timeout timer
func (n *Node) runElectionTimer() {
	defer n.wg.Done()

	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		timeout := n.randomElectionTimeout()
		time.Sleep(timeout)

		n.mu.Lock()
		if n.state != Leader && time.Since(n.lastContact) >= timeout {
			n.startElection()
		}
		n.mu.Unlock()
	}
}

// startElection starts a new election
func (n *Node) startElection() {
	n.currentTerm++
	n.state = Candidate
	n.votedFor = n.config.NodeID
	n.lastContact = time.Now()

	if n.onStateChange != nil {
		go n.onStateChange(Candidate)
	}

	// Request votes from peers
	votes := 1 // Vote for self
	voteCh := make(chan bool, len(n.config.Peers))

	for _, peer := range n.config.Peers {
		go func(p string) {
			granted := n.requestVote(p)
			voteCh <- granted
		}(peer)
	}

	// Collect votes
	needed := (len(n.config.Peers)+1)/2 + 1
	for i := 0; i < len(n.config.Peers); i++ {
		select {
		case granted := <-voteCh:
			if granted {
				votes++
			}
		case <-time.After(n.config.ElectionTimeout):
		case <-n.stopCh:
			return
		}

		if votes >= needed {
			n.becomeLeader()
			return
		}
	}
}

// becomeLeader transitions to leader state
func (n *Node) becomeLeader() {
	n.state = Leader
	n.leaderId = n.config.NodeID

	// Initialize leader state
	for _, peer := range n.config.Peers {
		n.nextIndex[peer] = uint64(len(n.log)) + 1
		n.matchIndex[peer] = 0
	}

	if n.onStateChange != nil {
		go n.onStateChange(Leader)
	}

	// Start heartbeat
	go n.sendHeartbeats()
}

// sendHeartbeats sends periodic heartbeats to all peers
func (n *Node) sendHeartbeats() {
	ticker := time.NewTicker(n.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.mu.RLock()
			if n.state != Leader {
				n.mu.RUnlock()
				return
			}
			n.mu.RUnlock()

			for _, peer := range n.config.Peers {
				go n.sendAppendEntries(peer)
			}
		}
	}
}

// sendAppendEntries sends append entries to a peer
func (n *Node) sendAppendEntries(peer string) {
	n.mu.RLock()
	if n.state != Leader {
		n.mu.RUnlock()
		return
	}

	nextIdx := n.nextIndex[peer]
	prevLogIndex := nextIdx - 1
	var prevLogTerm uint64
	if prevLogIndex > 0 && prevLogIndex <= uint64(len(n.log)) {
		prevLogTerm = n.log[prevLogIndex-1].Term
	}

	var entries []LogEntry
	if nextIdx <= uint64(len(n.log)) {
		entries = n.log[nextIdx-1:]
	}

	req := AppendEntriesRequest{
		Term:         n.currentTerm,
		LeaderId:     n.config.NodeID,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: n.commitIndex,
	}
	n.mu.RUnlock()

	resp, err := n.sendRPC(peer, MsgAppendEntries, req)
	if err != nil {
		return
	}

	var appendResp AppendEntriesResponse
	if err := json.Unmarshal(resp.Payload, &appendResp); err != nil {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if appendResp.Term > n.currentTerm {
		n.currentTerm = appendResp.Term
		n.state = Follower
		n.votedFor = ""
		return
	}

	if appendResp.Success {
		n.nextIndex[peer] = nextIdx + uint64(len(entries))
		n.matchIndex[peer] = n.nextIndex[peer] - 1
		n.updateCommitIndex()
	} else {
		if n.nextIndex[peer] > 1 {
			n.nextIndex[peer]--
		}
	}
}

// updateCommitIndex updates commit index based on matchIndex
func (n *Node) updateCommitIndex() {
	for i := n.commitIndex + 1; i <= uint64(len(n.log)); i++ {
		if n.log[i-1].Term != n.currentTerm {
			continue
		}

		matches := 1 // Self
		for _, peer := range n.config.Peers {
			if n.matchIndex[peer] >= i {
				matches++
			}
		}

		if matches > (len(n.config.Peers)+1)/2 {
			n.commitIndex = i
		}
	}
}

// applyLoop applies committed entries
func (n *Node) applyLoop() {
	defer n.wg.Done()

	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		n.mu.Lock()
		for n.lastApplied < n.commitIndex {
			n.lastApplied++
			entry := n.log[n.lastApplied-1]
			n.mu.Unlock()

			if n.onApply != nil {
				n.onApply(entry)
			}

			select {
			case n.applyCh <- entry:
			default:
			}

			n.mu.Lock()
		}
		n.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}
}

// requestVote sends a vote request to a peer
func (n *Node) requestVote(peer string) bool {
	n.mu.RLock()
	lastLogIndex := uint64(len(n.log))
	var lastLogTerm uint64
	if lastLogIndex > 0 {
		lastLogTerm = n.log[lastLogIndex-1].Term
	}

	req := RequestVoteRequest{
		Term:         n.currentTerm,
		CandidateId:  n.config.NodeID,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
	n.mu.RUnlock()

	resp, err := n.sendRPC(peer, MsgRequestVote, req)
	if err != nil {
		return false
	}

	var voteResp RequestVoteResponse
	if err := json.Unmarshal(resp.Payload, &voteResp); err != nil {
		return false
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if voteResp.Term > n.currentTerm {
		n.currentTerm = voteResp.Term
		n.state = Follower
		n.votedFor = ""
		return false
	}

	return voteResp.VoteGranted
}

// connectPeers connects to peer nodes
func (n *Node) connectPeers() {
	defer n.wg.Done()

	for _, peer := range n.config.Peers {
		conn, err := net.DialTimeout("tcp", peer, 5*time.Second)
		if err != nil {
			continue
		}

		n.mu.Lock()
		n.peers[peer] = conn
		n.mu.Unlock()
	}
}

// sendRPC sends an RPC message to a peer
func (n *Node) sendRPC(peer string, msgType MessageType, payload interface{}) (*RPCMessage, error) {
	conn, err := net.DialTimeout("tcp", peer, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(2 * time.Second))

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msg := RPCMessage{
		Type:    msgType,
		From:    n.config.NodeID,
		Payload: payloadBytes,
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(msg); err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(conn)
	var response RPCMessage
	if err := decoder.Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

// replicateEntry replicates a log entry to peers
func (n *Node) replicateEntry(ctx context.Context, entry LogEntry) error {
	// Send to all peers
	for _, peer := range n.config.Peers {
		go n.sendAppendEntries(peer)
	}

	// Wait for commit
	deadline := time.After(n.config.CommitTimeout)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return ErrTimeout
		case <-n.stopCh:
			return ErrShutdown
		default:
		}

		n.mu.RLock()
		if n.commitIndex >= entry.Index {
			n.mu.RUnlock()
			return nil
		}
		n.mu.RUnlock()

		time.Sleep(10 * time.Millisecond)
	}
}

// isLogUpToDate checks if candidate's log is at least as up-to-date as ours
func (n *Node) isLogUpToDate(lastLogIndex, lastLogTerm uint64) bool {
	myLastIndex := uint64(len(n.log))
	var myLastTerm uint64
	if myLastIndex > 0 {
		myLastTerm = n.log[myLastIndex-1].Term
	}

	if lastLogTerm != myLastTerm {
		return lastLogTerm > myLastTerm
	}
	return lastLogIndex >= myLastIndex
}

// randomElectionTimeout returns a random election timeout
func (n *Node) randomElectionTimeout() time.Duration {
	base := n.config.ElectionTimeout
	jitter, _ := rand.Int(rand.Reader, big.NewInt(int64(base)))
	return base + time.Duration(jitter.Int64())
}

// wrapResponse wraps a response in an RPCMessage
func (n *Node) wrapResponse(msgType MessageType, payload interface{}) *RPCMessage {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return &RPCMessage{
		Type:    msgType,
		From:    n.config.NodeID,
		Payload: data,
	}
}

// Stats returns node statistics
func (n *Node) Stats() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return map[string]interface{}{
		"node_id":      n.config.NodeID,
		"state":        n.state.String(),
		"term":         n.currentTerm,
		"leader":       n.leaderId,
		"log_length":   len(n.log),
		"commit_index": n.commitIndex,
		"last_applied": n.lastApplied,
		"peers":        len(n.config.Peers),
	}
}
