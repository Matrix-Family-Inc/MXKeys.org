package cluster

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestNewCluster(t *testing.T) {
	cfg := ClusterConfig{
		Enabled:       true,
		NodeID:        "test-node-1",
		BindAddress:   "127.0.0.1",
		BindPort:      7946,
		ConsensusMode: "crdt",
		SyncInterval:  5,
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	if c.nodeID != "test-node-1" {
		t.Errorf("nodeID = %q, want test-node-1", c.nodeID)
	}

	if c.localNode == nil {
		t.Error("localNode should not be nil")
	}
}

func TestClusterAutoNodeID(t *testing.T) {
	cfg := ClusterConfig{
		Enabled: true,
		// No NodeID specified
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	if c.nodeID == "" {
		t.Error("should auto-generate nodeID")
	}

	if len(c.nodeID) != 16 {
		t.Errorf("nodeID length = %d, want 16", len(c.nodeID))
	}
}

func TestNodeState(t *testing.T) {
	node := &Node{
		ID:        "node-1",
		Address:   "192.168.1.1",
		Port:      7946,
		State:     NodeStateStarting,
		StartedAt: time.Now(),
	}

	if node.State != NodeStateStarting {
		t.Errorf("state = %v, want starting", node.State)
	}

	node.State = NodeStateHealthy
	if node.State != NodeStateHealthy {
		t.Errorf("state = %v, want healthy", node.State)
	}
}

func TestClusterStats(t *testing.T) {
	cfg := ClusterConfig{
		Enabled:       true,
		NodeID:        "stats-node",
		BindPort:      7947,
		ConsensusMode: "crdt",
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	stats := c.Stats()

	if stats["enabled"] != true {
		t.Error("enabled should be true")
	}

	if stats["node_id"] != "stats-node" {
		t.Errorf("node_id = %v, want stats-node", stats["node_id"])
	}

	if stats["consensus_mode"] != "crdt" {
		t.Errorf("consensus_mode = %v, want crdt", stats["consensus_mode"])
	}

	if stats["total_nodes"].(int) != 1 {
		t.Errorf("total_nodes = %v, want 1", stats["total_nodes"])
	}
}

func TestKeyEntry(t *testing.T) {
	entry := &KeyEntry{
		ServerName:   "matrix.org",
		KeyID:        "ed25519:key1",
		KeyData:      "base64encodedkey",
		ValidUntilTS: time.Now().Add(24 * time.Hour).UnixMilli(),
		Timestamp:    time.Now(),
		NodeID:       "node-1",
		Hash:         hashKeyEntry("matrix.org", "ed25519:key1", "base64encodedkey"),
	}

	if entry.ServerName != "matrix.org" {
		t.Errorf("ServerName = %q, want matrix.org", entry.ServerName)
	}

	if entry.Hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestClusterMessage(t *testing.T) {
	msg := &ClusterMessage{
		Type:      MsgTypeJoin,
		From:      "node-1",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed ClusterMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Type != MsgTypeJoin {
		t.Errorf("type = %v, want join", parsed.Type)
	}

	if parsed.From != "node-1" {
		t.Errorf("from = %q, want node-1", parsed.From)
	}
}

func TestCRDTState(t *testing.T) {
	state := &CRDTState{
		keys:        make(map[string]map[string]*KeyEntry),
		vectorClock: make(map[string]int64),
	}

	// Add a key
	entry := &KeyEntry{
		ServerName: "test.server",
		KeyID:      "ed25519:key1",
		KeyData:    "keydata",
		Timestamp:  time.Now(),
		NodeID:     "node-1",
	}

	state.keys["test.server"] = make(map[string]*KeyEntry)
	state.keys["test.server"]["ed25519:key1"] = entry

	if len(state.keys) != 1 {
		t.Errorf("keys count = %d, want 1", len(state.keys))
	}

	if state.keys["test.server"]["ed25519:key1"].NodeID != "node-1" {
		t.Error("key entry node ID mismatch")
	}
}

func TestLWWMerge(t *testing.T) {
	state := &CRDTState{
		keys:        make(map[string]map[string]*KeyEntry),
		vectorClock: make(map[string]int64),
	}

	oldTime := time.Now().Add(-time.Hour)
	newTime := time.Now()

	// Old entry
	state.keys["server"] = make(map[string]*KeyEntry)
	state.keys["server"]["key1"] = &KeyEntry{
		ServerName: "server",
		KeyID:      "key1",
		KeyData:    "old_data",
		Timestamp:  oldTime,
	}

	// New entry should win
	newEntry := &KeyEntry{
		ServerName: "server",
		KeyID:      "key1",
		KeyData:    "new_data",
		Timestamp:  newTime,
	}

	if newEntry.Timestamp.After(state.keys["server"]["key1"].Timestamp) {
		state.keys["server"]["key1"] = newEntry
	}

	if state.keys["server"]["key1"].KeyData != "new_data" {
		t.Error("LWW merge failed: newer entry should win")
	}
}

func TestGenerateNodeID(t *testing.T) {
	id1 := generateNodeID()
	id2 := generateNodeID()

	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}

	if len(id1) != 16 {
		t.Errorf("ID length = %d, want 16", len(id1))
	}
}

func TestHashKeyEntry(t *testing.T) {
	hash1 := hashKeyEntry("server", "key", "data")
	hash2 := hashKeyEntry("server", "key", "data")
	hash3 := hashKeyEntry("server", "key", "different")

	if hash1 != hash2 {
		t.Error("same input should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("different input should produce different hash")
	}

	if len(hash1) != 64 {
		t.Errorf("hash length = %d, want 64 (sha256 hex)", len(hash1))
	}
}

func TestClusterNodes(t *testing.T) {
	cfg := ClusterConfig{
		Enabled:  true,
		NodeID:   "nodes-test",
		BindPort: 7948,
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	nodes := c.Nodes()

	if len(nodes) != 1 {
		t.Errorf("nodes count = %d, want 1", len(nodes))
	}

	if nodes[0].ID != "nodes-test" {
		t.Errorf("node ID = %q, want nodes-test", nodes[0].ID)
	}
}

func TestBroadcastKeyUpdateDisabled(t *testing.T) {
	cfg := ClusterConfig{
		Enabled: false,
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	// Should not panic when disabled
	c.BroadcastKeyUpdate("server", "key", "data", time.Now().UnixMilli())
}

func TestGetCachedKey(t *testing.T) {
	cfg := ClusterConfig{
		Enabled:  true,
		NodeID:   "cache-test",
		BindPort: 7949,
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	// No key initially
	entry := c.GetCachedKey("server", "key")
	if entry != nil {
		t.Error("should return nil for non-existent key")
	}

	// Add key
	c.state.mu.Lock()
	c.state.keys["server"] = make(map[string]*KeyEntry)
	c.state.keys["server"]["key"] = &KeyEntry{
		ServerName: "server",
		KeyID:      "key",
		KeyData:    "data",
	}
	c.state.mu.Unlock()

	// Now should exist
	entry = c.GetCachedKey("server", "key")
	if entry == nil {
		t.Error("should return cached key")
	}
	if entry.KeyData != "data" {
		t.Errorf("KeyData = %q, want data", entry.KeyData)
	}
}

func TestClusterStartStop(t *testing.T) {
	port := getFreePort(t)

	cfg := ClusterConfig{
		Enabled:       true,
		NodeID:        "lifecycle-test",
		BindAddress:   "127.0.0.1",
		BindPort:      port,
		ConsensusMode: "crdt",
		SyncInterval:  1,
	}

	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	ctx := context.Background()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("failed to start cluster: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if c.localNode.State != NodeStateHealthy {
		t.Errorf("state = %v, want healthy", c.localNode.State)
	}

	if err := c.Stop(); err != nil {
		t.Errorf("failed to stop cluster: %v", err)
	}
}
