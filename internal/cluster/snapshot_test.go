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
	"errors"
	"testing"
	"time"
)

// TestSnapshotKeyStateRoundTrip ensures the provider/installer pair
// preserves every field of every KeyEntry across a full serialization
// cycle. This is the foundation of durable Raft-replicated cache
// state; losing or mangling fields here would silently diverge
// replicas after a snapshot install.
func TestSnapshotKeyStateRoundTrip(t *testing.T) {
	src, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "src"})
	if err != nil {
		t.Fatalf("NewCluster(src): %v", err)
	}
	ts := time.Date(2026, 4, 21, 1, 2, 3, 0, time.UTC)
	entries := []*KeyEntry{
		{ServerName: "matrix.org", KeyID: "ed25519:auto", KeyData: "aaa", ValidUntilTS: 42, Timestamp: ts, NodeID: "src", Hash: "h1"},
		{ServerName: "matrix.org", KeyID: "ed25519:other", KeyData: "bbb", ValidUntilTS: 43, Timestamp: ts.Add(time.Second), NodeID: "src", Hash: "h2"},
		{ServerName: "example.org", KeyID: "ed25519:auto", KeyData: "ccc", ValidUntilTS: 44, Timestamp: ts.Add(2 * time.Second), NodeID: "src", Hash: "h3"},
	}
	// Seed entries AND bump the apply counter so the provider
	// returns the index the payload actually reflects.
	src.state.mu.Lock()
	for _, e := range entries {
		src.storeEntryLocked(e)
	}
	src.state.raftLastApplied = 42
	src.state.mu.Unlock()

	data, idx, err := src.snapshotKeyState()
	if err != nil {
		t.Fatalf("snapshotKeyState: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("snapshot payload must be non-empty")
	}
	if idx != 42 {
		t.Fatalf("snapshotKeyState index = %d, want 42", idx)
	}

	dst, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "dst"})
	if err != nil {
		t.Fatalf("NewCluster(dst): %v", err)
	}
	if err := dst.installKeySnapshot(data, 7, 3); err != nil {
		t.Fatalf("installKeySnapshot: %v", err)
	}

	for _, want := range entries {
		got := dst.GetCachedKey(want.ServerName, want.KeyID)
		if got == nil {
			t.Fatalf("entry missing after install: %s/%s", want.ServerName, want.KeyID)
		}
		if got.KeyData != want.KeyData || got.Hash != want.Hash ||
			got.ValidUntilTS != want.ValidUntilTS || got.NodeID != want.NodeID {
			t.Fatalf("entry mismatch for %s/%s: got %+v want %+v", want.ServerName, want.KeyID, got, want)
		}
		if !got.Timestamp.Equal(want.Timestamp) {
			t.Fatalf("timestamp drift for %s/%s: got %v want %v", want.ServerName, want.KeyID, got.Timestamp, want.Timestamp)
		}
	}
}

// TestInstallKeySnapshotRejectsUnknownVersion locks in the guard that
// refuses a payload from a future (or truncated) wire format, rather
// than loading garbage into the cache.
func TestInstallKeySnapshotRejectsUnknownVersion(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "dst"})
	if err != nil {
		t.Fatalf("NewCluster: %v", err)
	}
	// Hand-crafted payload with Version=99.
	payload := []byte(`{"v":99,"keys":{}}`)
	err = c.installKeySnapshot(payload, 1, 1)
	if err == nil {
		t.Fatalf("expected unsupported-version error, got nil")
	}
	if !errors.Is(err, ErrUnsupportedSnapshotVersion) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInstallKeySnapshotEmptyPayloadResetsCache validates the
// single-shot empty-snapshot branch used by InstallSnapshot's
// Offset=0/Done=true edge case.
func TestInstallKeySnapshotEmptyPayloadResetsCache(t *testing.T) {
	c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "dst"})
	if err != nil {
		t.Fatalf("NewCluster: %v", err)
	}
	c.storeEntry(&KeyEntry{
		ServerName: "srv", KeyID: "k", KeyData: "v",
		Timestamp: time.Now(), NodeID: "dst", Hash: "h",
	}, false)

	if err := c.installKeySnapshot(nil, 5, 2); err != nil {
		t.Fatalf("installKeySnapshot(empty): %v", err)
	}
	if got := c.GetCachedKey("srv", "k"); got != nil {
		t.Fatalf("empty snapshot must reset cache; got %+v", got)
	}
}

// TestSnapshotKeyStateDeterministic guards the invariant that two
// snapshots taken at the same logical state produce byte-identical
// payloads. Without this, replicas at the same commit index could
// write differing snapshot files, defeating the snapshot-audit
// story in ADR-0001.
func TestSnapshotKeyStateDeterministic(t *testing.T) {
	build := func() *Cluster {
		c, err := NewCluster(ClusterConfig{Enabled: true, NodeID: "n"})
		if err != nil {
			t.Fatalf("NewCluster: %v", err)
		}
		ts := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
		// Intentionally insert in different orders to exercise map
		// iteration non-determinism; encoding/json must sort keys.
		for _, e := range []*KeyEntry{
			{ServerName: "b.org", KeyID: "k2", KeyData: "x", Timestamp: ts, Hash: "hx"},
			{ServerName: "a.org", KeyID: "k1", KeyData: "y", Timestamp: ts, Hash: "hy"},
			{ServerName: "a.org", KeyID: "k3", KeyData: "z", Timestamp: ts, Hash: "hz"},
		} {
			c.storeEntry(e, false)
		}
		return c
	}

	aBytes, aIdx, err := build().snapshotKeyState()
	if err != nil {
		t.Fatalf("snapshotKeyState a: %v", err)
	}
	bBytes, bIdx, err := build().snapshotKeyState()
	if err != nil {
		t.Fatalf("snapshotKeyState b: %v", err)
	}
	if string(aBytes) != string(bBytes) {
		t.Fatalf("snapshotKeyState payload must be deterministic:\n a=%s\n b=%s", aBytes, bBytes)
	}
	if aIdx != bIdx {
		t.Fatalf("snapshotKeyState index must be deterministic: a=%d b=%d", aIdx, bIdx)
	}
}
