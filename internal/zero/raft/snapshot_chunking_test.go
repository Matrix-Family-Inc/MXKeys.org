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
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// TestInstallSnapshotChunkingReassembles feeds three contiguous chunks
// into handleInstallSnapshot and verifies that the installer sees the
// reassembled payload in order, only once, on the Done=true chunk.
func TestInstallSnapshotChunkingReassembles(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "follower",
		ElectionTimeout: 300 * time.Millisecond,
	})
	n.currentTerm = 3

	// Record what the installer receives; it must be called exactly
	// once with the concatenated bytes.
	var got []byte
	var calls int
	n.SetSnapshotInstaller(func(data []byte, idx, term uint64) error {
		calls++
		got = append([]byte(nil), data...)
		if idx != 42 || term != 3 {
			t.Errorf("installer got (idx=%d, term=%d), want (42, 3)", idx, term)
		}
		return nil
	})

	chunks := [][]byte{
		[]byte("first-"),
		[]byte("second-"),
		[]byte("third"),
	}
	total := bytes.Join(chunks, nil)

	var offset uint64
	for i, c := range chunks {
		req := InstallSnapshotRequest{
			Term:              3,
			LeaderID:          "leader",
			LastIncludedIndex: 42,
			LastIncludedTerm:  3,
			Offset:            offset,
			Done:              i == len(chunks)-1,
			Data:              c,
		}
		payload, _ := json.Marshal(req)
		n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})
		offset += uint64(len(c))
	}

	if calls != 1 {
		t.Fatalf("installer called %d times, want 1", calls)
	}
	if !bytes.Equal(got, total) {
		t.Fatalf("reassembled bytes = %q, want %q", got, total)
	}
}

// TestInstallSnapshotResetsOnOffsetZero validates the retry contract:
// if the leader retransmits from offset 0, the follower discards any
// partial buffer and starts fresh.
func TestInstallSnapshotResetsOnOffsetZero(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "follower",
		ElectionTimeout: 300 * time.Millisecond,
	})
	n.currentTerm = 1

	var installerData []byte
	n.SetSnapshotInstaller(func(data []byte, idx, term uint64) error {
		installerData = append([]byte(nil), data...)
		return nil
	})

	// Initial partial chunk.
	send := func(off uint64, data string, done bool) {
		req := InstallSnapshotRequest{
			Term:              1,
			LeaderID:          "leader",
			LastIncludedIndex: 10,
			LastIncludedTerm:  1,
			Offset:            off,
			Done:              done,
			Data:              []byte(data),
		}
		payload, _ := json.Marshal(req)
		n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})
	}

	send(0, "bad-start-", false)
	// Retry from 0 (simulates leader resending after timeout).
	send(0, "fresh-", false)
	send(6, "complete", true)

	if string(installerData) != "fresh-complete" {
		t.Fatalf("expected fresh-complete, got %q", installerData)
	}
}

// TestInstallSnapshotRejectsGappedOffset: offset 0 then skip to 100
// must reset the buffer, not append.
func TestInstallSnapshotRejectsGappedOffset(t *testing.T) {
	n := NewNode(Config{
		NodeID:          "follower",
		ElectionTimeout: 300 * time.Millisecond,
	})
	n.currentTerm = 1

	called := false
	n.SetSnapshotInstaller(func(data []byte, idx, term uint64) error {
		called = true
		return nil
	})

	send := func(off uint64, data string, done bool) {
		req := InstallSnapshotRequest{
			Term:              1,
			LeaderID:          "leader",
			LastIncludedIndex: 10,
			LastIncludedTerm:  1,
			Offset:            off,
			Done:              done,
			Data:              []byte(data),
		}
		payload, _ := json.Marshal(req)
		n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})
	}

	send(0, "ok", false)
	send(100, "gapped", true) // gap; must be ignored without calling installer
	if called {
		t.Fatal("installer must NOT be called for gapped offset; leader is expected to retry from 0")
	}
}

// TestSnapshotChunkSizeIsWithinWALLimit protects against a refactor
// that raises snapshotChunkSize above the WAL record budget.
func TestSnapshotChunkSizeIsWithinWALLimit(t *testing.T) {
	if snapshotChunkSize >= walMaxRecord {
		t.Fatalf("snapshotChunkSize %d >= walMaxRecord %d", snapshotChunkSize, walMaxRecord)
	}
}

