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
	"encoding/json"
	"fmt"
	"net"
	"time"

	"mxkeys/internal/zero/log"
)

// acceptLoop accepts incoming connections.
func (c *Cluster) acceptLoop() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		conn, err := c.listener.Accept()
		if err != nil {
			select {
			case <-c.stopCh:
				return
			default:
				log.Error("Accept error", "error", err)
				continue
			}
		}

		c.wg.Add(1)
		go c.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection.
func (c *Cluster) handleConnection(conn net.Conn) {
	defer c.wg.Done()
	defer conn.Close()

	msg, err := c.readMessage(conn)
	if err != nil {
		return
	}
	if err := c.verifyMessage(msg); err != nil {
		log.Warn("Rejected unsigned or invalid cluster message", "error", err)
		return
	}

	c.messagesTotal.Inc()
	response := c.handleMessage(msg)
	if response == nil {
		return
	}
	if err := c.signMessage(response); err != nil {
		log.Warn("Failed to sign cluster response", "error", err)
		return
	}
	_ = c.writeMessage(conn, response)
}

// joinSeeds connects to seed nodes.
func (c *Cluster) joinSeeds() {
	defer c.wg.Done()

	for _, seed := range c.config.Seeds {
		select {
		case <-c.stopCh:
			return
		default:
		}

		if err := c.connectToNode(seed); err != nil {
			log.Warn("Failed to connect to seed", "seed", seed, "error", err)
		}
	}
}

// connectToNode connects to another cluster node.
func (c *Cluster) connectToNode(address string) error {
	payload, err := json.Marshal(c.localNode)
	if err != nil {
		return fmt.Errorf("failed to marshal local node: %w", err)
	}
	msg := &ClusterMessage{
		Type:      MsgTypeJoin,
		From:      c.nodeID,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	if err := c.signMessage(msg); err != nil {
		return fmt.Errorf("failed to sign join message: %w", err)
	}

	response, err := c.roundTripMessage(address, msg)
	if err != nil {
		return err
	}

	if response.Type == MsgTypePong {
		var node Node
		if err := json.Unmarshal(response.Payload, &node); err != nil {
			return fmt.Errorf("failed to decode pong payload: %w", err)
		}

		c.mu.Lock()
		c.nodes[node.ID] = &node
		c.nodesTotal.Set(float64(len(c.nodes)))
		c.mu.Unlock()

		log.Info("Connected to node", "node_id", node.ID, "address", address)
	}

	// Request state sync.
	syncReq := &ClusterMessage{
		Type:      MsgTypeSyncReq,
		From:      c.nodeID,
		Timestamp: time.Now(),
	}
	if err := c.signMessage(syncReq); err != nil {
		return fmt.Errorf("failed to sign sync request: %w", err)
	}
	if syncResp, err := c.roundTripMessage(address, syncReq); err == nil && syncResp.Type == MsgTypeSync {
		c.handleSync(syncResp)
	}

	return nil
}

// syncLoop periodically syncs state with other nodes.
func (c *Cluster) syncLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(time.Duration(c.config.SyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.syncWithPeers()
		}
	}
}

// syncWithPeers syncs state with all known peers.
func (c *Cluster) syncWithPeers() {
	c.mu.RLock()
	peers := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		if node.ID != c.nodeID && node.State == NodeStateHealthy {
			peers = append(peers, node)
		}
	}
	c.mu.RUnlock()

	for _, peer := range peers {
		address := fmt.Sprintf("%s:%d", peer.Address, peer.Port)
		c.pingNode(address)
	}
}

// pingNode sends a ping to a node.
func (c *Cluster) pingNode(address string) {
	msg := &ClusterMessage{
		Type:      MsgTypePing,
		From:      c.nodeID,
		Timestamp: time.Now(),
	}
	if err := c.signMessage(msg); err != nil {
		log.Debug("Failed to sign ping", "address", address, "error", err)
		return
	}
	resp, err := c.roundTripMessage(address, msg)
	if err != nil {
		log.Debug("Failed to send ping", "address", address, "error", err)
		return
	}
	if resp.Type != MsgTypePong {
		log.Debug("Unexpected ping response", "address", address, "type", resp.Type)
	}
}

// broadcastMessage sends a message to all known nodes.
func (c *Cluster) broadcastMessage(msg *ClusterMessage) {
	c.mu.RLock()
	peers := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		if node.ID != c.nodeID {
			peers = append(peers, node)
		}
	}
	c.mu.RUnlock()

	for _, peer := range peers {
		address := fmt.Sprintf("%s:%d", peer.Address, peer.Port)
		c.wg.Add(1)
		go func(addr string) {
			defer c.wg.Done()
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()

			signed := *msg
			if err := c.signMessage(&signed); err != nil {
				log.Debug("Failed to sign cluster message", "address", addr, "error", err)
				return
			}

			if err := c.writeMessage(conn, &signed); err != nil {
				log.Debug("Failed to broadcast message", "address", addr, "error", err)
			}
		}(address)
	}
}

func (c *Cluster) roundTripMessage(address string, msg *ClusterMessage) (*ClusterMessage, error) {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := c.writeMessage(conn, msg); err != nil {
		return nil, err
	}

	response, err := c.readMessage(conn)
	if err != nil {
		return nil, err
	}
	if err := c.verifyMessage(response); err != nil {
		return nil, fmt.Errorf("failed to verify cluster response: %w", err)
	}
	return response, nil
}
