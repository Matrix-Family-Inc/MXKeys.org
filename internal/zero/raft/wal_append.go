/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package raft

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"time"
)

// Append writes a single LogEntry through the group-commit pipeline.
// Blocks until the batch containing this entry has been fsync'd (or an
// earlier error surfaces), providing a "durable before return" contract
// at a fraction of the syscall cost.
func (w *WAL) Append(entry LogEntry) error {
	w.mu.Lock()
	if w.closed || w.batchCh == nil {
		w.mu.Unlock()
		return ErrWALClosed
	}
	ch := w.batchCh
	w.mu.Unlock()

	done := make(chan error, 1)
	ch <- walItem{entry: entry, done: done}
	return <-done
}

// appendLocked writes a single entry synchronously bypassing the batcher.
// Used by the truncation rewrite path which already owns w.mu and has
// no concurrency by construction.
func (w *WAL) appendLocked(entry LogEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("raft wal: marshal: %w", err)
	}
	if len(payload) > walMaxRecord {
		return fmt.Errorf("raft wal: record too large: %d > %d", len(payload), walMaxRecord)
	}

	// Header layout (40 bytes): len(4) || crc32c(4) || hmac_sha256(32).
	// Payload length fits in uint32 by the walMaxRecord guard above.
	var hdr [walHeaderSize]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(payload))) // #nosec G115 -- bounded by walMaxRecord
	binary.LittleEndian.PutUint32(hdr[4:8], crc32.Checksum(payload, walCRC))

	// HMAC covers the len+crc prefix and the payload. Binding the
	// prefix into the HMAC means an attacker cannot truncate a record
	// without detection (shorter len would produce a different HMAC).
	mac := hmac.New(sha256.New, w.hmacKey)
	mac.Write(hdr[0:8])
	mac.Write(payload)
	copy(hdr[8:walHeaderSize], mac.Sum(nil))

	if _, err := w.file.Write(hdr[:]); err != nil {
		return fmt.Errorf("raft wal: write header: %w", err)
	}
	if _, err := w.file.Write(payload); err != nil {
		return fmt.Errorf("raft wal: write payload: %w", err)
	}
	return nil
}

// flushLoop is the group-commit batcher. See wal.go for the overall
// durability contract.
func (w *WAL) flushLoop(ch <-chan walItem) {
	defer close(w.flushDone)
	ticker := time.NewTicker(walGroupFlushInterval)
	defer ticker.Stop()

	var pending []walItem
	flush := func() {
		if len(pending) == 0 {
			return
		}
		w.mu.Lock()
		err := w.writeBatchLocked(pending)
		w.mu.Unlock()
		for _, it := range pending {
			it.done <- err
		}
		pending = pending[:0]
	}

	for {
		select {
		case item, ok := <-ch:
			if !ok {
				flush()
				return
			}
			pending = append(pending, item)
			// Opportunistically drain additional items arriving within
			// the same scheduling window.
			for drained := 0; drained < walBatchBufferSize; drained++ {
				select {
				case more, ok2 := <-ch:
					if !ok2 {
						flush()
						return
					}
					pending = append(pending, more)
				default:
					drained = walBatchBufferSize
				}
			}
		case <-ticker.C:
			flush()
		}
	}
}

// writeBatchLocked writes all pending records and optionally fsyncs
// once. Caller must hold w.mu.
func (w *WAL) writeBatchLocked(items []walItem) error {
	for _, it := range items {
		if err := w.appendLocked(it.entry); err != nil {
			return err
		}
	}
	if w.syncAll {
		if err := w.file.Sync(); err != nil {
			return fmt.Errorf("raft wal: fsync: %w", err)
		}
	}
	return nil
}
