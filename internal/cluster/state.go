/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"mxkeys/internal/zero/log"
)

// BroadcastKeyUpdate publishes a key update to the cluster.
//
// Consensus contract:
//
//   - In "crdt" mode the update is eagerly applied to the local LWW
//     state and gossiped to peers. Every node is free to write; LWW
//     on (Timestamp, Hash) resolves conflicts.
//   - In "raft" mode writes go through the replicated log. The local
//     cache is NOT populated before Submit: if this node is not the
//     leader, or Submit fails for any reason, we must not expose a
//     cached entry that never replicated. The apply callback wired in
//     startRaft is the single path that writes into c.state.keys; it
//     runs on every replica (including the leader) once the entry is
//     committed.
func (c *Cluster) BroadcastKeyUpdate(serverName, keyID, keyData string, validUntilTS int64) {
	if !c.config.Enabled {
		return
	}

	entry := &KeyEntry{
		ServerName:   serverName,
		KeyID:        keyID,
		KeyData:      keyData,
		ValidUntilTS: validUntilTS,
		Timestamp:    time.Now(),
		NodeID:       c.nodeID,
		Hash:         hashKeyEntry(serverName, keyID, keyData),
	}

	if c.consensusMode() == "raft" {
		if c.raftNode == nil {
			return
		}
		command, err := json.Marshal(raftCommand{
			Type:      "key_update",
			Entry:     entry,
			Timestamp: time.Now(),
		})
		if err != nil {
			log.Warn("Failed to marshal raft key update", "server", serverName, "key_id", keyID, "error", err)
			return
		}
		// Strict Raft semantics: do not touch local state until the
		// entry is committed. Submit only succeeds on the leader; on a
		// follower it returns ErrNotLeader and we surface that via a
		// warning. The committed entry reaches every replica through
		// the apply callback registered in startRaft.
		if err := c.raftNode.Submit(context.Background(), command); err != nil {
			log.Warn("Failed to replicate key update via raft", "server", serverName, "key_id", keyID, "error", err)
		}
		return
	}

	// CRDT path: local-first write plus gossip.
	c.storeEntry(entry, false)

	payload, err := json.Marshal(entry)
	if err != nil {
		log.Warn("Failed to marshal key update", "server", serverName, "key_id", keyID, "error", err)
		return
	}
	c.broadcastMessage(&ClusterMessage{
		Type:      MsgTypeKeyUpdate,
		From:      c.nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	})
}

// GetCachedKey returns a cached key from cluster state.
func (c *Cluster) GetCachedKey(serverName, keyID string) *KeyEntry {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()

	if keys, ok := c.state.keys[serverName]; ok {
		if entry := keys[keyID]; entry != nil {
			copied := *entry
			return &copied
		}
	}
	return nil
}

// SetOnKeyReceived sets callback for received keys.
func (c *Cluster) SetOnKeyReceived(fn func(serverName string, data []byte)) {
	c.onKeyReceived = fn
}

// Nodes returns list of known nodes.
func (c *Cluster) Nodes() []*Node {
	if c.consensusMode() == "raft" {
		return c.raftNodes()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Stats returns cluster statistics.
func (c *Cluster) Stats() map[string]interface{} {
	if c.consensusMode() == "raft" && c.raftNode != nil {
		stats := c.raftNode.Stats()
		stats["enabled"] = c.config.Enabled
		stats["consensus_mode"] = c.consensusMode()
		stats["advertise_address"] = c.advertiseAddress()
		stats["advertise_port"] = c.advertisePort()
		stats["cached_servers"] = c.cachedServerCount()
		stats["cached_keys"] = c.cachedKeyCount()
		return stats
	}

	c.mu.RLock()
	nodeCount := len(c.nodes)
	healthyCount := 0
	for _, node := range c.nodes {
		if node.State == NodeStateHealthy {
			healthyCount++
		}
	}
	c.mu.RUnlock()

	c.state.mu.RLock()
	keyCount := 0
	serverCount := len(c.state.keys)
	for _, keys := range c.state.keys {
		keyCount += len(keys)
	}
	c.state.mu.RUnlock()

	return map[string]interface{}{
		"enabled":           c.config.Enabled,
		"node_id":           c.nodeID,
		"consensus_mode":    c.consensusMode(),
		"total_nodes":       nodeCount,
		"healthy_nodes":     healthyCount,
		"cached_servers":    serverCount,
		"cached_keys":       keyCount,
		"advertise_address": c.advertiseAddress(),
		"advertise_port":    c.advertisePort(),
	}
}

func (c *Cluster) storeEntry(entry *KeyEntry, notify bool) {
	c.state.mu.Lock()
	if _, ok := c.state.keys[entry.ServerName]; !ok {
		c.state.keys[entry.ServerName] = make(map[string]*KeyEntry)
	}

	existing := c.state.keys[entry.ServerName][entry.KeyID]
	if existing == nil || entry.Timestamp.After(existing.Timestamp) {
		c.state.keys[entry.ServerName][entry.KeyID] = entry
	}
	c.state.mu.Unlock()

	if notify && c.onKeyReceived != nil && (existing == nil || entry.Timestamp.After(existing.Timestamp)) {
		data, err := json.Marshal(entry)
		if err == nil {
			c.onKeyReceived(entry.ServerName, data)
		}
	}
}

func (c *Cluster) raftNodes() []*Node {
	c.mu.RLock()
	local := *c.localNode
	c.mu.RUnlock()

	nodes := []*Node{&local}
	for _, peer := range c.config.Seeds {
		host, port, err := net.SplitHostPort(peer)
		if err != nil {
			host = peer
			port = "0"
		}
		portNum := 0
		// Sscanf failure leaves portNum at 0, which is the intended
		// sentinel for unparseable seed ports. Explicit discard quiets
		// errcheck without changing behavior.
		_, _ = fmt.Sscanf(port, "%d", &portNum)
		nodes = append(nodes, &Node{
			ID:       peer,
			Address:  host,
			Port:     portNum,
			State:    NodeStateUnknown,
			LastSeen: time.Time{},
		})
	}
	return nodes
}

func (c *Cluster) cachedServerCount() int {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	return len(c.state.keys)
}

func (c *Cluster) cachedKeyCount() int {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()
	total := 0
	for _, keys := range c.state.keys {
		total += len(keys)
	}
	return total
}

// hashKeyEntry creates a hash of a key entry.
func hashKeyEntry(serverName, keyID, keyData string) string {
	data := fmt.Sprintf("%s|%s|%s", serverName, keyID, keyData)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
