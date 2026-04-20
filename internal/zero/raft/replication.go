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
	"context"
	"encoding/json"
	"time"
)

type appendEntriesSnapshot struct {
	term         uint64
	nextIdx      uint64
	prevLogIndex uint64
	prevLogTerm  uint64
	leaderCommit uint64
	entries      []LogEntry
}

// sendAppendEntries sends append entries to a peer.
func (n *Node) sendAppendEntries(peer string) {
	defer n.wg.Done()

	snapshot, ok := n.appendEntriesSnapshot(peer)
	if !ok {
		return
	}

	req := AppendEntriesRequest{
		Term:         snapshot.term,
		LeaderId:     n.config.NodeID,
		PrevLogIndex: snapshot.prevLogIndex,
		PrevLogTerm:  snapshot.prevLogTerm,
		Entries:      snapshot.entries,
		LeaderCommit: snapshot.leaderCommit,
	}

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
	if n.state != Leader || n.currentTerm != snapshot.term || n.nextIndex[peer] != snapshot.nextIdx {
		return
	}

	if appendResp.Success {
		n.nextIndex[peer] = snapshot.nextIdx + uint64(len(snapshot.entries))
		n.matchIndex[peer] = n.nextIndex[peer] - 1
		n.updateCommitIndex()
	} else {
		if n.nextIndex[peer] > 1 {
			n.nextIndex[peer]--
		}
	}
}

func (n *Node) appendEntriesSnapshot(peer string) (appendEntriesSnapshot, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.state != Leader {
		return appendEntriesSnapshot{}, false
	}

	nextIdx := n.nextIndex[peer]
	snapshot := appendEntriesSnapshot{
		term:         n.currentTerm,
		nextIdx:      nextIdx,
		prevLogIndex: nextIdx - 1,
		leaderCommit: n.commitIndex,
	}
	if snapshot.prevLogIndex > 0 && snapshot.prevLogIndex <= uint64(len(n.log)) {
		snapshot.prevLogTerm = n.log[snapshot.prevLogIndex-1].Term
	}
	if nextIdx <= uint64(len(n.log)) {
		snapshot.entries = cloneLogEntries(n.log[nextIdx-1:])
	}
	return snapshot, true
}

func cloneLogEntries(entries []LogEntry) []LogEntry {
	if len(entries) == 0 {
		return nil
	}

	cloned := make([]LogEntry, len(entries))
	for i, entry := range entries {
		cloned[i] = LogEntry{
			Index: entry.Index,
			Term:  entry.Term,
		}
		if len(entry.Command) > 0 {
			cloned[i].Command = append(json.RawMessage(nil), entry.Command...)
		}
	}
	return cloned
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
		n.wg.Add(1)
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
