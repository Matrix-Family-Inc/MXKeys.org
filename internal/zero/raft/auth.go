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
	"encoding/hex"
	"fmt"
	"time"

	"mxkeys/internal/zero/canonical"
)

// signRPC computes an HMAC-SHA256 signature over the canonical JSON encoding
// of the MACed message fields (Type, From, Timestamp, Payload). See cluster
// auth for the rationale behind canonical JSON vs ad-hoc string formatting.
func (n *Node) signRPC(msg *RPCMessage) error {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	payload, err := rpcMACPayload(msg)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, []byte(n.config.SharedSecret))
	if _, err := mac.Write(payload); err != nil {
		return err
	}
	msg.Signature = hex.EncodeToString(mac.Sum(nil))
	return nil
}

// rpcMACPayload encodes the MACed subset of an RPCMessage as canonical JSON.
// Signature is deliberately excluded. Timestamp is RFC3339Nano text to avoid
// canonical JSON's safe-integer range (Unix nanoseconds exceed 2^53). Payload
// is hex-encoded for deterministic string representation.
func rpcMACPayload(msg *RPCMessage) ([]byte, error) {
	return canonical.Marshal(map[string]interface{}{
		"type":        string(msg.Type),
		"from":        msg.From,
		"timestamp":   msg.Timestamp.UTC().Format(time.RFC3339Nano),
		"payload_hex": hex.EncodeToString([]byte(msg.Payload)),
	})
}

// verifyRPC rejects messages with missing signatures, skew beyond the allowed
// window, invalid MACs, or replayed MACs within the freshness window.
func (n *Node) verifyRPC(msg *RPCMessage) error {
	if msg.Signature == "" {
		return fmt.Errorf("raft rpc signature is missing")
	}
	if skew := time.Since(msg.Timestamp); skew > maxRPCSkew || skew < -maxRPCSkew {
		return fmt.Errorf("raft rpc timestamp outside allowed skew")
	}
	expected := &RPCMessage{
		Type:      msg.Type,
		From:      msg.From,
		Timestamp: msg.Timestamp,
		Payload:   msg.Payload,
	}
	if err := n.signRPC(expected); err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected.Signature), []byte(msg.Signature)) {
		return fmt.Errorf("raft rpc signature mismatch")
	}
	if !n.trackRPCSignature(msg.Signature, msg.Timestamp) {
		return fmt.Errorf("raft rpc replay detected")
	}
	return nil
}

func (n *Node) trackRPCSignature(signature string, timestamp time.Time) bool {
	n.replayMu.Lock()
	defer n.replayMu.Unlock()

	cutoff := time.Now().Add(-maxRPCSkew)
	for seenSignature, seenAt := range n.seenRPCs {
		if seenAt.Before(cutoff) {
			delete(n.seenRPCs, seenSignature)
		}
	}

	if seenAt, exists := n.seenRPCs[signature]; exists && !seenAt.Before(cutoff) {
		return false
	}

	n.seenRPCs[signature] = timestamp
	return true
}
