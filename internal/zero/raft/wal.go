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

// walMagic is a 12-byte file prefix that identifies this file as a MXKeys
// Raft WAL of a known version. Any format change beyond this version must
// bump the last byte and extend Open/Replay to cover both.
//
// Layout: "MXKS_WAL_v2\x00" (12 bytes). v1 was the initial CRC32-IEEE
// variant never shipped outside the Phase 4 feature branch; v2 uses
// CRC32-Castagnoli with group-commit support.
var walMagic = [12]byte{'M', 'X', 'K', 'S', '_', 'W', 'A', 'L', '_', 'v', '2', 0}

// walMagicSize is the length of walMagic.
const walMagicSize = 12

// walHeaderSize is the fixed prefix of each WAL record: 4-byte LE payload
// length + 4-byte LE CRC32C over the payload.
const walHeaderSize = 8

// walMaxRecord caps the size of a single WAL payload to guard against
// truncated tails with garbage length fields. Matches Raft AppendEntries
// body limit so a follower's WAL cannot exceed an achievable RPC payload.
const walMaxRecord = 8 << 20 // 8 MiB

// walCRC is the CRC polynomial used for payload checksums. Castagnoli
// (CRC32C) is hardware-accelerated on x86 (SSE4.2) and ARM (crypto ext);
// better undetected-burst-error rate than IEEE and measurably faster on
// modern CPUs. The WAL magic byte v2 signals the switch for future readers.
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
// length or CRC did not match. Callers should treat the log as truncated at
// the last well-formed record.
var ErrWALCorrupt = errors.New("raft wal: corrupt record")

// ErrWALClosed is returned by Append after Close.
var ErrWALClosed = errors.New("raft wal: closed")

// WAL is an append-only log of Raft LogEntry bytes with per-record CRC32C.
//
// Durability model: writes are grouped by a batcher goroutine that issues
// one fsync per flush window (walGroupFlushInterval) rather than per
// Append. This amortizes the syscall cost across bursts of Submits while
// keeping the "Append returns only after durability" contract: Append sends
// its entry + a done channel onto a bounded queue and blocks on the done
// signal, which the batcher closes after fsync.
//
// The bounded queue is the backpressure mechanism: if the disk degrades,
// pending writes pile up in the channel (up to walBatchBufferSize), and new
// Appends block rather than accumulating unbounded in-memory state.
type WAL struct {
	mu   sync.Mutex
	path string
	dir  string
	file *os.File

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
}

// OpenWAL opens (or creates) the write-ahead log in the given directory.
// Directory permissions are enforced to 0700 on every open to recover from
// out-of-band relaxation.
//
// On a new or empty file, the WAL magic prefix (walMagic) is written. On a
// non-empty file, the first bytes are validated against the expected magic
// and format version; a mismatch returns an error rather than risking
// silent misinterpretation of old/foreign data.
func OpenWAL(opts WALOptions) (*WAL, error) {
	if opts.Dir == "" {
		return nil, fmt.Errorf("raft wal: dir is required")
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
		syncAll:   opts.SyncOnAppend,
		batchCh:   make(chan walItem, walBatchBufferSize),
		flushDone: make(chan struct{}),
	}
	// Pass the channel by value so the goroutine captures the handle even
	// if Close later sets w.batchCh = nil. This is the only shared channel
	// reference; no data race on w.batchCh between the batcher and Close.
	go w.flushLoop(w.batchCh)
	return w, nil
}

// ensureMagic writes the WAL magic at offset 0 on empty files, or validates
// it on non-empty files. The file is repositioned to end-of-file on return
// so subsequent O_APPEND writes land after data.
func ensureMagic(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("raft wal: stat: %w", err)
	}
	if fi.Size() == 0 {
		// Fresh file: write magic. No fsync here; the first record's fsync
		// covers it. A crash between magic-write and first record leaves an
		// empty-but-magiced file, which is equivalent to a brand-new WAL.
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
	if !bytes.Equal(got[:], walMagic[:]) {
		return fmt.Errorf("raft wal: %w (unknown format: %q)", ErrWALCorrupt, got)
	}

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("raft wal: seek end: %w", err)
	}
	return nil
}

// Sync fsyncs the WAL to stable storage. Redundant when SyncOnAppend is
// enabled, but useful before shutdown or during tests to guarantee
// durability of any in-flight batch.
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
