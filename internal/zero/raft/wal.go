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
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// walFileName is the canonical on-disk name of the Raft write-ahead log.
const walFileName = "raft.wal"

// walMagicV2 is the Phase-4 WAL format (CRC32C only, no HMAC).
// Retained solely so OpenWAL can emit a precise error message instead
// of a generic "bad magic" when a legacy file is encountered.
var walMagicV2 = [12]byte{'M', 'X', 'K', 'S', '_', 'W', 'A', 'L', '_', 'v', '2', 0}

// walMagicV3 is the current on-disk format.
//
// v3 differs from v2 in the per-record layout:
//
//	v2:  len(4) || crc32c(4) || payload
//	v3:  len(4) || crc32c(4) || hmac_sha256(32) || payload
//
// The HMAC is keyed with WALOptions.HMACKey and covers the header
// (len || crc) plus payload. This makes the WAL tamper-evident against
// an attacker with write access to the disk: corrupting a record to
// inject a valid-looking command is infeasible without the key, whereas
// CRC32C alone only protects against benign bit rot.
//
// CRC32C is kept alongside the HMAC as a fast pre-check: when the CRC
// matches but the HMAC does not, we can distinguish "bit rot" from
// "tampered" in diagnostics.
//
// Upgrade from v2 is not automatic: operators run `mxkeys walctl
// upgrade` (documented in docs/runbook/raft-wal-upgrade.md) or delete
// the WAL and let the snapshot replay re-establish state.
var walMagicV3 = [12]byte{'M', 'X', 'K', 'S', '_', 'W', 'A', 'L', '_', 'v', '3', 0}

// walMagic is the format this binary writes. New OpenWAL calls produce
// v3; v2 is read-only via dedicated tooling.
var walMagic = walMagicV3

// walMagicSize is the length of walMagic.
const walMagicSize = 12

// walHeaderSize is the fixed prefix of each v3 WAL record: 4-byte LE
// payload length + 4-byte LE CRC32C over the payload + 32-byte HMAC.
const walHeaderSize = 4 + 4 + 32

// walMaxRecord caps the size of a single WAL payload to guard against
// truncated tails with garbage length fields. Matches Raft AppendEntries
// body limit so a follower's WAL cannot exceed an achievable RPC payload.
const walMaxRecord = 8 << 20 // 8 MiB

// walCRC is the CRC polynomial used for payload checksums. Castagnoli
// (CRC32C) is hardware-accelerated on x86 (SSE4.2) and ARM (crypto ext);
// better undetected-burst-error rate than IEEE and measurably faster on
// modern CPUs.
var walCRC = crc32.MakeTable(crc32.Castagnoli)

// walGroupFlushInterval is the time window for the group-commit batcher:
// pending Appends are buffered and written + fsync'd once per interval.
// 2 ms is a good tradeoff between tail latency (p99 <= ~3 ms under load)
// and amortization (dozens of Submits per fsync at production QPS).
const walGroupFlushInterval = 2 * time.Millisecond

// walBatchBufferSize is the bounded queue depth used by the group-commit
// batcher. Append blocks when the queue is full; this is the backpressure
// mechanism that prevents a degraded disk from letting Submit accumulate
// unbounded in-flight work.
const walBatchBufferSize = 1024

// ErrWALCorrupt indicates that the WAL contained a record whose declared
// length, CRC, or HMAC did not match. Callers should treat the log as
// truncated at the last well-formed record.
var ErrWALCorrupt = errors.New("raft wal: corrupt record")

// ErrWALTampered indicates that a record's CRC passed but its HMAC did
// not. This is a strong signal that disk contents were modified by
// something other than this binary (an attacker or a bug in an
// unrelated writer).
var ErrWALTampered = errors.New("raft wal: record HMAC mismatch (tampered)")

// ErrWALClosed is returned by Append after Close.
var ErrWALClosed = errors.New("raft wal: closed")

// ErrWALLegacyFormat is returned when OpenWAL encounters the v2
// on-disk format. Operators upgrade with the documented walctl tool.
var ErrWALLegacyFormat = errors.New("raft wal: legacy v2 format; run 'mxkeys walctl upgrade' or rebuild from snapshot")

