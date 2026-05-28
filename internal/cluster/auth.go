/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package cluster

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"mxkeys/internal/zero/canonical"
)

const maxMessageSkew = 5 * time.Minute

// signMessage computes an HMAC-SHA256 signature over the canonical JSON
// encoding of the message fields (Type, From, Timestamp, Payload) and writes
// it into msg.Signature.
//
// Canonical JSON is used instead of ad-hoc string formatting so structural
// ambiguity (e.g. "|" inside From or a different encoding of Payload) cannot
// produce collisions across different messages.
func (c *Cluster) signMessage(msg *ClusterMessage) error {
	payload, err := messageMACPayload(msg)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, []byte(c.config.SharedSecret))
	if _, err := mac.Write(payload); err != nil {
		return err
	}
	msg.Signature = hex.EncodeToString(mac.Sum(nil))
	return nil
}

// verifyMessage rejects messages with missing signatures, skew beyond the
// allowed window, invalid MACs, or replayed MACs within the freshness window.
func (c *Cluster) verifyMessage(msg *ClusterMessage) error {
	if msg.Signature == "" {
		return fmt.Errorf("cluster message signature is missing")
	}
	if skew := time.Since(msg.Timestamp); skew > maxMessageSkew || skew < -maxMessageSkew {
		return fmt.Errorf("cluster message timestamp outside allowed skew")
	}

	expected := &ClusterMessage{
		Type:      msg.Type,
		From:      msg.From,
		Timestamp: msg.Timestamp,
		Payload:   msg.Payload,
	}
	if err := c.signMessage(expected); err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected.Signature), []byte(msg.Signature)) {
		return fmt.Errorf("cluster message signature mismatch")
	}
	if !c.trackMessageMAC(msg.Signature, msg.Timestamp) {
		return fmt.Errorf("cluster message replay detected")
	}
	return nil
}

// messageMACPayload encodes the MACed subset of a ClusterMessage as canonical
// JSON. The Signature field is deliberately excluded to avoid a chicken-and-egg
// problem. Timestamp is serialized as RFC3339 nanosecond string for exact
// cross-peer reproducibility without hitting canonical JSON's safe-integer
// range (current Unix nanoseconds exceed 2^53). Payload is hex-encoded to
// guarantee a deterministic string representation.
func messageMACPayload(msg *ClusterMessage) ([]byte, error) {
	return canonical.Marshal(map[string]interface{}{
		"type":        string(msg.Type),
		"from":        msg.From,
		"timestamp":   msg.Timestamp.UTC().Format(time.RFC3339Nano),
		"payload_hex": hex.EncodeToString([]byte(msg.Payload)),
	})
}

func (c *Cluster) trackMessageMAC(signature string, timestamp time.Time) bool {
	c.replayMu.Lock()
	defer c.replayMu.Unlock()

	cutoff := time.Now().Add(-maxMessageSkew)
	for seenSignature, seenAt := range c.seenMACs {
		if seenAt.Before(cutoff) {
			delete(c.seenMACs, seenSignature)
		}
	}

	if seenAt, exists := c.seenMACs[signature]; exists && !seenAt.Before(cutoff) {
		return false
	}

	c.seenMACs[signature] = timestamp
	return true
}
