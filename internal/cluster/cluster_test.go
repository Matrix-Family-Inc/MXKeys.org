package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"
)

const testClusterSecret = "cluster-test-secret"

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitFor(t *testing.T, timeout, step time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(step)
	}
	t.Fatalf("condition was not met within %s", timeout)
}

func TestNewClusterInitialState(t *testing.T) {
	c, err := NewCluster(ClusterConfig{
		Enabled:          true,
		NodeID:           "node-a",
		BindAddress:      "127.0.0.1",
		BindPort:         freePort(t),
		AdvertiseAddress: "127.0.0.1",
		ConsensusMode:    "crdt",
		SyncInterval:     1,
		SharedSecret:     testClusterSecret,
	})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	if c.nodeID != "node-a" {
		t.Fatalf("nodeID = %q, want node-a", c.nodeID)
	}
	if c.localNode == nil {
		t.Fatalf("localNode must be initialized")
	}
	if c.localNode.State != NodeStateStarting {
		t.Fatalf("initial local state = %s, want %s", c.localNode.State, NodeStateStarting)
	}
	if got := len(c.Nodes()); got != 1 {
		t.Fatalf("initial node count = %d, want 1", got)
	}
}

func TestNodeIDAndHashDeterminism(t *testing.T) {
	id1 := generateNodeID()
	id2 := generateNodeID()
	if id1 == id2 {
		t.Fatalf("generateNodeID must return unique values")
	}
	if len(id1) != 16 {
		t.Fatalf("generateNodeID length = %d, want 16", len(id1))
	}

	h1 := hashKeyEntry("srv", "key", "data")
	h2 := hashKeyEntry("srv", "key", "data")
	h3 := hashKeyEntry("srv", "key", "other")
	if h1 != h2 {
		t.Fatalf("hash must be deterministic")
	}
	if h1 == h3 {
		t.Fatalf("hash must change with input")
	}
}

func TestClusterMessageSigning(t *testing.T) {
	c, err := NewCluster(ClusterConfig{
		Enabled:          true,
		NodeID:           "node-a",
		BindAddress:      "127.0.0.1",
		BindPort:         freePort(t),
		AdvertiseAddress: "127.0.0.1",
		ConsensusMode:    "crdt",
		SyncInterval:     1,
		SharedSecret:     testClusterSecret,
	})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	msg := &ClusterMessage{
		Type:      MsgTypePing,
		From:      "node-a",
		Timestamp: time.Now(),
		Payload:   json.RawMessage(`{"ok":true}`),
	}
	if err := c.signMessage(msg); err != nil {
		t.Fatalf("signMessage() error = %v", err)
	}
	if err := c.verifyMessage(msg); err != nil {
		t.Fatalf("verifyMessage() error = %v", err)
	}

	msg.Payload = json.RawMessage(`{"ok":false}`)
	if err := c.verifyMessage(msg); err == nil {
		t.Fatal("expected tampered message to fail verification")
	}
}

func TestReadBoundedJSONRejectsOversizedClusterPayload(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxClusterMessageSize+1)
	if _, err := readBoundedJSON(bytes.NewReader(oversized), maxClusterMessageSize); err == nil {
		t.Fatal("expected oversized cluster payload to be rejected")
	}
}

func TestJoinPingLeaveMessageFlow(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "local"})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	remote := Node{ID: "remote", Address: "127.0.0.1", Port: 9999, State: NodeStateStarting}
	payload, _ := json.Marshal(remote)

	resp := c.handleMessage(&ClusterMessage{
		Type:      MsgTypeJoin,
		From:      remote.ID,
		Timestamp: time.Now(),
		Payload:   payload,
	})
	if resp == nil || resp.Type != MsgTypePong {
		t.Fatalf("join must return pong response")
	}
	if got := len(c.Nodes()); got != 2 {
		t.Fatalf("node count after join = %d, want 2", got)
	}

	c.handleMessage(&ClusterMessage{Type: MsgTypePing, From: remote.ID, Timestamp: time.Now()})
	nodes := c.Nodes()
	var got *Node
	for _, n := range nodes {
		if n.ID == remote.ID {
			got = n
			break
		}
	}
	if got == nil || got.State != NodeStateHealthy {
		t.Fatalf("remote node must become healthy after ping")
	}

	c.handleMessage(&ClusterMessage{Type: MsgTypeLeave, From: remote.ID, Timestamp: time.Now()})
	if got.State != NodeStateLeaving {
		t.Fatalf("remote node state = %s, want %s", got.State, NodeStateLeaving)
	}
}

