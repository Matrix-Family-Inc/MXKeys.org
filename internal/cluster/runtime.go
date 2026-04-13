/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/raft"
)

type raftCommand struct {
	Type      string    `json:"type"`
	Entry     *KeyEntry `json:"entry,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (c *Cluster) startCRDT() error {
	addr := fmt.Sprintf("%s:%d", c.config.BindAddress, c.config.BindPort)
	listener, err := net.Listen("tcp", addr)
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
	log.Warn("Raft consensus mode is experimental: no persistent log storage, no snapshots; use CRDT for production workloads")
	node := raft.NewNode(raft.Config{
		NodeID:            c.nodeID,
		BindAddress:       c.config.BindAddress,
		BindPort:          c.config.BindPort,
		Peers:             c.config.Seeds,
		ElectionTimeout:   300 * time.Millisecond,
		HeartbeatInterval: 100 * time.Millisecond,
		CommitTimeout:     5 * time.Second,
		SharedSecret:      c.config.SharedSecret,
	})
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

	log.Info("Cluster started",
		"node_id", c.nodeID,
		"bind_address", fmt.Sprintf("%s:%d", c.config.BindAddress, c.config.BindPort),
		"advertise_address", fmt.Sprintf("%s:%d", c.advertiseAddress(), c.advertisePort()),
		"consensus_mode", c.consensusMode(),
	)
	return nil
}

func (c *Cluster) stopRaft() error {
	c.setLocalState(NodeStateLeaving)
	if c.raftNode == nil {
		return nil
	}
	err := c.raftNode.Stop()
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
