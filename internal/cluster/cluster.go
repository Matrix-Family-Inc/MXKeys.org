/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
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
	"mxkeys/internal/zero/nettls"
	"mxkeys/internal/zero/raft"
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
	Enabled          bool
	NodeID           string
	BindAddress      string
	BindPort         int
	AdvertiseAddress string
	AdvertisePort    int
	Seeds            []string
	ConsensusMode    string
	SyncInterval     int
	SharedSecret     string
	// RaftStateDir is the directory holding the Raft WAL and snapshot.
	// Required when ConsensusMode="raft" for crash-safe replication.
	RaftStateDir string
	// RaftSyncOnAppend fsyncs the Raft WAL after each append. Default true.
	RaftSyncOnAppend bool

	// TLS configures transport-level encryption and mutual
	// authentication for cluster traffic (both CRDT and Raft).
	// When TLS.Enabled is false the transport stays on plain TCP
	// (backward-compatible default). Mutual authentication
	// (RequireClientCert=true) is strongly recommended in production.
	TLS nettls.Config
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
	stopOnce sync.Once
	raftNode *raft.Node
	replayMu sync.Mutex
	seenMACs map[string]time.Time

	// Callbacks
	onKeyReceived func(serverName string, data []byte)

	// Metrics
	nodesTotal    *metrics.Gauge
	syncTotal     *metrics.Counter
	messagesTotal *metrics.Counter
}

// CRDTState holds the shared key cache plus the raft bookkeeping
// that must stay consistent with it.
//
// Lock discipline:
//
//   - mu serialises both fields. Every field here MUST be read or
//     written while holding mu (at least RLock for reads, Lock for
//     writes). No field escapes via pointer outside critical
//     sections.
//   - raftLastApplied tracks the highest raft log index whose
//     onApply callback has committed its mutations to keys. Pairing
//     it with keys under the same mu is what gives the snapshot
//     provider atomic (payload, index) captures; without that
//     pairing a snapshot's LastIncludedIndex could lag the payload
//     by an arbitrary number of applied entries.
//   - raftLastApplied is only used by the raft consensus mode. In
//     CRDT mode it stays at zero.
type CRDTState struct {
	mu sync.RWMutex

	// LWW key cache state: map[serverName]map[keyID]KeyEntry.
	keys map[string]map[string]*KeyEntry

	// raftLastApplied is the highest Raft log index whose apply
	// callback has written its KeyEntry into keys. Updated from
	// startRaft's onApply under mu together with the keys mutation.
	raftLastApplied uint64
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
	Signature string          `json:"signature,omitempty"`
}

// NewCluster creates a new cluster instance
func NewCluster(cfg ClusterConfig) (*Cluster, error) {
	if cfg.NodeID == "" {
		cfg.NodeID = generateNodeID()
	}
	if cfg.ConsensusMode == "" {
		cfg.ConsensusMode = "crdt"
	}

	c := &Cluster{
		config: cfg,
		nodeID: cfg.NodeID,
		nodes:  make(map[string]*Node),
		state: &CRDTState{
			keys: make(map[string]map[string]*KeyEntry),
		},
		stopCh:   make(chan struct{}),
		seenMACs: make(map[string]time.Time),
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
		Address:   c.advertiseAddress(),
		Port:      c.advertisePort(),
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
	if c.config.SharedSecret == "" {
		return fmt.Errorf("cluster shared secret is required")
	}

	stopCtx := ctx
	go func() {
		<-stopCtx.Done()
		_ = c.Stop()
	}()

	switch c.consensusMode() {
	case "raft":
		return c.startRaft(ctx)
	case "crdt":
		return c.startCRDT()
	default:
		return fmt.Errorf("unsupported consensus mode: %s", c.config.ConsensusMode)
	}
}

// Stop gracefully stops the cluster
func (c *Cluster) Stop() error {
	if !c.config.Enabled {
		return nil
	}
	var stopErr error
	c.stopOnce.Do(func() {
		log.Info("Cluster stopping", "node_id", c.nodeID, "consensus_mode", c.consensusMode())

		switch c.consensusMode() {
		case "raft":
			stopErr = c.stopRaft()
		default:
			stopErr = c.stopCRDT()
		}
	})
	return stopErr
}

func (c *Cluster) consensusMode() string {
	if c.config.ConsensusMode == "" {
		return "crdt"
	}
	return c.config.ConsensusMode
}

func (c *Cluster) advertiseAddress() string {
	if c.config.AdvertiseAddress != "" {
		return c.config.AdvertiseAddress
	}
	return c.config.BindAddress
}

func (c *Cluster) advertisePort() int {
	if c.config.AdvertisePort > 0 {
		return c.config.AdvertisePort
	}
	return c.config.BindPort
}

// generateNodeID generates a unique node ID
func generateNodeID() string {
	data := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}
