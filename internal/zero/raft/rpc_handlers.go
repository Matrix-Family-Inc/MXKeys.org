/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package raft

import (
	"encoding/json"
	"time"
)

// handleRPC processes incoming RPC messages.
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

// handleRequestVote handles vote requests.
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

	// Reply false if term < currentTerm.
	if req.Term < n.currentTerm {
		return n.wrapResponse(MsgRequestVoteRes, response)
	}

	// Update term if necessary.
	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.state = Follower
		n.votedFor = ""
	}

	// Check if we can vote.
	if (n.votedFor == "" || n.votedFor == req.CandidateId) && n.isLogUpToDate(req.LastLogIndex, req.LastLogTerm) {
		n.votedFor = req.CandidateId
		n.lastContact = time.Now()
		response.VoteGranted = true
	}

	response.Term = n.currentTerm
	return n.wrapResponse(MsgRequestVoteRes, response)
}

// handleAppendEntries handles append entries requests.
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

	// Reply false if term < currentTerm.
	if req.Term < n.currentTerm {
		return n.wrapResponse(MsgAppendRes, response)
	}

	// Update term and state.
	if req.Term > n.currentTerm {
		n.currentTerm = req.Term
		n.votedFor = ""
	}

	n.state = Follower
	n.leaderId = req.LeaderId
	n.lastContact = time.Now()

	// Check log consistency.
	if req.PrevLogIndex > 0 {
		if req.PrevLogIndex > uint64(len(n.log)) {
			return n.wrapResponse(MsgAppendRes, response)
		}
		if n.log[req.PrevLogIndex-1].Term != req.PrevLogTerm {
			return n.wrapResponse(MsgAppendRes, response)
		}
	}

	// Append new entries.
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

	// Update commit index.
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

// isLogUpToDate checks if candidate's log is at least as up-to-date as ours.
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

// wrapResponse wraps a response in an RPCMessage.
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
