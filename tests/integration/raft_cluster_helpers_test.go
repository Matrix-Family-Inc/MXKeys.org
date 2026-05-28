//go:build integration
// +build integration

/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// Shared harness for the raft_cluster_*_test.go integration suite.
// Owns the node struct, cluster builder, port picker, start/stop
// lifecycle, and the waitUntil / leadership helpers. Split out of
// the scenario files so every scenario file stays under the
// per-file line budget without each one reimplementing the harness.

package integration

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mxkeys/internal/zero/raft"
)

// clusterSecret is the HMAC secret every node shares in the integration
// harness. Any non-empty value works; keep it constant so the WAL HMAC
// signing and verification paths run end-to-end.
const clusterSecret = "integration-cluster-secret-32-bytes-padding!"

// node bundles everything the test needs to drive a single Raft peer.
//
// applied is written from the Raft apply goroutine and read from test
// goroutines. appliedMu guards both fields.
type node struct {
	id   string
	addr string
	dir  string
	raft *raft.Node

	appliedMu sync.Mutex
	applied   []raft.LogEntry
}

// appliedSnapshot returns a copy of the applied-entry slice under the
// lock. Callers must not rely on the returned slice's capacity for
// further writes.
func (n *node) appliedSnapshot() []raft.LogEntry {
	n.appliedMu.Lock()
	defer n.appliedMu.Unlock()
	return append([]raft.LogEntry(nil), n.applied...)
}

func (n *node) appliedLen() int {
	n.appliedMu.Lock()
	defer n.appliedMu.Unlock()
	return len(n.applied)
}

// pickFreePorts returns n TCP ports by binding and immediately closing.
// The OS guarantees kernel-chosen ports do not collide between calls in
// the same second, which is enough for an integration harness.
func pickFreePorts(t *testing.T, n int) []string {
	t.Helper()
	addrs := make([]string, 0, n)
	listeners := make([]net.Listener, 0, n)
	for i := 0; i < n; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("listen: %v", err)
		}
		listeners = append(listeners, l)
		addrs = append(addrs, l.Addr().String())
	}
	for _, l := range listeners {
		_ = l.Close()
	}
	return addrs
}

// newCluster builds n peers with persistent state directories, wires
// them as mutual peers, and returns them without starting. Callers
// decide when to Start and in what order. Clean up via t.Cleanup.
func newCluster(t *testing.T, n int) []*node {
	t.Helper()
	addrs := pickFreePorts(t, n)
	root := t.TempDir()

	nodes := make([]*node, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("n%d", i+1)
		peers := make([]string, 0, n-1)
		for j, a := range addrs {
			if j == i {
				continue
			}
			peers = append(peers, a)
		}

		host, port, err := net.SplitHostPort(addrs[i])
		if err != nil {
			t.Fatalf("split %q: %v", addrs[i], err)
		}
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err != nil {
			t.Fatalf("port parse %q: %v", port, err)
		}

		dir := filepath.Join(root, id)

		n := &node{
			id:   id,
			addr: addrs[i],
			dir:  dir,
		}
		n.raft = raft.NewNode(raft.Config{
			NodeID:            id,
			BindAddress:       host,
			BindPort:          p,
			Peers:             peers,
			ElectionTimeout:   200 * time.Millisecond,
			HeartbeatInterval: 50 * time.Millisecond,
			CommitTimeout:     2 * time.Second,
			SharedSecret:      clusterSecret,
		})
		if err := n.raft.SetStateDir(dir, true); err != nil {
			t.Fatalf("SetStateDir %s: %v", id, err)
		}

		// applyCb captures applied log entries so the test can assert
		// replication convergence without peeking at raft internals.
		// Guarded by appliedMu because the Raft apply goroutine calls
		// this while the test goroutine reads.
		captured := n
		n.raft.SetOnApply(func(e raft.LogEntry) {
			captured.appliedMu.Lock()
			captured.applied = append(captured.applied, e)
			captured.appliedMu.Unlock()
		})
		nodes[i] = n
	}
	return nodes
}

func startAll(t *testing.T, ctx context.Context, nodes []*node) {
	t.Helper()
	for _, n := range nodes {
		if err := n.raft.Start(ctx); err != nil {
			t.Fatalf("start %s: %v", n.id, err)
		}
	}
	t.Cleanup(func() {
		for _, n := range nodes {
			_ = n.raft.Stop()
		}
	})
}

// waitUntil polls fn every 10 ms until it returns true or the deadline
// expires. The per-attempt poll interval is short enough that election
// latency shows up as a single failed retry, not a hard timeout.
func waitUntil(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for: %s", timeout, desc)
}

func countLeaders(nodes []*node) int {
	n := 0
	for _, x := range nodes {
		if x.raft.State() == raft.Leader {
			n++
		}
	}
	return n
}

func findLeader(nodes []*node) *node {
	for _, n := range nodes {
		if n.raft.State() == raft.Leader {
			return n
		}
	}
	return nil
}
