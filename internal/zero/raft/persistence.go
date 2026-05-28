/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package raft

import (
	"errors"
	"fmt"

	"mxkeys/internal/zero/log"
)

// SetStateDir attaches a persistent state directory to the node. The same
// directory holds the WAL and the snapshot file. Must be called before
// Start; subsequent calls are no-ops to keep the invariant that persistence
// is bound once per node lifetime.
//
// When stateDir is empty the node runs in the legacy in-memory mode and no
// durability guarantees apply.
func (n *Node) SetStateDir(stateDir string, syncOnAppend bool) error {
	if stateDir == "" {
		return fmt.Errorf("raft: state dir is required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.wal != nil {
		return nil
	}

	if n.config.SharedSecret == "" {
		return fmt.Errorf("raft: SharedSecret is required to open the WAL (used to derive the WAL HMAC key)")
	}
	w, err := OpenWAL(WALOptions{
		Dir:          stateDir,
		SyncOnAppend: syncOnAppend,
		HMACKey:      []byte(n.config.SharedSecret),
	})
	if err != nil {
		return err
	}
	n.wal = w
	n.stateDir = stateDir
	return nil
}

// SetSnapshotProvider registers the state-machine snapshot callback used
// when CompactLog is invoked. See SnapshotProvider.
func (n *Node) SetSnapshotProvider(fn SnapshotProvider) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.snapshotProvider = fn
}

// SetSnapshotInstaller registers the state-machine snapshot-install callback
// used on startup and when a leader sends InstallSnapshot. See
// SnapshotInstaller.
func (n *Node) SetSnapshotInstaller(fn SnapshotInstaller) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.snapshotInstaller = fn
}

// LoadFromDisk restores node state from the persisted snapshot + WAL.
// Safe to call before Start; a no-op when SetStateDir was not called.
//
// Atomicity contract: LoadFromDisk inspects EVERY input before it
// touches the state machine or the Node. The snapshot file is
// opened and CRC-verified, the WAL is fully read into memory, and
// the installer contract is checked. Only after all inputs pass
// validation does the method run the application installer and
// acquire n.mu to apply snapshot/WAL state to Node fields. Any
// earlier failure leaves both the state machine and the Node
// unchanged, so a broken WAL or missing installer cannot half-load
// a node that a caller then mistakenly uses.
//
// Recovery:
//  1. If a snapshot exists, install it via snapshotInstaller and
//     set snapshotIndex/snapshotTerm/commitIndex/lastApplied.
//     Missing installer with a snapshot present is a configuration
//     error: ErrSnapshotInstallerRequired.
//  2. Replay WAL records with Index > snapshotIndex. Corrupt tails
//     truncate the replay (ErrWALCorrupt).
//  3. currentTerm advances to max(replayed-term, snapshotTerm).
func (n *Node) LoadFromDisk() error {
	if n.stateDir == "" {
		return nil
	}

	// A previous transfer that crashed mid-stream may have left a
	// spill file behind. Remove it before the node accepts new
	// InstallSnapshot chunks so the next transfer starts from a
	// known-empty state instead of inheriting partial bytes.
	cleanupStalePendingSnapshot(n.stateDir)

	// Phase 1: snapshot header + CRC pre-verify. The file stays
	// open (still owned by us) so the installer in Phase 3 can
	// stream its data portion without re-opening.
	f, meta, err := LoadSnapshotReader(n.stateDir)
	haveSnapshot := false
	switch {
	case errors.Is(err, ErrNoSnapshot):
		// no-op
	case err != nil:
		return fmt.Errorf("raft: load snapshot: %w", err)
	default:
		haveSnapshot = true
	}
	if haveSnapshot {
		defer func() {
			if f != nil {
				_ = f.Close()
			}
		}()
	}

	// Phase 2: installer-contract check. A persisted snapshot is
	// meaningless without a state machine to apply it to.
	// Rejecting here keeps the Node out of the "snapshotIndex
	// moved but nothing was applied" half-state described in
	// issue 2 of the audit pass.
	if haveSnapshot && n.snapshotInstaller == nil {
		return ErrSnapshotInstallerRequired
	}

	// Phase 3: WAL. Read the full record stream into memory before
	// touching the state machine so a fatal read error can abort
	// without having half-mutated anything. ErrWALCorrupt is
	// recoverable: keep the well-formed prefix we got.
	var walEntries []LogEntry
	var walCorrupt bool
	if n.wal != nil {
		entries, werr := n.wal.ReadAll()
		if werr != nil && !errors.Is(werr, ErrWALCorrupt) {
			return fmt.Errorf("raft: read wal: %w", werr)
		}
		walEntries = entries
		walCorrupt = errors.Is(werr, ErrWALCorrupt)
	}

	// Phase 4: run installer. After this point state-machine
	// mutations are committed; failures can no longer be rolled
	// back, so we must not touch state after Phase 3 gate-checks.
	if haveSnapshot {
		cr := &countingReader{r: f}
		if ierr := n.snapshotInstaller(cr, meta.Size, meta.LastIncludedIndex, meta.LastIncludedTerm); ierr != nil {
			return fmt.Errorf("raft: install snapshot: %w", ierr)
		}
		if cr.count != meta.Size {
			return fmt.Errorf("raft: startup installer read %d of %d bytes: short read violates installer contract", cr.count, meta.Size)
		}
	}

	// Phase 5: apply to Node under a single critical section.
	if walCorrupt {
		log.Warn("Raft WAL tail corrupt; truncating to last well-formed record",
			"kept_entries", len(walEntries),
		)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if haveSnapshot {
		n.snapshotIndex = meta.LastIncludedIndex
		n.snapshotTerm = meta.LastIncludedTerm
		n.logOffset = meta.LastIncludedIndex
		n.commitIndex = meta.LastIncludedIndex
		n.lastApplied = meta.LastIncludedIndex
		if meta.LastIncludedTerm > n.currentTerm {
			n.currentTerm = meta.LastIncludedTerm
		}
		log.Info("Raft snapshot loaded",
			"last_included_index", meta.LastIncludedIndex,
			"last_included_term", meta.LastIncludedTerm,
			"size", meta.Size,
		)
	}
	for _, e := range walEntries {
		if e.Index <= n.snapshotIndex {
			// Already captured by the snapshot; ignore.
			continue
		}
		n.log = append(n.log, e)
		if e.Term > n.currentTerm {
			n.currentTerm = e.Term
		}
	}
	return nil
}

// persistEntry durably stores a single appended log entry.
// Caller may hold n.mu; WAL has its own mutex.
func (n *Node) persistEntry(entry LogEntry) error {
	if n.wal == nil {
		return nil
	}
	return n.wal.Append(entry)
}

// truncateLogAfter drops log entries with Index > lastKeepIndex from both
// the in-memory slice and the WAL. Used when a follower's tail conflicts
// with the leader's view of history.
//
// Caller must hold n.mu.
func (n *Node) truncateLogAfter(lastKeepIndex uint64) error {
	n.truncateSliceAfter(lastKeepIndex)
	if n.wal == nil {
		return nil
	}
	return n.wal.TruncateAfter(lastKeepIndex)
}

// CompactLog lives in compact.go to keep this file focused on
// startup/restore and the small WAL-adjacent helpers.
