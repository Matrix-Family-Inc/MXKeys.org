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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// walFileName is the canonical on-disk name of the Raft write-ahead log.
const walFileName = "raft.wal"

// walHeaderSize is the fixed prefix of each WAL record: 4-byte LE payload
// length + 4-byte LE CRC32 over the payload.
const walHeaderSize = 8

// walMaxRecord caps the size of a single WAL payload to guard against
// truncated tails with garbage length fields. Matches Raft AppendEntries
// body limit so a follower's WAL cannot exceed an achievable RPC payload.
const walMaxRecord = 8 << 20 // 8 MiB

// walCRC is the CRC polynomial used for payload checksums. IEEE matches
// Go's default and is supported by every standard library.
var walCRC = crc32.IEEETable

// ErrWALCorrupt indicates that the WAL contained a record whose declared
// length or CRC did not match. Callers should treat the log as truncated at
// the last well-formed record.
var ErrWALCorrupt = errors.New("raft wal: corrupt record")

// WAL is an append-only log of Raft LogEntry bytes with per-record CRC32.
// All operations are safe for concurrent use.
type WAL struct {
	mu      sync.Mutex
	path    string
	dir     string
	file    *os.File
	syncAll bool
}

// WALOptions controls WAL construction.
type WALOptions struct {
	// Dir is the directory that holds raft.wal. Created with 0700 if missing.
	Dir string
	// SyncOnAppend issues an fsync after every Append. Strongly recommended
	// for production; tests may disable it for throughput. Default true.
	SyncOnAppend bool
}

// OpenWAL opens (or creates) the write-ahead log in the given directory.
// Directory permissions are enforced to 0700 on every open to recover from
// out-of-band relaxation.
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

	return &WAL{
		path:    path,
		dir:     opts.Dir,
		file:    f,
		syncAll: opts.SyncOnAppend,
	}, nil
}

// Append writes a single LogEntry as a WAL record. When SyncOnAppend is set,
// the file is fsync'd before Append returns so the entry is durable on power
// loss.
func (w *WAL) Append(entry LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.appendLocked(entry)
}

// appendLocked is the lock-free body of Append. Callers must hold w.mu.
func (w *WAL) appendLocked(entry LogEntry) error {
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("raft wal: marshal: %w", err)
	}
	if len(payload) > walMaxRecord {
		return fmt.Errorf("raft wal: record too large: %d > %d", len(payload), walMaxRecord)
	}

	var hdr [walHeaderSize]byte
	binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(payload)))
	binary.LittleEndian.PutUint32(hdr[4:8], crc32.Checksum(payload, walCRC))

	if _, err := w.file.Write(hdr[:]); err != nil {
		return fmt.Errorf("raft wal: write header: %w", err)
	}
	if _, err := w.file.Write(payload); err != nil {
		return fmt.Errorf("raft wal: write payload: %w", err)
	}

	if w.syncAll {
		if err := w.file.Sync(); err != nil {
			return fmt.Errorf("raft wal: fsync: %w", err)
		}
	}
	return nil
}

// Sync fsyncs the WAL to stable storage. Call after a batch of Append calls
// when SyncOnAppend is disabled.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

// Close syncs and releases the underlying file.
func (w *WAL) Close() error {
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

// ReadAll returns every well-formed entry in the WAL. If a corrupt record is
// encountered, reading stops and the successfully parsed prefix is returned
// along with ErrWALCorrupt so the caller can decide whether to truncate.
func (w *WAL) ReadAll() ([]LogEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.readAllLocked()
}

// readAllLocked parses the current WAL from offset 0 to EOF. The file offset
// is restored to end-of-file before return so subsequent Appends land after
// the existing data. Caller must hold w.mu.
func (w *WAL) readAllLocked() ([]LogEntry, error) {
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("raft wal: seek: %w", err)
	}
	defer func() {
		_, _ = w.file.Seek(0, io.SeekEnd)
	}()

	var (
		entries []LogEntry
		hdr     [walHeaderSize]byte
	)

	for {
		_, err := io.ReadFull(w.file, hdr[:])
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return entries, ErrWALCorrupt
		}
		if err != nil {
			return entries, fmt.Errorf("raft wal: read header: %w", err)
		}

		length := binary.LittleEndian.Uint32(hdr[0:4])
		declaredCRC := binary.LittleEndian.Uint32(hdr[4:8])
		if length == 0 || length > walMaxRecord {
			return entries, ErrWALCorrupt
		}

		payload := make([]byte, length)
		if _, err := io.ReadFull(w.file, payload); err != nil {
			return entries, ErrWALCorrupt
		}

		if crc32.Checksum(payload, walCRC) != declaredCRC {
			return entries, ErrWALCorrupt
		}

		var entry LogEntry
		if err := json.Unmarshal(payload, &entry); err != nil {
			return entries, ErrWALCorrupt
		}
		entries = append(entries, entry)
	}
}
