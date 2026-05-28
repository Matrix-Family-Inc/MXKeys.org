/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// Package walupgrade converts a legacy MXKS_WAL_v2 file to the
// current v3 format. The conversion re-authenticates every record
// under the configured cluster secret and writes the result through
// a temp file with an atomic rename, so a crash mid-upgrade leaves
// the original file untouched.
//
// Layout differences between formats:
//
//	v2:  magic(12="MXKS_WAL_v2\x00") || records of  len(4) || crc32c(4) || payload
//	v3:  magic(12="MXKS_WAL_v3\x00") || records of  len(4) || crc32c(4) || hmac_sha256(32) || payload
//
// The upgrade is performed offline. The notary service MUST be stopped
// so that no process holds raft.wal open.
package walupgrade

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"os"
	"path/filepath"
)

const (
	walFileName  = "raft.wal"
	walV2BackupSuffix = ".v2-backup"
	walMagicSize = 12
	walV2HeaderSize = 8  // len(4) + crc(4)
	walV3HeaderSize = 40 // len(4) + crc(4) + hmac(32)
	walMaxRecord    = 8 << 20
)

var (
	walMagicV2 = [walMagicSize]byte{'M', 'X', 'K', 'S', '_', 'W', 'A', 'L', '_', 'v', '2', 0}
	walMagicV3 = [walMagicSize]byte{'M', 'X', 'K', 'S', '_', 'W', 'A', 'L', '_', 'v', '3', 0}
	walCRC     = crc32.MakeTable(crc32.Castagnoli)
)

// Options controls the upgrade.
type Options struct {
	// Dir holds raft.wal. The upgrade writes raft.wal.upgrade
	// and then atomically renames to raft.wal.
	Dir string

	// HMACKey is the cluster shared secret. Required; empty is a
	// hard error.
	HMACKey []byte

	// KeepV2 preserves the original file as raft.wal.v2-backup in
	// the same directory. Recommended for the first upgrade so the
	// operator has a rollback path.
	KeepV2 bool
}

// Report describes a successful upgrade.
type Report struct {
	Records      int
	V3Path       string
	V2BackupPath string
}

// ErrNotV2 is returned when the input file is not a MXKS_WAL_v2
// file. The most common cause is an already-upgraded v3 file.
var ErrNotV2 = errors.New("walupgrade: input is not a v2 WAL")

// Upgrade converts opts.Dir/raft.wal from v2 to v3 in place.
func Upgrade(opts Options) (Report, error) {
	if opts.Dir == "" {
		return Report{}, errors.New("walupgrade: dir is required")
	}
	if len(opts.HMACKey) == 0 {
		return Report{}, errors.New("walupgrade: HMACKey is required")
	}

	src := filepath.Join(opts.Dir, walFileName)
	records, err := readV2(src)
	if err != nil {
		return Report{}, err
	}

	tmp := src + ".upgrade"
	if err := writeV3(tmp, records, opts.HMACKey); err != nil {
		_ = os.Remove(tmp)
		return Report{}, err
	}

	var backupPath string
	if opts.KeepV2 {
		backupPath = src + walV2BackupSuffix
		if err := os.Rename(src, backupPath); err != nil {
			_ = os.Remove(tmp)
			return Report{}, fmt.Errorf("walupgrade: backup rename: %w", err)
		}
	} else {
		if err := os.Remove(src); err != nil {
			_ = os.Remove(tmp)
			return Report{}, fmt.Errorf("walupgrade: remove v2: %w", err)
		}
	}
	if err := os.Rename(tmp, src); err != nil {
		return Report{}, fmt.Errorf("walupgrade: rename v3: %w", err)
	}

	return Report{
		Records:      len(records),
		V3Path:       src,
		V2BackupPath: backupPath,
	}, nil
}

// readV2 pulls every well-formed record out of a v2 WAL. Truncated
// tails stop the read at the last well-formed record, matching the
// runtime reader's semantics.
func readV2(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("walupgrade: open %s: %w", path, err)
	}
	defer f.Close()

	var magic [walMagicSize]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return nil, fmt.Errorf("walupgrade: read magic: %w", err)
	}
	if bytes.Equal(magic[:], walMagicV3[:]) {
		return nil, fmt.Errorf("walupgrade: %s already in v3 format", path)
	}
	if !bytes.Equal(magic[:], walMagicV2[:]) {
		return nil, ErrNotV2
	}

	var records [][]byte
	var hdr [walV2HeaderSize]byte
	for {
		_, err := io.ReadFull(f, hdr[:])
		if errors.Is(err, io.EOF) {
			return records, nil
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			// Torn tail at the end of the file; stop cleanly.
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("walupgrade: read header: %w", err)
		}
		length := binary.LittleEndian.Uint32(hdr[0:4])
		declaredCRC := binary.LittleEndian.Uint32(hdr[4:8])
		if length == 0 || length > walMaxRecord {
			return records, nil
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(f, payload); err != nil {
			return records, nil
		}
		if crc32.Checksum(payload, walCRC) != declaredCRC {
			return records, nil
		}
		records = append(records, payload)
	}
}

// writeV3 creates a v3 WAL at path with the supplied records. Every
// record is HMAC'd with key. The file is fsynced before close so a
// crash does not leave the rename pointing at an empty file.
func writeV3(path string, records [][]byte, key []byte) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("walupgrade: create %s: %w", path, err)
	}
	if _, err := f.Write(walMagicV3[:]); err != nil {
		_ = f.Close()
		return fmt.Errorf("walupgrade: write magic: %w", err)
	}
	for i, payload := range records {
		if err := writeV3Record(f, payload, key); err != nil {
			_ = f.Close()
			return fmt.Errorf("walupgrade: record %d: %w", i, err)
		}
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("walupgrade: fsync: %w", err)
	}
	return f.Close()
}

func writeV3Record(w io.Writer, payload []byte, key []byte) error {
	n := len(payload)
	if n > walMaxRecord {
		return fmt.Errorf("walupgrade: record length %d exceeds cap %d", n, walMaxRecord)
	}
	if n < 0 || n > math.MaxUint32 {
		return fmt.Errorf("walupgrade: record length %d out of uint32 range", n)
	}
	nu32 := uint32(n)
	var hdr [walV3HeaderSize]byte
	binary.LittleEndian.PutUint32(hdr[0:4], nu32)
	binary.LittleEndian.PutUint32(hdr[4:8], crc32.Checksum(payload, walCRC))
	mac := hmac.New(sha256.New, key)
	mac.Write(hdr[0:8])
	mac.Write(payload)
	copy(hdr[8:], mac.Sum(nil))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	return nil
}