func TestHandleKeyUpdateLWWAndCallback(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "local"})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	callbacks := 0
	c.SetOnKeyReceived(func(serverName string, data []byte) {
		callbacks++
	})

	newest := KeyEntry{
		ServerName: "matrix.org",
		KeyID:      "ed25519:auto",
		KeyData:    "new",
		Timestamp:  time.Now(),
		NodeID:     "n1",
	}
	old := newest
	old.KeyData = "old"
	old.Timestamp = newest.Timestamp.Add(-time.Minute)

	newPayload, _ := json.Marshal(newest)
	c.handleKeyUpdate(&ClusterMessage{Type: MsgTypeKeyUpdate, Payload: newPayload})
	oldPayload, _ := json.Marshal(old)
	c.handleKeyUpdate(&ClusterMessage{Type: MsgTypeKeyUpdate, Payload: oldPayload})

	got := c.GetCachedKey("matrix.org", "ed25519:auto")
	if got == nil {
		t.Fatalf("expected key to be cached")
	}
	if got.KeyData != "new" {
		t.Fatalf("LWW violated: got %q, want %q", got.KeyData, "new")
	}
	if callbacks != 1 {
		t.Fatalf("callback count = %d, want 1 (stale update must not trigger)", callbacks)
	}
}

func TestMergeStatePrefersNewestEntries(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "local"})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	c.state.keys["srv"] = map[string]*KeyEntry{
		"key1": {ServerName: "srv", KeyID: "key1", KeyData: "old", Timestamp: time.Now().Add(-time.Hour)},
	}
	remote := map[string]map[string]*KeyEntry{
		"srv": {
			"key1": {ServerName: "srv", KeyID: "key1", KeyData: "new", Timestamp: time.Now()},
		},
		"other": {
			"k": {ServerName: "other", KeyID: "k", KeyData: "v", Timestamp: time.Now()},
		},
	}

	c.mergeState(remote)

	if got := c.GetCachedKey("srv", "key1"); got == nil || got.KeyData != "new" {
		t.Fatalf("merge did not apply newest value")
	}
	if got := c.GetCachedKey("other", "k"); got == nil || got.KeyData != "v" {
		t.Fatalf("merge did not add missing server/key")
	}
}

func TestHandleSyncRequestReturnsCurrentState(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "local"})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	c.state.keys["srv"] = map[string]*KeyEntry{
		"key": {ServerName: "srv", KeyID: "key", KeyData: "data", Timestamp: time.Now()},
	}

	resp := c.handleSyncRequest(&ClusterMessage{Type: MsgTypeSyncReq})
	if resp == nil || resp.Type != MsgTypeSync {
		t.Fatalf("sync request must return sync response")
	}

	var state map[string]map[string]*KeyEntry
	if err := json.Unmarshal(resp.Payload, &state); err != nil {
		t.Fatalf("failed to decode sync payload: %v", err)
	}
	if state["srv"]["key"].KeyData != "data" {
		t.Fatalf("unexpected sync payload")
	}
}

func TestBroadcastKeyUpdateStoresLocalEntry(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "local"})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}

	c.BroadcastKeyUpdate("srv", "key", "payload", 1234)
	entry := c.GetCachedKey("srv", "key")
	if entry == nil {
		t.Fatalf("key must be cached locally")
	}
	if entry.NodeID != "local" {
		t.Fatalf("entry node_id = %q, want local", entry.NodeID)
	}
	if entry.Hash != hashKeyEntry("srv", "key", "payload") {
		t.Fatalf("entry hash mismatch")
	}
}

func TestBroadcastKeyUpdateDisabledIsNoop(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: false})
	if err != nil {
		t.Fatalf("NewCluster() error = %v", err)
	}
	c.BroadcastKeyUpdate("srv", "key", "payload", 1234)
	if got := c.GetCachedKey("srv", "key"); got != nil {
		t.Fatalf("disabled cluster must not cache broadcasted keys")
	}
}

func TestClusterLifecycleAndSeedJoin(t *testing.T) {
	seedPort := freePort(t)
	joinerPort := freePort(t)

	seed, err := NewCluster(ClusterConfig{
		Enabled:          true,
		NodeID:           "seed",
		BindAddress:      "127.0.0.1",
		BindPort:         seedPort,
		AdvertiseAddress: "127.0.0.1",
		ConsensusMode:    "crdt",
		SyncInterval:     1,
		SharedSecret:     testClusterSecret,
	})
	if err != nil {
		t.Fatalf("seed NewCluster() error = %v", err)
	}

	joiner, err := NewCluster(ClusterConfig{
		Enabled:          true,
		NodeID:           "joiner",
		BindAddress:      "127.0.0.1",
		BindPort:         joinerPort,
		AdvertiseAddress: "127.0.0.1",
		Seeds:            []string{fmt.Sprintf("127.0.0.1:%d", seedPort)},
		ConsensusMode:    "crdt",
		SyncInterval:     1,
		SharedSecret:     testClusterSecret,
	})
	if err != nil {
		t.Fatalf("joiner NewCluster() error = %v", err)
	}

	ctx := context.Background()
	if err := seed.Start(ctx); err != nil {
		t.Fatalf("seed Start() error = %v", err)
	}
	defer func() { _ = seed.Stop() }()

	if err := joiner.Start(ctx); err != nil {
		t.Fatalf("joiner Start() error = %v", err)
	}
	defer func() { _ = joiner.Stop() }()

	waitFor(t, 2*time.Second, 25*time.Millisecond, func() bool {
		return len(joiner.Nodes()) >= 2 && len(seed.Nodes()) >= 2
	})
	if joiner.localNode.State != NodeStateHealthy {
		t.Fatalf("joiner state = %s, want %s", joiner.localNode.State, NodeStateHealthy)
	}
}
