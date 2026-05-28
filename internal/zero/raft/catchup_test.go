/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

package raft

import (
	"testing"
	"time"
)

// TestNeedsSnapshotCatchUp locks in the contract that drives
// automatic InstallSnapshot catch-up: a leader must switch from
// AppendEntries to SendInstallSnapshot whenever a peer's nextIndex
// falls at or below the snapshot boundary. Before this guard the
// leader spun forever decrementing through a log prefix that only
// exists on disk.
func TestNeedsSnapshotCatchUp(t *testing.T) {
	newLeader := func() *Node {
		n := NewNode(Config{NodeID: "l", ElectionTimeout: 200 * time.Millisecond})
		n.state = Leader
		return n
	}

	t.Run("no snapshot means never catch up", func(t *testing.T) {
		n := newLeader()
		n.snapshotIndex = 0
		n.nextIndex["p"] = 1
		if n.needsSnapshotCatchUp("p") {
			t.Fatalf("without a snapshot, catch-up must not be chosen")
		}
	})

	t.Run("nextIdx above snapshot boundary ships AE", func(t *testing.T) {
		n := newLeader()
		n.snapshotIndex = 10
		n.nextIndex["p"] = 11
		if n.needsSnapshotCatchUp("p") {
			t.Fatalf("nextIdx=11 > snapshotIndex=10 must go via AppendEntries")
		}
	})

	t.Run("nextIdx at snapshot boundary requires InstallSnapshot", func(t *testing.T) {
		n := newLeader()
		n.snapshotIndex = 10
		n.nextIndex["p"] = 10
		if !n.needsSnapshotCatchUp("p") {
			t.Fatalf("nextIdx=10 == snapshotIndex=10 must trigger InstallSnapshot")
		}
	})

	t.Run("nextIdx below snapshot boundary requires InstallSnapshot", func(t *testing.T) {
		n := newLeader()
		n.snapshotIndex = 10
		n.nextIndex["p"] = 3
		if !n.needsSnapshotCatchUp("p") {
			t.Fatalf("nextIdx=3 below snapshot must trigger InstallSnapshot")
		}
	})

	t.Run("non-leader never drives catch-up", func(t *testing.T) {
		n := newLeader()
		n.state = Follower
		n.snapshotIndex = 10
		n.nextIndex["p"] = 1
		if n.needsSnapshotCatchUp("p") {
			t.Fatalf("a non-leader must not drive InstallSnapshot")
		}
	})
}

// TestBecomeLeaderPostCompactionNextIndex guards the bug where
// becomeLeader() initialised nextIndex from the in-memory slice
// length instead of the absolute log length. After compaction
// logOffset > 0 and len(n.log) no longer matches the tail. Using
// the bare slice length placed nextIndex below the real next slot,
// which in turn forced the new leader to decrement through entries
// that only existed inside the on-disk snapshot.
func TestBecomeLeaderPostCompactionNextIndex(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "leader",
		Peers:           []string{"peer-a:1", "peer-b:2"},
		ElectionTimeout: 200 * time.Millisecond,
	})
	n.state = Candidate
	// Simulate post-compaction state: snapshot covers 100 absolute
	// indices, with two uncompacted entries above the boundary.
	n.snapshotIndex = 100
	n.snapshotTerm = 4
	n.logOffset = 100
	n.log = []LogEntry{
		{Index: 101, Term: 4},
		{Index: 102, Term: 4},
	}

	n.becomeLeader()

	want := n.logLen() + 1
	if want != 103 {
		t.Fatalf("sanity: logLen+1 = %d, want 103", want)
	}
	for _, peer := range n.config.Peers {
		if n.nextIndex[peer] != want {
			t.Fatalf("nextIndex[%s] = %d, want %d", peer, n.nextIndex[peer], want)
		}
		if n.matchIndex[peer] != 0 {
			t.Fatalf("matchIndex[%s] = %d, want 0 (reset on becomeLeader)", peer, n.matchIndex[peer])
		}
	}
}
