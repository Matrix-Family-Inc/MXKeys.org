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

// TestRaftAutomaticInstallSnapshotCatchUp drives the production
// path that was previously broken: a leader whose lagging peer has
// nextIndex at or below the snapshot boundary must automatically
// switch from AppendEntries to SendInstallSnapshot. Without the fix
// sendHeartbeats would loop forever decrementing nextIndex through
// entries that only exist on disk.
//
// Scenario:
//
//  1. 3-node cluster elects a leader.
//  2. Every node hooks a snapshot provider/installer reflecting a
//     tracked state blob.
//  3. We stop one follower, submit further entries on the leader
//     (2/3 quorum still commits), drain and compact the leader.
//  4. The leader's snapshotIndex now sits strictly above the stopped
//     follower's matchIndex; its nextIndex[victim] falls inside the
//     snapshot prefix.
//  5. We bring the victim back up with the same addr and a fresh
//     state directory. The leader's heartbeat tick detects the
//     catch-up condition and streams InstallSnapshot. The installer
//     counter on the victim must increment.
func TestRaftAutomaticInstallSnapshotCatchUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	nodes := newCluster(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	payload := []byte("catchup-state-blob")
	installed := make([]*atomic.Int32, 3)
	for i, n := range nodes {
		i, n := i, n
		installed[i] = &atomic.Int32{}
		// Provider atomically reads (payload, idx) under the harness's
		// own apply mutex: idx is the index of the latest entry that
		// the apply callback captured into n.applied, which exactly
		// matches the moment the static payload reflects.
		n.raft.SetSnapshotProvider(func() ([]byte, uint64, error) {
			n.appliedMu.Lock()
			var idx uint64
			if len(n.applied) > 0 {
				idx = n.applied[len(n.applied)-1].Index
			}
			n.appliedMu.Unlock()
			return append([]byte(nil), payload...), idx, nil
		})
		n.raft.SetSnapshotInstaller(func(data []byte, _, _ uint64) error {
			installed[i].Add(1)
			return nil
		})
	}

	startAll(t, ctx, nodes)
	waitUntil(t, 3*time.Second, "leader elected", func() bool {
		return findLeader(nodes) != nil
	})
	leader := findLeader(nodes)

	var victimIdx int
	var victim *node
	for i, n := range nodes {
		if n != leader {
			victim = n
			victimIdx = i
			break
		}
	}

	// Submit a baseline so every node has a non-trivial applied log.
	for i := 0; i < 3; i++ {
		if err := leader.raft.Submit(ctx, json.RawMessage(fmt.Sprintf(`"pre-%d"`, i))); err != nil {
			t.Fatalf("pre-submit: %v", err)
		}
	}
	waitUntil(t, 3*time.Second, "victim caught up baseline", func() bool {
		return victim.appliedLen() >= 3
	})

	// Stop the victim; remaining 2-of-3 quorum still commits.
	_ = victim.raft.Stop()

	for i := 0; i < 5; i++ {
		if err := leader.raft.Submit(ctx, json.RawMessage(fmt.Sprintf(`"post-%d"`, i))); err != nil {
			t.Fatalf("post-submit: %v", err)
		}
	}
	waitUntil(t, 3*time.Second, "leader applied post-entries", func() bool {
		return leader.raft.LastApplied() >= 8
	})

	if err := leader.raft.CompactLog(); err != nil {
		t.Fatalf("CompactLog: %v", err)
	}

	// Rebuild the victim with a fresh state dir on the same addr so
	// the leader's stale nextIndex[victim] > new victim's zero log.
	freshDir := t.TempDir()
	host, port, err := net.SplitHostPort(victim.addr)
	if err != nil {
		t.Fatalf("split victim addr: %v", err)
	}
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		t.Fatalf("parse victim port: %v", err)
	}
	peers := make([]string, 0, len(nodes)-1)
	for i, n := range nodes {
		if i == victimIdx {
			continue
		}
		peers = append(peers, n.addr)
	}
	reborn := raft.NewNode(raft.Config{
		NodeID:            victim.id,
		BindAddress:       host,
		BindPort:          portNum,
		Peers:             peers,
		ElectionTimeout:   200 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
		CommitTimeout:     2 * time.Second,
		SharedSecret:      clusterSecret,
	})
	if err := reborn.SetStateDir(freshDir, true); err != nil {
		t.Fatalf("reborn SetStateDir: %v", err)
	}
	// Reborn node has no apply history yet; its provider returns 0
	// as the applied index. It will never call CompactLog in this
	// test before observing applies, so the value is academic.
	reborn.SetSnapshotProvider(func() ([]byte, uint64, error) {
		return append([]byte(nil), payload...), 0, nil
	})
	reborn.SetSnapshotInstaller(func(data []byte, _, _ uint64) error {
		installed[victimIdx].Add(1)
		return nil
	})
	if err := reborn.Start(ctx); err != nil {
		t.Fatalf("reborn Start: %v", err)
	}
	t.Cleanup(func() { _ = reborn.Stop() })

	// The leader's heartbeat tick (50 ms) should observe the stale
	// nextIndex, detect nextIdx <= snapshotIndex, and drive
	// SendInstallSnapshot automatically. We only care that the
	// installer counter advanced on the reborn node.
	waitUntil(t, 5*time.Second, "automatic InstallSnapshot caught up victim", func() bool {
		return installed[victimIdx].Load() >= 1
	})
}

// TestRaftProposeFromFollowerForwardsToLeader exercises the
// follower-forward path. A Propose call on a node that is not the
// leader must transparently forward to the current leader via
// MsgForwardProposal, and the committed entry must apply on every
// replica. Previously the only write API was Submit, which returns
// ErrNotLeader on followers; client writes hitting a follower were
// silently dropped at the cluster level.
func TestRaftProposeFromFollowerForwardsToLeader(t *testing.T) {
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

	var follower *node
	for _, n := range nodes {
		if n != leader {
			follower = n
			break
		}
	}

	// Wait until the follower has seen at least one AppendEntries
	// from the leader so that its leaderAddr is populated. Without
	// this, Propose would race the first heartbeat and return
	// ErrNoLeader on the initial tick.
	waitUntil(t, 2*time.Second, "follower learned leader id", func() bool {
		return follower.raft.LeaderID() == leader.raft.LeaderID() && follower.raft.LeaderID() != ""
	})

	cmd := json.RawMessage(`{"forwarded":true}`)
	if err := follower.raft.Propose(ctx, cmd); err != nil {
		t.Fatalf("Propose on follower: %v", err)
	}

	waitUntil(t, 3*time.Second, "every node applied forwarded entry", func() bool {
		for _, n := range nodes {
			found := false
			for _, e := range n.appliedSnapshot() {
				if string(e.Command) == string(cmd) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	})
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
		n.raft.SetSnapshotProvider(func() ([]byte, uint64, error) {
			n.appliedMu.Lock()
			var idx uint64
			if len(n.applied) > 0 {
				idx = n.applied[len(n.applied)-1].Index
			}
			n.appliedMu.Unlock()
			return append([]byte(nil), payload...), idx, nil
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
