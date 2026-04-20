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

	"mxkeys/internal/zero/nettls"
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
	listener, err := nettls.Listen("tcp", addr, n.config.TLS)
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
//
// Durability + group-commit contract:
//  1. Under n.mu we atomically reserve the next index AND append to
//     n.log. This keeps log indices strictly monotonic even under
//     concurrent Submits (without this step two callers could compute
//     the same n.logLen()+1 and collide).
//  2. We drop n.mu and block on WAL persistence. The WAL batcher
//     amortizes the fsync across every Submit (and every follower-side
//     append) whose window overlaps, so throughput grows with load
//     instead of being capped by per-call fsync latency.
//  3. On persist success we re-acquire n.mu, advance commitIndex, and
//     replicate. On persist failure we re-acquire n.mu and truncate
//     the in-memory entry (WAL never saw it), then return the error
//     so the caller knows the submission did not land.
//
// Leadership change during step 2 is handled at commit time: the new
// leader's AppendEntries will overwrite our in-memory tail and the
// non-durable entry disappears.
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
	// Publish into n.log under the lock to serialize index assignment.
	// The entry is visible to replication; commitIndex will not advance
	// past it until persist succeeds (enforced by the post-persist
	// updateCommitIndex call below, which is the only caller on the
	// leader's write path).
	n.log = append(n.log, entry)
	publishedLen := uint64(len(n.log))
	n.mu.Unlock()

	// Persist outside the lock. Concurrent Submits flow through the WAL
	// batcher and share a single fsync per flush window.
	if perr := n.persistEntry(entry); perr != nil {
		n.mu.Lock()
		// Truncate only if the tail we appended is still on top.
		// A fresh leader's AppendEntries could have already rewritten
		// the tail; leave that state alone.
		if uint64(len(n.log)) == publishedLen &&
			len(n.log) > 0 &&
			n.log[len(n.log)-1].Index == entry.Index {
			n.log = n.log[:len(n.log)-1]
		}
		n.mu.Unlock()
		return fmt.Errorf("raft: persist: %w", perr)
	}

	n.mu.Lock()
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
