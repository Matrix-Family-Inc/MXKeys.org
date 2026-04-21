/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package walupgrade

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
)

// writeV2 drops a minimal v2 WAL containing the given records.
func writeV2(t *testing.T, path string, records [][]byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(walMagicV2[:]); err != nil {
		t.Fatalf("write magic: %v", err)
	}
	for _, r := range records {
		var hdr [walV2HeaderSize]byte
		binary.LittleEndian.PutUint32(hdr[0:4], uint32(len(r)))
		binary.LittleEndian.PutUint32(hdr[4:8], crc32.Checksum(r, walCRC))
		if _, err := f.Write(hdr[:]); err != nil {
			t.Fatalf("write hdr: %v", err)
		}
		if _, err := f.Write(r); err != nil {
			t.Fatalf("write payload: %v", err)
		}
	}
}

// readV3Payloads walks the v3 file at path and returns the raw
// payload bytes, verifying CRC and HMAC for every record as it goes.
func readV3Payloads(t *testing.T, path string, key []byte) [][]byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v3: %v", err)
	}
	if !bytes.Equal(raw[:walMagicSize], walMagicV3[:]) {
		t.Fatalf("v3 magic mismatch: got %q", raw[:walMagicSize])
	}
	var out [][]byte
	pos := walMagicSize
	for pos < len(raw) {
		if pos+walV3HeaderSize > len(raw) {
			t.Fatalf("short header at pos %d", pos)
		}
		length := int(binary.LittleEndian.Uint32(raw[pos : pos+4]))
		crcVal := binary.LittleEndian.Uint32(raw[pos+4 : pos+8])
		macField := raw[pos+8 : pos+walV3HeaderSize]
		payload := raw[pos+walV3HeaderSize : pos+walV3HeaderSize+length]

		if crc32.Checksum(payload, walCRC) != crcVal {
			t.Fatalf("CRC mismatch at pos %d", pos)
		}
		mac := hmac.New(sha256.New, key)
		mac.Write(raw[pos : pos+8])
		mac.Write(payload)
		if !hmac.Equal(mac.Sum(nil), macField) {
			t.Fatalf("HMAC mismatch at pos %d", pos)
		}
		out = append(out, append([]byte(nil), payload...))
		pos += walV3HeaderSize + length
	}
	return out
}

func TestUpgradeHappyPath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, walFileName)

	want := [][]byte{[]byte("a"), []byte("bb"), []byte("three bytes")}
	writeV2(t, src, want)

	key := []byte("integration-cluster-secret-ok-enough")
	rep, err := Upgrade(Options{Dir: dir, HMACKey: key, KeepV2: true})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if rep.Records != len(want) {
		t.Fatalf("report.Records = %d, want %d", rep.Records, len(want))
	}
	if rep.V3Path != src {
		t.Fatalf("report.V3Path = %q, want %q", rep.V3Path, src)
	}
	if _, err := os.Stat(rep.V2BackupPath); err != nil {
		t.Fatalf("backup must exist: %v", err)
	}

	got := readV3Payloads(t, src, key)
	if len(got) != len(want) {
		t.Fatalf("got %d records, want %d", len(got), len(want))
	}
	for i := range want {
		if !bytes.Equal(got[i], want[i]) {
			t.Fatalf("record %d mismatch: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestUpgradeRejectsV3Input(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, walFileName)
	// Write a v3-magic file so Upgrade detects the attempt.
	if err := os.WriteFile(src, walMagicV3[:], 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := Upgrade(Options{Dir: dir, HMACKey: []byte("k")})
	if err == nil {
		t.Fatal("expected Upgrade to reject already-v3 file")
	}
}

func TestUpgradeStopsAtTornTail(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, walFileName)
	writeV2(t, src, [][]byte{[]byte("good")})
	// Append garbage bytes that look like a partial header.
	f, err := os.OpenFile(src, os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := f.Write([]byte{0x00, 0x01, 0x02}); err != nil {
		t.Fatalf("append: %v", err)
	}
	_ = f.Close()

	key := []byte("key")
	rep, err := Upgrade(Options{Dir: dir, HMACKey: key})
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if rep.Records != 1 {
		t.Fatalf("expected 1 record, got %d", rep.Records)
	}
}

func TestUpgradeRejectsEmptyKey(t *testing.T) {
	dir := t.TempDir()
	if _, err := Upgrade(Options{Dir: dir}); err == nil {
		t.Fatal("expected error for empty HMACKey")
	}
}
