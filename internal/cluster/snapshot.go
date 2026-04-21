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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mxkeys/internal/zero/log"
)

// keySnapshotVersion is the wire version of the snapshot payload
// produced by snapshotKeyState. A mismatch causes installKeySnapshot
// to refuse the payload rather than silently loading a shape the
// current build cannot interpret. Bump on any incompatible change.
const keySnapshotVersion uint32 = 1

// compactionCheckInterval is how often raftCompactionLoop evaluates
// whether the log is long enough to benefit from CompactLog. Short
// enough to keep the log bounded under steady write traffic, long
// enough to avoid work when the cluster is idle.
const compactionCheckInterval = 30 * time.Second

// compactionLogThreshold is the in-memory log length that triggers a
// compaction attempt. Picked conservatively so that a lagging
// follower is always served via AppendEntries (cheap) rather than
// InstallSnapshot (expensive) under normal operation.
const compactionLogThreshold = 1024

// ErrUnsupportedSnapshotVersion signals that installKeySnapshot was
// handed a payload from an incompatible build.
var ErrUnsupportedSnapshotVersion = errors.New("cluster: unsupported key snapshot version")

// keyStateSnapshot is the wire format for the cluster key cache that
// Raft persists as a snapshot. We keep it separate from KeyEntry so
// that future additions (e.g. tombstones, tenant tags) do not leak
// into the RPC surface.
type keyStateSnapshot struct {
	Version uint32                         `json:"v"`
	Keys    map[string]map[string]KeyEntry `json:"keys"`
}

// snapshotKeyState implements raft.SnapshotProvider. It produces a
// deterministic JSON serialization of the LWW key cache.
//
// Determinism matters: two replicas at the same commit index must
// produce byte-identical snapshots so that (LastIncludedIndex,
// payload) remain consistent across the cluster for replay and
// audit. encoding/json sorts map keys for us, so JSON is sufficient.
func (c *Cluster) snapshotKeyState() ([]byte, error) {
	c.state.mu.RLock()
	snap := keyStateSnapshot{
		Version: keySnapshotVersion,
		Keys:    make(map[string]map[string]KeyEntry, len(c.state.keys)),
	}
	for serverName, byID := range c.state.keys {
		clone := make(map[string]KeyEntry, len(byID))
		for keyID, entry := range byID {
			if entry == nil {
				continue
			}
			clone[keyID] = *entry
		}
		snap.Keys[serverName] = clone
	}
	c.state.mu.RUnlock()

	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("cluster: marshal key snapshot: %w", err)
	}
	return data, nil
}

// installKeySnapshot implements raft.SnapshotInstaller. It replaces
// the LWW cache with the snapshot contents.
//
// Must be idempotent for the same (lastIndex, lastTerm) pair: Raft
// may call this during startup (LoadFromDisk) and again when a
// leader pushes an InstallSnapshot for the same tuple.
func (c *Cluster) installKeySnapshot(data []byte, lastIncludedIndex, lastIncludedTerm uint64) error {
	if len(data) == 0 {
		c.state.mu.Lock()
		c.state.keys = make(map[string]map[string]*KeyEntry)
		c.state.mu.Unlock()
		log.Info("Cluster key snapshot installed (empty)",
			"last_included_index", lastIncludedIndex,
			"last_included_term", lastIncludedTerm,
		)
		return nil
	}

	var snap keyStateSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("cluster: decode key snapshot: %w", err)
	}
	if snap.Version != keySnapshotVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrUnsupportedSnapshotVersion, snap.Version, keySnapshotVersion)
	}

	rebuilt := make(map[string]map[string]*KeyEntry, len(snap.Keys))
	for serverName, byID := range snap.Keys {
		inner := make(map[string]*KeyEntry, len(byID))
		for keyID, entry := range byID {
			e := entry
			inner[keyID] = &e
		}
		rebuilt[serverName] = inner
	}

	c.state.mu.Lock()
	c.state.keys = rebuilt
	c.state.mu.Unlock()

	log.Info("Cluster key snapshot installed",
		"last_included_index", lastIncludedIndex,
		"last_included_term", lastIncludedTerm,
		"servers", len(rebuilt),
	)
	return nil
}

// raftCompactionLoop periodically evaluates the Raft in-memory log
// and triggers CompactLog when it grows beyond compactionLogThreshold.
// Compaction snapshots the current cache via snapshotKeyState,
// truncates the WAL prefix, and drops the in-memory log prefix
// covered by the snapshot.
//
// Without this loop the WAL grows unbounded and recovery time scales
// linearly with total history. With it, recovery is bounded by the
// snapshot size plus the most recent compactionLogThreshold entries.
func (c *Cluster) raftCompactionLoop(ctx context.Context) {
	defer c.wg.Done()

	t := time.NewTicker(compactionCheckInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-t.C:
			c.maybeCompactRaftLog()
		}
	}
}

// maybeCompactRaftLog is the per-tick body of raftCompactionLoop,
// extracted so it can be called directly from tests without waiting
// for a ticker.
func (c *Cluster) maybeCompactRaftLog() {
	if c.raftNode == nil {
		return
	}
	stats := c.raftNode.Stats()
	logLen, _ := stats["log_length"].(int)
	if logLen < compactionLogThreshold {
		return
	}
	if c.raftNode.LastApplied() == 0 {
		return
	}
	if err := c.raftNode.CompactLog(); err != nil {
		// CompactLog returns a descriptive error when there is
		// nothing to compact; that is expected on a fresh cluster
		// and should not alarm the operator.
		log.Debug("Raft compaction skipped", "error", err)
	}
}
