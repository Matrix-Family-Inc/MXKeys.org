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
	case MsgPreVote:
		return n.handlePreVote(msg)
	case MsgRequestVote:
		return n.handleRequestVote(msg)
	case MsgAppendEntries:
		return n.handleAppendEntries(msg)
	case MsgInstallSnapshot:
		return n.handleInstallSnapshot(msg)
	case MsgForwardProposal:
		return n.handleForwardProposal(msg)
	default:
		return nil
	}
}

// handlePreVote answers pre-vote probes. It does not mutate n.currentTerm,
// n.votedFor, or n.state. That is the whole point of a pre-vote.
//
// A grant requires both:
//   - the candidate's log is at least as up to date as ours, at the
//     hypothetical term they would campaign with (req.Term);
//   - we have not heard from the current leader recently. The
//     "recently" window is the election timeout; if we have, granting
//     pre-votes would let a partitioned node disrupt a healthy
//     leader.
func (n *Node) handlePreVote(msg *RPCMessage) *RPCMessage {
	var req PreVoteRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		n.mu.RLock()
		term := n.currentTerm
		n.mu.RUnlock()
		return n.wrapResponse(MsgPreVoteRes, PreVoteResponse{Term: term, VoteGranted: false})
	}

	n.mu.RLock()
	defer n.mu.RUnlock()

	resp := PreVoteResponse{Term: n.currentTerm, VoteGranted: false}
	if req.Term < n.currentTerm {
		return n.wrapResponse(MsgPreVoteRes, resp)
	}
	// Refuse while we still believe a leader is reachable. A node that
	// just heard from the leader should not help a campaigner unseat it.
	if time.Since(n.lastContact) < n.config.ElectionTimeout && n.leaderId != "" {
		return n.wrapResponse(MsgPreVoteRes, resp)
	}
	if !n.isLogUpToDate(req.LastLogIndex, req.LastLogTerm) {
		return n.wrapResponse(MsgPreVoteRes, resp)
	}
	resp.VoteGranted = true
	return n.wrapResponse(MsgPreVoteRes, resp)
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
	// Never overwrite a known leaderAddr with an empty one. A mixed-
	// version leader (or any peer that shipped before AdvertiseAddr
	// plumbing) does not populate LeaderAddress; accepting an empty
	// overwrite would strip the follower's forwarding endpoint and
	// leave Propose returning ErrNoLeader even though leadership is
	// healthy. Populated addresses always win over stale ones.
	if req.LeaderAddress != "" {
		n.leaderAddr = req.LeaderAddress
	}
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
