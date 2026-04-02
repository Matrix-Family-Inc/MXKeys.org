/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Mar 16 2026 UTC
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
	"sync"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/metrics"
)

// NodeState represents the state of a cluster node
type NodeState string

const (
	NodeStateUnknown  NodeState = "unknown"
	NodeStateStarting NodeState = "starting"
	NodeStateHealthy  NodeState = "healthy"
	NodeStateDegraded NodeState = "degraded"
	NodeStateLeaving  NodeState = "leaving"
	NodeStateDead     NodeState = "dead"
)

// ClusterConfig holds cluster configuration
type ClusterConfig struct {
	Enabled       bool
	NodeID        string
	BindAddress   string
	BindPort      int
	Seeds         []string
	ConsensusMode string
	SyncInterval  int
}

// Node represents a cluster node
type Node struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	State     NodeState `json:"state"`
	LastSeen  time.Time `json:"last_seen"`
	Version   string    `json:"version"`
	KeyCount  int64     `json:"key_count"`
	StartedAt time.Time `json:"started_at"`
}

// Cluster manages the distributed notary cluster
type Cluster struct {
	config ClusterConfig
	nodeID string

	mu        sync.RWMutex
	nodes     map[string]*Node
	localNode *Node
	state     *CRDTState

	listener net.Listener
	stopCh   chan struct{}
	wg       sync.WaitGroup

	// Callbacks
	onKeyReceived func(serverName string, data []byte)

	// Metrics
	nodesTotal    *metrics.Gauge
	syncTotal     *metrics.Counter
	messagesTotal *metrics.Counter
}

// CRDTState holds the CRDT-based shared state
type CRDTState struct {
	mu sync.RWMutex

	// LWW-Element-Set for key cache
	// map[serverName]map[keyID]KeyEntry
	keys map[string]map[string]*KeyEntry

	// Vector clock for causality
	vectorClock map[string]int64
}

// KeyEntry represents a cached key in CRDT state
type KeyEntry struct {
	ServerName   string    `json:"server_name"`
	KeyID        string    `json:"key_id"`
	KeyData      string    `json:"key_data"` // base64 encoded
	ValidUntilTS int64     `json:"valid_until_ts"`
	Timestamp    time.Time `json:"timestamp"`
	NodeID       string    `json:"node_id"` // which node added this
	Hash         string    `json:"hash"`
}

// Message types for cluster communication
type MessageType string

const (
	MsgTypeJoin      MessageType = "join"
	MsgTypeLeave     MessageType = "leave"
	MsgTypePing      MessageType = "ping"
	MsgTypePong      MessageType = "pong"
	MsgTypeSync      MessageType = "sync"
	MsgTypeSyncReq   MessageType = "sync_request"
	MsgTypeKeyUpdate MessageType = "key_update"
)

