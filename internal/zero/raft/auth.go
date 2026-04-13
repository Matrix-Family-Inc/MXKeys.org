/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package raft

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

func (n *Node) signRPC(msg *RPCMessage) error {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	mac := hmac.New(sha256.New, []byte(n.config.SharedSecret))
	if _, err := mac.Write([]byte(fmt.Sprintf("%s|%s|%d|%x", msg.Type, msg.From, msg.Timestamp.UnixNano(), []byte(msg.Payload)))); err != nil {
		return err
	}
	msg.Signature = hex.EncodeToString(mac.Sum(nil))
	return nil
}

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
