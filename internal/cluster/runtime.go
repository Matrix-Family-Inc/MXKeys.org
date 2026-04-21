/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Updated
 */

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/nettls"
	"mxkeys/internal/zero/raft"
)

type raftCommand struct {
	Type      string    `json:"type"`
	Entry     *KeyEntry `json:"entry,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (c *Cluster) startCRDT() error {
	addr := fmt.Sprintf("%s:%d", c.config.BindAddress, c.config.BindPort)
	listener, err := nettls.Listen("tcp", addr, c.config.TLS)
	if err != nil {
		return fmt.Errorf("failed to start cluster listener: %w", err)
	}
	c.listener = listener

	c.setLocalState(NodeStateHealthy)

	log.Info("Cluster started",
		"node_id", c.nodeID,
		"bind_address", addr,
		"advertise_address", fmt.Sprintf("%s:%d", c.advertiseAddress(), c.advertisePort()),
		"consensus_mode", c.consensusMode(),
		"tls_enabled", c.config.TLS.Enabled,
	)

	c.wg.Add(1)
	go c.acceptLoop()
	c.wg.Add(1)
	go c.joinSeeds()
	c.wg.Add(1)
	go c.syncLoop()
	return nil
}

func (c *Cluster) stopCRDT() error {
	c.setLocalState(NodeStateLeaving)
	c.broadcastMessage(&ClusterMessage{
		Type:      MsgTypeLeave,
		From:      c.nodeID,
		Timestamp: time.Now(),
	})

	close(c.stopCh)
	if c.listener != nil {
		_ = c.listener.Close()
	}
	c.wg.Wait()
	log.Info("Cluster stopped", "node_id", c.nodeID, "consensus_mode", c.consensusMode())
	return nil
}

func (c *Cluster) startRaft(ctx context.Context) error {
	node := raft.NewNode(raft.Config{
		NodeID:            c.nodeID,
		BindAddress:       c.config.BindAddress,
		BindPort:          c.config.BindPort,
		Peers:             c.config.Seeds,
		ElectionTimeout:   300 * time.Millisecond,
		HeartbeatInterval: 100 * time.Millisecond,
		CommitTimeout:     5 * time.Second,
		SharedSecret:      c.config.SharedSecret,
		TLS:               c.config.TLS,
	})

	// Attach persistent state so committed log entries survive restart.
	// RaftStateDir is mandatory for consensus_mode=raft; Validate()
	// enforces this, and we reject the empty case here too as a
	// defense-in-depth guard against callers that bypass config validation.
	if c.config.RaftStateDir == "" {
		return fmt.Errorf("cluster.raft_state_dir is required when cluster.consensus_mode=raft")
	}
	if err := node.SetStateDir(c.config.RaftStateDir, c.config.RaftSyncOnAppend); err != nil {
		return fmt.Errorf("failed to attach raft state dir %q: %w", c.config.RaftStateDir, err)
	}
	// Wire state-machine snapshot callbacks BEFORE Start so that
	// LoadFromDisk (invoked inside Start) can restore the cache from a
	// persisted snapshot, and so that log compaction has a provider to
	// call when it runs.
	node.SetSnapshotProvider(c.snapshotKeyState)
	node.SetSnapshotInstaller(c.installKeySnapshot)

	node.SetOnStateChange(func(state raft.State) {
		switch state {
		case raft.Leader, raft.Follower:
			c.setLocalState(NodeStateHealthy)
		case raft.Candidate:
			c.setLocalState(NodeStateStarting)
		default:
			c.setLocalState(NodeStateUnknown)
		}
	})
	node.SetOnApply(func(entry raft.LogEntry) {
		var cmd raftCommand
		if err := json.Unmarshal(entry.Command, &cmd); err != nil {
			log.Warn("Failed to decode raft command", "error", err)
			return
		}
		switch cmd.Type {
		case "key_update":
			if cmd.Entry != nil {
				c.storeEntry(cmd.Entry, true)
			}
		}
	})
	if err := node.Start(ctx); err != nil {
		return fmt.Errorf("failed to start raft node: %w", err)
	}
	c.raftNode = node
	c.setLocalState(NodeStateHealthy)

	c.wg.Add(1)
	go c.raftCompactionLoop(ctx)

	log.Info("Cluster started",
		"node_id", c.nodeID,
		"bind_address", fmt.Sprintf("%s:%d", c.config.BindAddress, c.config.BindPort),
		"advertise_address", fmt.Sprintf("%s:%d", c.advertiseAddress(), c.advertisePort()),
		"consensus_mode", c.consensusMode(),
		"tls_enabled", c.config.TLS.Enabled,
		"raft_state_dir", c.config.RaftStateDir,
	)
	return nil
}

func (c *Cluster) stopRaft() error {
	c.setLocalState(NodeStateLeaving)
	// Signal the compaction loop to wind down even when Stop() is
	// invoked independently of the startup context cancellation. The
	// Cluster uses stopOnce so a second close here cannot race with
	// stopCRDT on the same instance.
	close(c.stopCh)
	if c.raftNode == nil {
		c.wg.Wait()
		return nil
	}
	err := c.raftNode.Stop()
	c.wg.Wait()
	log.Info("Cluster stopped", "node_id", c.nodeID, "consensus_mode", c.consensusMode())
	return err
}

func (c *Cluster) setLocalState(state NodeState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.localNode != nil {
		c.localNode.State = state
		c.localNode.LastSeen = time.Now()
	}
}
