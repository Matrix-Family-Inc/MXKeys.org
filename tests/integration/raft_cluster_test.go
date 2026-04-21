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

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
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

// TestRaftClusterElectsSingleLeader brings up a 3-node cluster and
// checks that exactly one node becomes leader within the
// election-timeout window. Pre-vote must not prevent election when
// no leader has ever existed.
func TestRaftClusterElectsSingleLeader(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	nodes := newCluster(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startAll(t, ctx, nodes)

	waitUntil(t, 3*time.Second, "exactly one leader", func() bool {
		return countLeaders(nodes) == 1
	})
}

// TestRaftSubmitReplicatesToAllPeers submits a single command on the
// leader and verifies every follower applies the identical entry.
func TestRaftSubmitReplicatesToAllPeers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	nodes := newCluster(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startAll(t, ctx, nodes)

	waitUntil(t, 3*time.Second, "leader elected", func() bool {
		return findLeader(nodes) != nil
	})
	leader := findLeader(nodes)

	cmd := json.RawMessage(`{"type":"test","n":1}`)
	if err := leader.raft.Submit(ctx, cmd); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	waitUntil(t, 3*time.Second, "all nodes apply the entry", func() bool {
		for _, n := range nodes {
			a := n.appliedSnapshot()
			if len(a) == 0 || string(a[0].Command) != string(cmd) {
				return false
			}
		}
		return true
	})
}

// TestRaftSurvivesLeaderKill kills the elected leader and verifies the
// remaining two peers elect a new leader. Exercises the pre-vote and
// real-vote paths plus term bumping under leader failure.
func TestRaftSurvivesLeaderKill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	nodes := newCluster(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startAll(t, ctx, nodes)

	waitUntil(t, 3*time.Second, "first leader", func() bool {
		return findLeader(nodes) != nil
	})
	oldLeader := findLeader(nodes)
	oldTerm := oldLeader.raft.Term()

	// Simulate leader kill.
	_ = oldLeader.raft.Stop()

	// The remaining two peers must re-elect within a few election
	// timeouts. ElectionTimeout = 200ms; allow 3 s for pre-vote +
	// real vote round plus network jitter.
	waitUntil(t, 5*time.Second, "new leader among survivors", func() bool {
		for _, n := range nodes {
			if n == oldLeader {
				continue
			}
			if n.raft.State() == raft.Leader && n.raft.Term() > oldTerm {
				return true
			}
		}
		return false
	})
}

// TestRaftConcurrentSubmits drives the leader with concurrent Submit
// calls from many goroutines to exercise the group-commit batcher
// and the persist-outside-lock invariant. All submissions must land
// in log order (monotonic indices) and every follower must converge
// to the same applied state.
func TestRaftConcurrentSubmits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	nodes := newCluster(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startAll(t, ctx, nodes)

	waitUntil(t, 3*time.Second, "leader elected", func() bool {
		return findLeader(nodes) != nil
	})
	leader := findLeader(nodes)

	const workers = 16
	const perWorker = 4
	total := workers * perWorker

	var ok atomic.Int32
	errCh := make(chan error, total)
	for w := 0; w < workers; w++ {
		go func(w int) {
			for i := 0; i < perWorker; i++ {
				cmd := json.RawMessage(fmt.Sprintf(`{"w":%d,"i":%d}`, w, i))
				if err := leader.raft.Submit(ctx, cmd); err != nil {
					errCh <- err
					return
				}
				ok.Add(1)
			}
			errCh <- nil
		}(w)
	}
	for w := 0; w < workers; w++ {
		if err := <-errCh; err != nil {
			t.Fatalf("submit: %v", err)
		}
	}
	if int(ok.Load()) != total {
		t.Fatalf("expected %d submits acked, got %d", total, ok.Load())
	}

	waitUntil(t, 5*time.Second, "all nodes apply all entries", func() bool {
		for _, n := range nodes {
			if n.appliedLen() < total {
				return false
			}
		}
		return true
	})

	// Every peer must see the same applied entries in the same order.
	// Raft guarantees ordering within a term; in a single-term run the
	// check is simple byte equality.
	ref := nodes[0].appliedSnapshot()[:total]
	for _, n := range nodes[1:] {
		got := n.appliedSnapshot()[:total]
		for i := 0; i < total; i++ {
			if got[i].Index != ref[i].Index || string(got[i].Command) != string(ref[i].Command) {
				t.Fatalf("divergence at idx %d between n1 and %s", i, n.id)
			}
		}
	}
}

// TestRaftInstallSnapshotStreamsLargePayload drives the end-to-end
// chunked InstallSnapshot path with a payload that spans multiple
// 512 KiB chunks. Requires a snapshot provider and installer hooked
// on both sides.
func TestRaftInstallSnapshotStreamsLargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	nodes := newCluster(t, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1.5 MiB payload: three 512 KiB chunks.
	payload := make([]byte, 1536*1024)
	for i := range payload {
		payload[i] = byte(i)
	}

	var installed atomic.Int32
	for _, n := range nodes {
		n := n
		n.raft.SetSnapshotProvider(func() ([]byte, error) {
			return append([]byte(nil), payload...), nil
		})
		n.raft.SetSnapshotInstaller(func(data []byte, idx, term uint64) error {
			if len(data) != len(payload) {
				return fmt.Errorf("size mismatch: got %d want %d", len(data), len(payload))
			}
			for i := range data {
				if data[i] != payload[i] {
					return fmt.Errorf("byte %d mismatch", i)
				}
			}
			installed.Add(1)
			return nil
		})
	}
	startAll(t, ctx, nodes)

	waitUntil(t, 3*time.Second, "leader elected", func() bool {
		return findLeader(nodes) != nil
	})
	leader := findLeader(nodes)

	// Submit an entry so the leader has a non-trivial log, then
	// compact. CompactLog calls the snapshot provider and truncates.
	if err := leader.raft.Submit(ctx, json.RawMessage(`"first"`)); err != nil {
		t.Fatalf("submit: %v", err)
	}
	waitUntil(t, 2*time.Second, "entry committed", func() bool {
		return leader.raft.CommitIndex() >= 1
	})

	var follower *node
	for _, n := range nodes {
		if n != leader {
			follower = n
			break
		}
	}

	// Wait until lastApplied has caught up with the commit index on
	// the leader; CompactLog refuses when there is nothing to compact.
	waitUntil(t, 2*time.Second, "leader applied the first entry", func() bool {
		return leader.raft.LastApplied() >= leader.raft.CommitIndex()
	})
	if err := leader.raft.CompactLog(); err != nil {
		t.Fatalf("CompactLog: %v", err)
	}

	// Force the leader to stream to the follower. The follower's
	// installer verifies byte identity of the reassembled payload.
	if err := leader.raft.SendInstallSnapshot(ctx, follower.addr); err != nil {
		t.Fatalf("SendInstallSnapshot: %v", err)
	}
	waitUntil(t, 3*time.Second, "follower installed snapshot", func() bool {
		// installed.Add happens on both nodes (self-install path on leader
		// is not exercised here; only the follower's install counts).
		return installed.Load() >= 1
	})
}