// WAL is an append-only log of Raft LogEntry bytes.
//
// Every record is individually:
//   - CRC32C-checksummed to catch bit rot;
//   - HMAC-SHA256-authenticated with the cluster shared secret to catch
//     intentional tampering.
//
// Durability model: writes are grouped by a batcher goroutine that issues
// one fsync per flush window (walGroupFlushInterval) rather than per
// Append. This amortizes the syscall cost across bursts of Submits while
// keeping the "Append returns only after durability" contract: Append
// sends its entry + a done channel onto a bounded queue and blocks on
// the done signal, which the batcher closes after fsync.
type WAL struct {
	mu   sync.Mutex
	path string
	dir  string
	file *os.File

	// hmacKey is the secret used to authenticate each record.
	// Non-empty; OpenWAL refuses to construct a WAL without one.
	hmacKey []byte

	// Group commit plumbing. batchCh is nil when group-commit is disabled
	// (truncation/rewrite path uses the synchronous appendLocked helper).
	batchCh   chan walItem
	flushDone chan struct{}
	syncAll   bool
	closed    bool
}

// walItem is a unit of work for the group-commit batcher.
type walItem struct {
	entry LogEntry
	done  chan error
}

// WALOptions controls WAL construction.
type WALOptions struct {
	// Dir is the directory that holds raft.wal. Created with 0700 if missing.
	Dir string
	// SyncOnAppend issues an fsync after each batch flush. Strongly
	// recommended for production; tests may disable it for throughput.
	// Default true.
	SyncOnAppend bool
	// HMACKey is the cluster shared secret used to authenticate every
	// record. Required; OpenWAL fails if empty.
	HMACKey []byte
}

// OpenWAL opens (or creates) the write-ahead log in the given directory.
//
// Directory permissions are enforced to 0700 on every open to recover
// from out-of-band relaxation. Missing or empty files are initialized
// with the v3 magic prefix; existing files must match v3 exactly
// (ErrWALLegacyFormat for v2; ErrWALCorrupt for anything else).
//
// opts.HMACKey must be non-empty; the WAL format is always authenticated.
func OpenWAL(opts WALOptions) (*WAL, error) {
	if opts.Dir == "" {
		return nil, fmt.Errorf("raft wal: dir is required")
	}
	if len(opts.HMACKey) == 0 {
		return nil, fmt.Errorf("raft wal: HMAC key is required (derived from cluster.shared_secret)")
	}
	if err := os.MkdirAll(opts.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("raft wal: mkdir: %w", err)
	}
	if err := os.Chmod(opts.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("raft wal: chmod dir: %w", err)
	}

	path := filepath.Join(opts.Dir, walFileName)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("raft wal: open %s: %w", path, err)
	}

	if err := ensureMagic(f); err != nil {
		_ = f.Close()
		return nil, err
	}

	w := &WAL{
		path:      path,
		dir:       opts.Dir,
		file:      f,
		hmacKey:   append([]byte(nil), opts.HMACKey...),
		syncAll:   opts.SyncOnAppend,
		batchCh:   make(chan walItem, walBatchBufferSize),
		flushDone: make(chan struct{}),
	}
	// Pass the channel by value so the goroutine captures the handle even
	// if Close later sets w.batchCh = nil.
	go w.flushLoop(w.batchCh)
	return w, nil
}

// ensureMagic writes the v3 magic on empty files, or validates it on
// non-empty files. Returns ErrWALLegacyFormat for v2 files.
func ensureMagic(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("raft wal: stat: %w", err)
	}
	if fi.Size() == 0 {
		if _, err := f.Write(walMagic[:]); err != nil {
			return fmt.Errorf("raft wal: write magic: %w", err)
		}
		return nil
	}
	if fi.Size() < walMagicSize {
		return fmt.Errorf("raft wal: %w (file shorter than magic prefix)", ErrWALCorrupt)
	}

	var got [walMagicSize]byte
	if _, err := f.ReadAt(got[:], 0); err != nil {
		return fmt.Errorf("raft wal: read magic: %w", err)
	}
	if bytes.Equal(got[:], walMagic[:]) {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("raft wal: seek end: %w", err)
		}
		return nil
	}
	if bytes.Equal(got[:], walMagicV2[:]) {
		return ErrWALLegacyFormat
	}
	return fmt.Errorf("raft wal: %w (unknown format: %q)", ErrWALCorrupt, got)
}

// Sync fsyncs the WAL to stable storage.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	return w.file.Sync()
}

// Close drains pending batches, syncs, and releases the file. Safe to call
// multiple times; subsequent Append calls return ErrWALClosed.
func (w *WAL) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	ch := w.batchCh
	w.batchCh = nil
	w.mu.Unlock()

	if ch != nil {
		close(ch)
		<-w.flushDone
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	_ = w.file.Sync()
	err := w.file.Close()
	w.file = nil
	return err
}