// TestInstallSnapshotResponseSuccessContract exercises the Success flag
// across every terminal branch of handleInstallSnapshot. This is the
// protocol-level guard that prevents the leader from advancing
// nextIndex/matchIndex on a follower that did not actually install the
// snapshot.
func TestInstallSnapshotResponseSuccessContract(t *testing.T) {
	decode := func(msg *RPCMessage) InstallSnapshotResponse {
		t.Helper()
		var resp InstallSnapshotResponse
		if err := json.Unmarshal(msg.Payload, &resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return resp
	}

	t.Run("done chunk with installer success returns success=true", func(t *testing.T) {
		n := NewNode(Config{NodeID: "f", ElectionTimeout: 300 * time.Millisecond})
		n.currentTerm = 1
		n.SetSnapshotInstaller(func([]byte, uint64, uint64) error { return nil })

		req := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: 5, LastIncludedTerm: 1,
			Offset: 0, Done: true, Data: []byte("state"),
		}
		payload, _ := json.Marshal(req)
		resp := decode(n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload}))
		if !resp.Success {
			t.Fatalf("expected success=true for clean Done install, got %+v", resp)
		}
	})

	t.Run("done chunk with installer error returns success=false", func(t *testing.T) {
		n := NewNode(Config{NodeID: "f", ElectionTimeout: 300 * time.Millisecond})
		n.currentTerm = 1
		n.SetSnapshotInstaller(func([]byte, uint64, uint64) error { return errors.New("boom") })

		req := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: 5, LastIncludedTerm: 1,
			Offset: 0, Done: true, Data: []byte("state"),
		}
		payload, _ := json.Marshal(req)
		resp := decode(n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload}))
		if resp.Success {
			t.Fatalf("expected success=false when installer errors, got %+v", resp)
		}
	})

	t.Run("non-done chunk buffered returns success=true", func(t *testing.T) {
		n := NewNode(Config{NodeID: "f", ElectionTimeout: 300 * time.Millisecond})
		n.currentTerm = 1
		n.SetSnapshotInstaller(func([]byte, uint64, uint64) error { return nil })

		req := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: 5, LastIncludedTerm: 1,
			Offset: 0, Done: false, Data: []byte("partial"),
		}
		payload, _ := json.Marshal(req)
		resp := decode(n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload}))
		if !resp.Success {
			t.Fatalf("expected success=true for buffered chunk, got %+v", resp)
		}
		if resp.BytesStored != uint64(len(req.Data)) {
			t.Fatalf("bytes_stored = %d, want %d", resp.BytesStored, len(req.Data))
		}
	})

	t.Run("offset gap returns success=false", func(t *testing.T) {
		n := NewNode(Config{NodeID: "f", ElectionTimeout: 300 * time.Millisecond})
		n.currentTerm = 1

		first := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: 5, LastIncludedTerm: 1,
			Offset: 0, Done: false, Data: []byte("ok"),
		}
		payload, _ := json.Marshal(first)
		_ = n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload})

		gapped := first
		gapped.Offset = 100
		gapped.Done = true
		gapped.Data = []byte("gap")
		payload, _ = json.Marshal(gapped)
		resp := decode(n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload}))
		if resp.Success {
			t.Fatalf("expected success=false for gapped offset, got %+v", resp)
		}
	})

	t.Run("stale term returns success=false", func(t *testing.T) {
		n := NewNode(Config{NodeID: "f", ElectionTimeout: 300 * time.Millisecond})
		n.currentTerm = 5

		req := InstallSnapshotRequest{
			Term: 1, LeaderID: "L", LastIncludedIndex: 5, LastIncludedTerm: 1,
			Offset: 0, Done: true, Data: []byte("state"),
		}
		payload, _ := json.Marshal(req)
		resp := decode(n.handleInstallSnapshot(&RPCMessage{Type: MsgInstallSnapshot, Payload: payload}))
		if resp.Success {
			t.Fatalf("expected success=false for stale leader term, got %+v", resp)
		}
		if resp.Term != 5 {
			t.Fatalf("expected response term=5 (follower's current), got %d", resp.Term)
		}
	})
}
