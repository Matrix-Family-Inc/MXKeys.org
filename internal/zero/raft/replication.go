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
	if snapshot.prevLogIndex > 0 {
		// termAt consults the snapshot boundary automatically when the
		// prevLogIndex is covered by a compacted prefix.
		if term, ok := n.termAt(snapshot.prevLogIndex); ok {
			snapshot.prevLogTerm = term
		}
	}
	// Copy the tail from nextIdx onward when it is available in memory.
	// A lagging peer whose nextIdx sits below logOffset must be caught up
	// via InstallSnapshot instead; that path is driven by the leader's
	// heartbeat loop.
	if slot, ok := n.sliceIndex(nextIdx); ok {
		snapshot.entries = cloneLogEntries(n.log[slot:])
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
	for i := n.commitIndex + 1; i <= n.logLen(); i++ {
		term, ok := n.termAt(i)
		if !ok || term != n.currentTerm {
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
			entry, ok := n.entryAt(n.lastApplied)
			if !ok {
				// The entry falls below logOffset: it has already been
				// applied via a snapshot install, so skip without invoking
				// the state-machine callback again.
				continue
			}
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
