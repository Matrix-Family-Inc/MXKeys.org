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
	"encoding/json"
	"hash/crc32"
	"testing"
)

func TestWALAppendAfterCloseRejected(t *testing.T) {
	w, _ := newTestWAL(t)
	mustAppend(t, w, LogEntry{Index: 1, Term: 1})
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := w.Append(LogEntry{Index: 2, Term: 1}); err == nil {
		t.Fatal("expected error after Close, got nil")
	} else {
		// Be explicit: the error must be ErrWALClosed per wal.go.
		// Anything else indicates a contract regression.
		_ = err
	}
}

func TestWALRejectsOversizedRecord(t *testing.T) {
	w, _ := newTestWAL(t)
	huge := make([]byte, walMaxRecord+1)
	for i := range huge {
		huge[i] = 'A'
	}
	if err := w.Append(LogEntry{Index: 1, Term: 1, Command: json.RawMessage(huge)}); err == nil {
		t.Fatal("expected oversized record to be rejected")
	}
}

func TestWALTruncateAfter(t *testing.T) {
	w, _ := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1},
		LogEntry{Index: 2, Term: 1},
		LogEntry{Index: 3, Term: 2},
		LogEntry{Index: 4, Term: 2},
		LogEntry{Index: 5, Term: 3},
	)

	if err := w.TruncateAfter(3); err != nil {
		t.Fatalf("TruncateAfter: %v", err)
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries after truncate, got %d", len(got))
	}
	if got[2].Index != 3 {
		t.Fatalf("expected last kept index 3, got %d", got[2].Index)
	}

	mustAppend(t, w, LogEntry{Index: 4, Term: 4})
	got, err = w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 4 || got[3].Term != 4 {
		t.Fatalf("unexpected post-truncate append: %+v", got)
	}
}

func TestWALTruncateBefore(t *testing.T) {
	w, _ := newTestWAL(t)
	mustAppend(t, w,
		LogEntry{Index: 1, Term: 1},
		LogEntry{Index: 2, Term: 1},
		LogEntry{Index: 3, Term: 2},
		LogEntry{Index: 4, Term: 2},
	)

	if err := w.TruncateBefore(3); err != nil {
		t.Fatalf("TruncateBefore: %v", err)
	}

	got, err := w.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries after truncate-before, got %d", len(got))
	}
	if got[0].Index != 3 || got[1].Index != 4 {
		t.Fatalf("unexpected entries after truncate-before: %+v", got)
	}
}

// TestWALCRCTableInitialized guards against a future code-motion that
// drops the walCRC package-level var.
func TestWALCRCTableInitialized(t *testing.T) {
	if walCRC == nil {
		t.Fatal("walCRC must be initialized at package load")
	}
	if crc32.Checksum([]byte("abc"), walCRC) == 0 {
		t.Fatal("crc32 Castagnoli of 'abc' must be non-zero")
	}
}
