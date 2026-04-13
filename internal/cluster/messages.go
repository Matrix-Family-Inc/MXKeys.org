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

package cluster

import (
	"encoding/json"
	"time"

	"mxkeys/internal/zero/log"
)

// handleMessage processes an incoming cluster message.
func (c *Cluster) handleMessage(msg *ClusterMessage) *ClusterMessage {
	switch msg.Type {
	case MsgTypeJoin:
		return c.handleJoin(msg)
	case MsgTypeLeave:
		return c.handleLeave(msg)
	case MsgTypePing:
		return c.handlePing(msg)
	case MsgTypeSync:
		return c.handleSync(msg)
	case MsgTypeSyncReq:
		return c.handleSyncRequest(msg)
	case MsgTypeKeyUpdate:
		return c.handleKeyUpdate(msg)
	default:
		return nil
	}
}

// handleJoin processes a join message.
func (c *Cluster) handleJoin(msg *ClusterMessage) *ClusterMessage {
	var node Node
	if err := json.Unmarshal(msg.Payload, &node); err != nil {
		return nil
	}
	node.LastSeen = time.Now()
	if node.State == NodeStateUnknown {
		node.State = NodeStateHealthy
	}

	c.mu.Lock()
	c.nodes[node.ID] = &node
	c.nodesTotal.Set(float64(len(c.nodes)))
	c.mu.Unlock()

	log.Info("Node joined cluster", "node_id", node.ID, "address", node.Address)

	// Return our node info.
	payload, err := json.Marshal(c.localNode)
	if err != nil {
		return nil
	}
	return &ClusterMessage{
		Type:      MsgTypePong,
		From:      c.nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// handleLeave processes a leave message.
func (c *Cluster) handleLeave(msg *ClusterMessage) *ClusterMessage {
	c.mu.Lock()
	if node, ok := c.nodes[msg.From]; ok {
		node.State = NodeStateLeaving
	}
	c.mu.Unlock()

	log.Info("Node leaving cluster", "node_id", msg.From)
	return nil
}

// handlePing processes a ping message.
func (c *Cluster) handlePing(msg *ClusterMessage) *ClusterMessage {
	c.mu.Lock()
	if node, ok := c.nodes[msg.From]; ok {
		node.LastSeen = time.Now()
		node.State = NodeStateHealthy
	}
	c.mu.Unlock()

	payload, err := json.Marshal(c.localNode)
	if err != nil {
		return nil
	}
	return &ClusterMessage{
		Type:      MsgTypePong,
		From:      c.nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// handleSync processes a full state sync.
func (c *Cluster) handleSync(msg *ClusterMessage) *ClusterMessage {
	var remoteState map[string]map[string]*KeyEntry
	if err := json.Unmarshal(msg.Payload, &remoteState); err != nil {
		return nil
	}

	c.mergeState(remoteState)
	c.syncTotal.Inc()

	return nil
}

// handleSyncRequest responds with our current state.
func (c *Cluster) handleSyncRequest(msg *ClusterMessage) *ClusterMessage {
	c.state.mu.RLock()
	payload, err := json.Marshal(c.state.keys)
	c.state.mu.RUnlock()
	if err != nil {
		return nil
	}

	return &ClusterMessage{
		Type:      MsgTypeSync,
		From:      c.nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}

// handleKeyUpdate processes a key update.
func (c *Cluster) handleKeyUpdate(msg *ClusterMessage) *ClusterMessage {
	var entry KeyEntry
	if err := json.Unmarshal(msg.Payload, &entry); err != nil {
		return nil
	}
	c.storeEntry(&entry, true)

	return nil
}

// mergeState merges remote state with local state using LWW semantics.
func (c *Cluster) mergeState(remote map[string]map[string]*KeyEntry) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	for serverName, keys := range remote {
		if _, ok := c.state.keys[serverName]; !ok {
			c.state.keys[serverName] = make(map[string]*KeyEntry)
		}

		for keyID, entry := range keys {
			if entry == nil {
				continue
			}
			existing := c.state.keys[serverName][keyID]
			if existing == nil || entry.Timestamp.After(existing.Timestamp) {
				c.state.keys[serverName][keyID] = entry
			}
		}
	}
}
