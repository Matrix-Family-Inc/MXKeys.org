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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

// ReadAll returns every well-formed entry in the WAL. If a corrupt record is
// encountered, reading stops and the successfully parsed prefix is returned
// along with ErrWALCorrupt so the caller can decide whether to truncate.
func (w *WAL) ReadAll() ([]LogEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.readAllLocked()
}

// readAllLocked parses the current WAL from after the magic prefix to EOF.
// The file offset is restored to end-of-file before return so subsequent
// Appends land after the existing data. Caller must hold w.mu.
//
// Validates the magic prefix on every call: if the file has shrunk below
// the magic length or the magic mismatches, ErrWALCorrupt is returned with
// an empty entry slice.
func (w *WAL) readAllLocked() ([]LogEntry, error) {
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("raft wal: seek: %w", err)
	}
	defer func() {
		_, _ = w.file.Seek(0, io.SeekEnd)
	}()

	var magic [walMagicSize]byte
	if _, err := io.ReadFull(w.file, magic[:]); err != nil {
		if errors.Is(err, io.EOF) {
			// Empty file: treat as empty WAL. The magic has not been
			// written yet; callers may choose to reinitialize via reopen.
			return nil, nil
		}
		return nil, ErrWALCorrupt
	}
	if !bytes.Equal(magic[:], walMagic[:]) {
		return nil, ErrWALCorrupt
	}

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
