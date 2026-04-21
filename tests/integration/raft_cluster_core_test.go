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

// Core cluster scenarios: leader election, baseline replication,
// leader kill + re-election, and concurrent Submit convergence.
// Snapshot / install-snapshot / follower-forward scenarios live in
// raft_cluster_snapshot_test.go; shared harness (node, newCluster,
// waitUntil, ...) in raft_cluster_helpers_test.go.

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"mxkeys/internal/zero/raft"
)

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
