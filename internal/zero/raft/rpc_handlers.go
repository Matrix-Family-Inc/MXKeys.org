/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
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
	case MsgInstallSnapshot:
		return n.handleInstallSnapshot(msg)
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

	// Check log consistency. prevLog may sit anywhere:
	//   * Exactly at the snapshot boundary: termAt consults snapshotTerm.
	//   * Inside the in-memory window: termAt looks it up in n.log.
	//   * Beyond the tail or below the snapshot prefix: reject (Success=false)
	//     so the leader decrements nextIndex and retries (or sends InstallSnapshot).
	if req.PrevLogIndex > 0 {
		term, ok := n.termAt(req.PrevLogIndex)
		if !ok || term != req.PrevLogTerm {
			return n.wrapResponse(MsgAppendRes, response)
		}
	}

	// Append new entries. On term conflict, truncate the local log and the
	// WAL before accepting the leader's replacement. On each append the WAL
	// is updated before the in-memory slice so a crash between the two
	// cannot lose a record the follower has already acknowledged.
	for i, entry := range req.Entries {
		idx := req.PrevLogIndex + uint64(i) + 1
		existing, inWindow := n.entryAt(idx)
		if inWindow {
			if existing.Term != entry.Term {
				if err := n.truncateLogAfter(idx - 1); err != nil {
					return n.wrapResponse(MsgAppendRes, response)
				}
				if err := n.persistEntry(entry); err != nil {
					return n.wrapResponse(MsgAppendRes, response)
				}
				n.log = append(n.log, entry)
			}
		} else if idx > n.logLen() {
			if err := n.persistEntry(entry); err != nil {
				return n.wrapResponse(MsgAppendRes, response)
			}
			n.log = append(n.log, entry)
		}
		// Else: idx falls under logOffset. Already durably in the snapshot,
		// silently skip.
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
// Uses lastLogIndexTerm so a post-compaction follower with an empty in-memory
// log still reports its durable snapshot boundary rather than a bogus zero.
func (n *Node) isLogUpToDate(lastLogIndex, lastLogTerm uint64) bool {
	myLastIndex, myLastTerm := n.lastLogIndexTerm()

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
