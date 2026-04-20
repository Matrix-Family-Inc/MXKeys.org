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
	"fmt"
	"net"
)

// Start starts the Raft node.
// When a state directory is attached (see SetStateDir), this also loads the
// persisted snapshot and replays the WAL before opening the listener so the
// in-memory state reflects durable history prior to accepting RPCs.
func (n *Node) Start(ctx context.Context) error {
	if n.config.SharedSecret == "" {
		return fmt.Errorf("raft shared secret is required")
	}

	if err := n.LoadFromDisk(); err != nil {
		return fmt.Errorf("raft: load from disk: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", n.config.BindAddress, n.config.BindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	n.listener = listener

	// Accept connections.
	n.wg.Add(1)
	go n.acceptLoop()

	// Run election timer.
	n.wg.Add(1)
	go n.runElectionTimer()

	// Apply committed entries.
	n.wg.Add(1)
	go n.applyLoop()

	go func() {
		<-ctx.Done()
		_ = n.Stop()
	}()

	return nil
}

// Stop stops the Raft node.
func (n *Node) Stop() error {
	n.stopOnce.Do(func() {
		close(n.stopCh)

		if n.listener != nil {
			_ = n.listener.Close()
		}
	})

	n.wg.Wait()
	return nil
}

// Submit submits a command to the Raft cluster.
// Leader-only: the command is appended to the log, persisted to WAL before
// commit (durability precondition for linearizability), then replicated.
//
// Uses logLen() so the assigned Index accounts for any compacted prefix.
func (n *Node) Submit(ctx context.Context, command []byte) error {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return ErrNotLeader
	}

	entry := LogEntry{
		Index:   n.logLen() + 1,
		Term:    n.currentTerm,
		Command: command,
	}
	// Persist BEFORE making the entry visible to replication so a crash
	// immediately after Submit cannot return success for an un-durable entry.
	if err := n.persistEntry(entry); err != nil {
		n.mu.Unlock()
		return fmt.Errorf("raft: persist: %w", err)
	}
	n.log = append(n.log, entry)
	n.updateCommitIndex()
	n.mu.Unlock()

	return n.replicateEntry(ctx, entry)
}

// State returns the current node state.
func (n *Node) State() State {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

// IsLeader returns true if this node is the leader.
func (n *Node) IsLeader() bool {
	return n.State() == Leader
}

// LeaderID returns the current leader ID.
func (n *Node) LeaderID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.leaderId
}

// Term returns the current term.
func (n *Node) Term() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.currentTerm
}

// SetOnStateChange sets callback for state changes.
func (n *Node) SetOnStateChange(fn func(State)) {
	n.onStateChange = fn
}

// SetOnApply sets callback for applied entries.
func (n *Node) SetOnApply(fn func(LogEntry)) {
	n.onApply = fn
}

// Stats returns node statistics.
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