// ClusterMessage represents a message between cluster nodes
type ClusterMessage struct {
	Type      MessageType     `json:"type"`
	From      string          `json:"from"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// NewCluster creates a new cluster instance
func NewCluster(cfg ClusterConfig) (*Cluster, error) {
	if cfg.NodeID == "" {
		cfg.NodeID = generateNodeID()
	}

	c := &Cluster{
		config: cfg,
		nodeID: cfg.NodeID,
		nodes:  make(map[string]*Node),
		state: &CRDTState{
			keys:        make(map[string]map[string]*KeyEntry),
			vectorClock: make(map[string]int64),
		},
		stopCh: make(chan struct{}),
		nodesTotal: metrics.NewGauge(metrics.GaugeOpts{
			Namespace: "mxkeys",
			Subsystem: "cluster",
			Name:      "nodes_total",
			Help:      "Total nodes in cluster",
		}),
		syncTotal: metrics.NewCounter(metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "cluster",
			Name:      "sync_total",
			Help:      "Total sync operations",
		}),
		messagesTotal: metrics.NewCounter(metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "cluster",
			Name:      "messages_total",
			Help:      "Total cluster messages",
		}),
	}

	c.localNode = &Node{
		ID:        cfg.NodeID,
		Address:   cfg.BindAddress,
		Port:      cfg.BindPort,
		State:     NodeStateStarting,
		StartedAt: time.Now(),
	}
	c.nodes[cfg.NodeID] = c.localNode

	return c, nil
}

// Start starts the cluster
func (c *Cluster) Start(ctx context.Context) error {
	if !c.config.Enabled {
		log.Info("Cluster mode disabled")
		return nil
	}

	// Start TCP listener
	addr := fmt.Sprintf("%s:%d", c.config.BindAddress, c.config.BindPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start cluster listener: %w", err)
	}
	c.listener = listener

	log.Info("Cluster started",
		"node_id", c.nodeID,
		"address", addr,
		"consensus_mode", c.config.ConsensusMode,
	)

	// Accept connections
	c.wg.Add(1)
	go c.acceptLoop()

	// Join seed nodes
	c.wg.Add(1)
	go c.joinSeeds()

	// Periodic sync
	c.wg.Add(1)
	go c.syncLoop()

	// Mark as healthy
	c.mu.Lock()
	c.localNode.State = NodeStateHealthy
	c.mu.Unlock()

	return nil
}

// Stop gracefully stops the cluster
func (c *Cluster) Stop() error {
	if !c.config.Enabled {
		return nil
	}

	log.Info("Cluster stopping", "node_id", c.nodeID)

	// Broadcast leave message
	c.broadcastMessage(&ClusterMessage{
		Type:      MsgTypeLeave,
		From:      c.nodeID,
		Timestamp: time.Now(),
	})

	close(c.stopCh)

	if c.listener != nil {
		c.listener.Close()
	}

	c.wg.Wait()

	log.Info("Cluster stopped", "node_id", c.nodeID)
	return nil
}

// acceptLoop accepts incoming connections
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

		go c.handleConnection(conn)
	}
}

// handleConnection handles an incoming connection
func (c *Cluster) handleConnection(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var msg ClusterMessage
		if err := decoder.Decode(&msg); err != nil {
			return
		}

		c.messagesTotal.Inc()

		response := c.handleMessage(&msg)
		if response != nil {
			if err := encoder.Encode(response); err != nil {
				return
			}
		}
	}
}

// handleMessage processes an incoming cluster message
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

// handleJoin processes a join message
func (c *Cluster) handleJoin(msg *ClusterMessage) *ClusterMessage {
	var node Node
	if err := json.Unmarshal(msg.Payload, &node); err != nil {
		return nil
	}

	c.mu.Lock()
	c.nodes[node.ID] = &node
	c.nodesTotal.Set(float64(len(c.nodes)))
	c.mu.Unlock()

	log.Info("Node joined cluster", "node_id", node.ID, "address", node.Address)

	// Return our node info
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

// handleLeave processes a leave message
func (c *Cluster) handleLeave(msg *ClusterMessage) *ClusterMessage {
	c.mu.Lock()
	if node, ok := c.nodes[msg.From]; ok {
		node.State = NodeStateLeaving
	}
	c.mu.Unlock()

	log.Info("Node leaving cluster", "node_id", msg.From)
	return nil
}

// handlePing processes a ping message
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

// handleSync processes a full state sync
func (c *Cluster) handleSync(msg *ClusterMessage) *ClusterMessage {
	var remoteState map[string]map[string]*KeyEntry
	if err := json.Unmarshal(msg.Payload, &remoteState); err != nil {
		return nil
	}

	c.mergeState(remoteState)
	c.syncTotal.Inc()

	return nil
}

// handleSyncRequest responds with our current state
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

// handleKeyUpdate processes a key update
func (c *Cluster) handleKeyUpdate(msg *ClusterMessage) *ClusterMessage {
	var entry KeyEntry
	if err := json.Unmarshal(msg.Payload, &entry); err != nil {
		return nil
	}

	c.state.mu.Lock()
	if _, ok := c.state.keys[entry.ServerName]; !ok {
		c.state.keys[entry.ServerName] = make(map[string]*KeyEntry)
	}

	// LWW merge: keep entry with latest timestamp
	existing := c.state.keys[entry.ServerName][entry.KeyID]
	if existing == nil || entry.Timestamp.After(existing.Timestamp) {
		c.state.keys[entry.ServerName][entry.KeyID] = &entry

		// Notify callback
		if c.onKeyReceived != nil {
			data, err := json.Marshal(entry)
			if err == nil {
				c.onKeyReceived(entry.ServerName, data)
			}
		}
	}
	c.state.mu.Unlock()

	return nil
}

// mergeState merges remote state with local state using LWW semantics
func (c *Cluster) mergeState(remote map[string]map[string]*KeyEntry) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()

	for serverName, keys := range remote {
		if _, ok := c.state.keys[serverName]; !ok {
			c.state.keys[serverName] = make(map[string]*KeyEntry)
		}

		for keyID, entry := range keys {
			existing := c.state.keys[serverName][keyID]
			if existing == nil || entry.Timestamp.After(existing.Timestamp) {
				c.state.keys[serverName][keyID] = entry
			}
		}
	}
}

// joinSeeds connects to seed nodes
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

// connectToNode connects to another cluster node
func (c *Cluster) connectToNode(address string) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send join message
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

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	if err := encoder.Encode(msg); err != nil {
		return err
	}

	// Wait for response
	var response ClusterMessage
	if err := decoder.Decode(&response); err != nil {
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

	// Request state sync
	syncReq := &ClusterMessage{
		Type:      MsgTypeSyncReq,
		From:      c.nodeID,
		Timestamp: time.Now(),
	}
	if err := encoder.Encode(syncReq); err != nil {
		return err
	}

	var syncResp ClusterMessage
	if err := decoder.Decode(&syncResp); err == nil && syncResp.Type == MsgTypeSync {
		c.handleSync(&syncResp)
	}

	return nil
}

// syncLoop periodically syncs state with other nodes
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

// syncWithPeers syncs state with all known peers
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

// pingNode sends a ping to a node
func (c *Cluster) pingNode(address string) {
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	msg := &ClusterMessage{
		Type:      MsgTypePing,
		From:      c.nodeID,
		Timestamp: time.Now(),
	}

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(msg); err != nil {
		log.Debug("Failed to send ping", "address", address, "error", err)
	}
}

// broadcastMessage sends a message to all known nodes
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
		go func(addr string) {
			conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
			if err != nil {
				return
			}
			defer conn.Close()

			encoder := json.NewEncoder(conn)
			if err := encoder.Encode(msg); err != nil {
				log.Debug("Failed to broadcast message", "address", addr, "error", err)
			}
		}(address)
	}
}

// BroadcastKeyUpdate broadcasts a key update to all nodes
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

	// Store locally
	c.state.mu.Lock()
	if _, ok := c.state.keys[serverName]; !ok {
		c.state.keys[serverName] = make(map[string]*KeyEntry)
	}
	c.state.keys[serverName][keyID] = entry
	c.state.mu.Unlock()

	// Broadcast to peers
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

// GetCachedKey returns a cached key from cluster state
func (c *Cluster) GetCachedKey(serverName, keyID string) *KeyEntry {
	c.state.mu.RLock()
	defer c.state.mu.RUnlock()

	if keys, ok := c.state.keys[serverName]; ok {
		return keys[keyID]
	}
	return nil
}

// SetOnKeyReceived sets callback for received keys
func (c *Cluster) SetOnKeyReceived(fn func(serverName string, data []byte)) {
	c.onKeyReceived = fn
}

// Nodes returns list of known nodes
func (c *Cluster) Nodes() []*Node {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]*Node, 0, len(c.nodes))
	for _, node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Stats returns cluster statistics
func (c *Cluster) Stats() map[string]interface{} {
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
		"enabled":        c.config.Enabled,
		"node_id":        c.nodeID,
		"consensus_mode": c.config.ConsensusMode,
		"total_nodes":    nodeCount,
		"healthy_nodes":  healthyCount,
		"cached_servers": serverCount,
		"cached_keys":    keyCount,
	}
}

// generateNodeID generates a unique node ID
func generateNodeID() string {
	data := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// hashKeyEntry creates a hash of a key entry
func hashKeyEntry(serverName, keyID, keyData string) string {
	data := fmt.Sprintf("%s|%s|%s", serverName, keyID, keyData)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
