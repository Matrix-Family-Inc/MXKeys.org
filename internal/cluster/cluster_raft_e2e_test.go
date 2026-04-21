/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package cluster

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"
)

const e2eRaftSharedSecret = "e2e-raft-shared-secret-32-bytes-padding!"

// e2eAllocAddr returns a kernel-chosen "127.0.0.1:port" by binding
// and closing. Short-lived race window, acceptable for a test harness.
func e2eAllocAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

// e2eBuildNode wires a single cluster.Cluster with raft mode on a
// pre-allocated address/port and a per-node state directory. It does
// NOT start the cluster; the caller controls startup order.
func e2eBuildNode(t *testing.T, id, addr string, stateDir string, peers []string) *Cluster {
	t.Helper()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split addr %q: %v", addr, err)
	}
	var portNum int
	if _, err := fmt.Sscanf(port, "%d", &portNum); err != nil {
		t.Fatalf("parse port %q: %v", port, err)
	}
	cfg := ClusterConfig{
		Enabled:          true,
		NodeID:           id,
		BindAddress:      host,
		BindPort:          portNum,
		AdvertiseAddress: host,
		AdvertisePort:    portNum,
		Seeds:            peers,
		ConsensusMode:    "raft",
		SyncInterval:     1,
		SharedSecret:     e2eRaftSharedSecret,
		RaftStateDir:     stateDir,
		RaftSyncOnAppend: true,
	}
	c, err := NewCluster(cfg)
	if err != nil {
		t.Fatalf("NewCluster %s: %v", id, err)
	}
	return c
}

// waitUntilE2E polls fn every 10 ms until it returns true or the
// timeout elapses. Shared helper for this file's scenarios.
func waitUntilE2E(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
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

// TestRaftClusterEndToEndWriteCompactRestart exercises the full
// production path through the cluster.Cluster API rather than the
// raw raft.Node layer:
//
//  1. Three raft-mode Cluster instances start and elect a leader.
//  2. BroadcastKeyUpdate runs on a follower. The write reaches the
//     LWW cache on every replica via Propose → forward →
//     leader.Submit → commit → onApply.
//  3. CompactLog runs on the leader. The new atomicity contract
//     gives SaveSnapshot a (payload, raftLastApplied) pair
//     captured under a single c.state.mu lock; the saved file's
//     LastIncludedIndex is trustworthy.
//  4. One node is stopped and rebuilt from the same state dir.
//     LoadFromDisk replays the snapshot via installKeySnapshot,
//     which restores the LWW cache and advances raftLastApplied to
//     match the snapshot metadata.
//  5. GetCachedKey on the restarted node returns the entry that
//     was originally written through the follower.
//
// A passing run proves every seam between notary-hook writes and
// post-restart durability holds end to end.
func TestRaftClusterEndToEndWriteCompactRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping raft end-to-end test in short mode")
	}

	addrs := []string{e2eAllocAddr(t), e2eAllocAddr(t), e2eAllocAddr(t)}
	root := t.TempDir()
	dirs := []string{
		filepath.Join(root, "n1"),
		filepath.Join(root, "n2"),
		filepath.Join(root, "n3"),
	}

	peersFor := func(i int) []string {
		out := make([]string, 0, len(addrs)-1)
		for j, a := range addrs {
			if j == i {
				continue
			}
			out = append(out, a)
		}
		return out
	}

	nodes := []*Cluster{
		e2eBuildNode(t, "n1", addrs[0], dirs[0], peersFor(0)),
		e2eBuildNode(t, "n2", addrs[1], dirs[1], peersFor(1)),
		e2eBuildNode(t, "n3", addrs[2], dirs[2], peersFor(2)),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i, c := range nodes {
		if err := c.Start(ctx); err != nil {
			t.Fatalf("Start %s: %v", []string{"n1", "n2", "n3"}[i], err)
		}
	}
	cleanup := func() {
		for _, c := range nodes {
			if c == nil {
				continue
			}
			_ = c.Stop()
		}
	}
	defer cleanup()

	// Wait for a leader. Cluster exposes this via raft node stats.
	findLeader := func() *Cluster {
		for _, c := range nodes {
			if c.raftNode != nil && c.raftNode.IsLeader() {
				return c
			}
		}
		return nil
	}
	waitUntilE2E(t, 3*time.Second, "leader elected", func() bool {
		return findLeader() != nil
	})
	leader := findLeader()

	var follower *Cluster
	for _, c := range nodes {
		if c != leader {
			follower = c
			break
		}
	}

	// Ensure the follower has a leader address wired up so Propose
	// can forward rather than immediately returning ErrNoLeader.
	waitUntilE2E(t, 2*time.Second, "follower knows leader", func() bool {
		return follower.raftNode.LeaderID() == leader.raftNode.LeaderID() &&
			follower.raftNode.LeaderID() != ""
	})

	// Write via follower. The payload and the apply must converge
	// on every replica before we compact.
	follower.BroadcastKeyUpdate("matrix.example", "ed25519:auto", "cafebabe", 9999)
	waitUntilE2E(t, 5*time.Second, "every replica caches the forwarded entry", func() bool {
		for _, c := range nodes {
			if c.GetCachedKey("matrix.example", "ed25519:auto") == nil {
				return false
			}
		}
		return true
	})

	// Drive compaction on the leader. The snapshot provider must
	// return (payload, raftLastApplied) atomically; CompactLog
	// validates the reported index against snapshotIndex/commitIndex
	// and only then persists.
	leader.state.mu.RLock()
	appliedBeforeCompact := leader.state.raftLastApplied
	leader.state.mu.RUnlock()
	if appliedBeforeCompact == 0 {
		t.Fatalf("leader raftLastApplied must be > 0 after a successful Propose; got 0")
	}
	if err := leader.raftNode.CompactLog(); err != nil {
		t.Fatalf("CompactLog on leader: %v", err)
	}

	// Stop n1 (arbitrary choice); we will rebuild it from the
	// same state dir.
	restartIdx := 0
	restartDir := dirs[restartIdx]
	restartAddr := addrs[restartIdx]
	_ = nodes[restartIdx].Stop()
	nodes[restartIdx] = nil // so cleanup skips it

	// Rebuild n1 pointing at the same state directory.
	reborn := e2eBuildNode(t, "n1", restartAddr, restartDir, peersFor(restartIdx))
	if err := reborn.Start(ctx); err != nil {
		t.Fatalf("reborn Start: %v", err)
	}
	defer func() { _ = reborn.Stop() }()

	// The restart path: LoadFromDisk → installKeySnapshot restores
	// keys AND bumps raftLastApplied to the snapshot's
	// LastIncludedIndex. GetCachedKey must see the entry that was
	// written through the follower before the restart.
	waitUntilE2E(t, 5*time.Second, "reborn node restored key from snapshot", func() bool {
		return reborn.GetCachedKey("matrix.example", "ed25519:auto") != nil
	})

	// raftLastApplied after restore must be at or above the index
	// captured in the snapshot; if the installer had failed to
	// track the counter, snapshotKeyState would later return a
	// zero index and break future compactions.
	reborn.state.mu.RLock()
	appliedAfterRestore := reborn.state.raftLastApplied
	reborn.state.mu.RUnlock()
	if appliedAfterRestore < appliedBeforeCompact {
		t.Fatalf("raftLastApplied after restore = %d, must be >= pre-compact value %d", appliedAfterRestore, appliedBeforeCompact)
	}
}
