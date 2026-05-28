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

// Snapshot / install-snapshot / follower-forward scenarios for the
// integration raft cluster. Core scenarios (election, baseline
// replication, leader kill, concurrent Submit) live in
// raft_cluster_core_test.go. Shared harness in
// raft_cluster_helpers_test.go.

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"mxkeys/internal/zero/raft"
)

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
		n.raft.SetSnapshotInstaller(func(r io.Reader, _ int64, _, _ uint64) error {
			if _, err := io.Copy(io.Discard, r); err != nil {
				return err
			}
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
	reborn.SetSnapshotInstaller(func(r io.Reader, _ int64, _, _ uint64) error {
		if _, err := io.Copy(io.Discard, r); err != nil {
			return err
		}
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
		n.raft.SetSnapshotInstaller(func(r io.Reader, size int64, _, _ uint64) error {
			data, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			if int64(len(data)) != size {
				return fmt.Errorf("installer read %d bytes, size arg %d", len(data), size)
			}
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
