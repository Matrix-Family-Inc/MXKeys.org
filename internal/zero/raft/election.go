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
	"crypto/rand"
	"encoding/json"
	"math/big"
	"time"
)

// runElectionTimer runs the election timeout timer.
//
// When the timer expires without contact from a leader, the node first
// probes the cluster with a pre-vote round. Only if a majority of
// peers would grant a real vote does the node promote to Candidate and
// start a real election (which bumps currentTerm). This prevents a
// partitioned node from disrupting a stable leader just because its
// own term counter ticked while it was unreachable.
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

		n.mu.RLock()
		shouldStart := n.state != Leader && time.Since(n.lastContact) >= timeout
		n.mu.RUnlock()
		if !shouldStart {
			continue
		}
		if n.runPreVote() {
			n.startElection()
		}
	}
}

// runPreVote performs a pre-vote round. Returns true if a majority of
// peers (including self, which always agrees) would grant a vote at
// the hypothetical next term. The node state is NOT mutated during
// this phase.
func (n *Node) runPreVote() bool {
	n.mu.RLock()
	peers := append([]string(nil), n.config.Peers...)
	nextTerm := n.currentTerm + 1
	lastLogIndex, lastLogTerm := n.lastLogIndexTerm()
	electionTimeout := n.config.ElectionTimeout
	n.mu.RUnlock()

	if len(peers) == 0 {
		// Single-node degenerate case; no pre-vote necessary.
		return true
	}

	needed := (len(peers)+1)/2 + 1
	grants := 1 // self

	results := make(chan bool, len(peers))
	for _, peer := range peers {
		go func(p string) {
			ok := n.sendPreVote(p, nextTerm, lastLogIndex, lastLogTerm)
			results <- ok
		}(peer)
	}

	deadline := time.After(electionTimeout)
	for i := 0; i < len(peers); i++ {
		select {
		case granted := <-results:
			if granted {
				grants++
				if grants >= needed {
					return true
				}
			}
		case <-deadline:
			return grants >= needed
		case <-n.stopCh:
			return false
		}
	}
	return grants >= needed
}

// sendPreVote dispatches a single pre-vote probe. Unlike requestVote,
// does NOT update local state even if the peer's term response is
// higher than ours. Pre-vote is informational only.
func (n *Node) sendPreVote(peer string, term, lastLogIndex, lastLogTerm uint64) bool {
	req := PreVoteRequest{
		Term:         term,
		CandidateId:  n.config.NodeID,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
	resp, err := n.sendRPC(peer, MsgPreVote, req)
	if err != nil {
		return false
	}
	var voteResp PreVoteResponse
	if err := json.Unmarshal(resp.Payload, &voteResp); err != nil {
		return false
	}
	// If the peer's currentTerm is higher than our hypothetical term,
	// our pre-vote is irrelevant: we will not win an election at that
	// term anyway.
	if voteResp.Term > term {
		return false
	}
	return voteResp.VoteGranted
}

// startElection starts a new election.
func (n *Node) startElection() {
	n.mu.Lock()
	n.currentTerm++
	n.state = Candidate
	n.votedFor = n.config.NodeID
	n.lastContact = time.Now()
	currentTerm := n.currentTerm
	peers := append([]string(nil), n.config.Peers...)
	lastLogIndex, lastLogTerm := n.lastLogIndexTerm()
	stateChange := n.onStateChange
	electionTimeout := n.config.ElectionTimeout
	n.mu.Unlock()

	if stateChange != nil {
		go stateChange(Candidate)
	}

	// Request votes from peers.
	votes := 1 // Vote for self.
	voteCh := make(chan bool, len(peers))
	needed := (len(peers)+1)/2 + 1

	if votes >= needed {
		n.mu.Lock()
		if n.state == Candidate && n.currentTerm == currentTerm {
			n.becomeLeader()
		}
		n.mu.Unlock()
		return
	}

	for _, peer := range peers {
		n.wg.Add(1)
		go func(p string, term uint64, logIndex uint64, logTerm uint64) {
			defer n.wg.Done()
			granted := n.requestVote(p, term, logIndex, logTerm)
			voteCh <- granted
		}(peer, currentTerm, lastLogIndex, lastLogTerm)
	}

	// Collect votes.
	for i := 0; i < len(peers); i++ {
		select {
		case granted := <-voteCh:
			if granted {
				votes++
			}
		case <-time.After(electionTimeout):
		case <-n.stopCh:
			return
		}

		if votes >= needed {
			n.mu.Lock()
			if n.state == Candidate && n.currentTerm == currentTerm {
				n.becomeLeader()
			}
			n.mu.Unlock()
			return
		}
	}
}

// becomeLeader transitions to leader state.
func (n *Node) becomeLeader() {
	n.state = Leader
	n.leaderId = n.config.NodeID

	// Initialize leader state.
	for _, peer := range n.config.Peers {
		n.nextIndex[peer] = uint64(len(n.log)) + 1
		n.matchIndex[peer] = 0
	}

	if n.onStateChange != nil {
		go n.onStateChange(Leader)
	}

	// Start heartbeat.
	if len(n.config.Peers) > 0 {
		n.wg.Add(1)
		go n.sendHeartbeats()
	}
}

// sendHeartbeats sends periodic heartbeats to all peers.
func (n *Node) sendHeartbeats() {
	defer n.wg.Done()
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
				n.wg.Add(1)
				go n.sendAppendEntries(peer)
			}
		}
	}
}

// requestVote sends a vote request to a peer.
func (n *Node) requestVote(peer string, term uint64, lastLogIndex uint64, lastLogTerm uint64) bool {
	req := RequestVoteRequest{
		Term:         term,
		CandidateId:  n.config.NodeID,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

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

// randomElectionTimeout returns a random election timeout.
func (n *Node) randomElectionTimeout() time.Duration {
	base := n.config.ElectionTimeout
	jitter, _ := rand.Int(rand.Reader, big.NewInt(int64(base)))
	return base + time.Duration(jitter.Int64())
}
