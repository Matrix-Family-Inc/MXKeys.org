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
	"context"
	"encoding/json"
	"time"
)

// sendAppendEntries sends append entries to a peer.
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

// updateCommitIndex updates commit index based on matchIndex.
func (n *Node) updateCommitIndex() {
	for i := n.commitIndex + 1; i <= uint64(len(n.log)); i++ {
		if n.log[i-1].Term != n.currentTerm {
			continue
		}

		matches := 1 // Self.
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

// applyLoop applies committed entries.
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

// replicateEntry replicates a log entry to peers.
func (n *Node) replicateEntry(ctx context.Context, entry LogEntry) error {
	// Send to all peers.
	for _, peer := range n.config.Peers {
		go n.sendAppendEntries(peer)
	}

	// Wait for commit.
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
