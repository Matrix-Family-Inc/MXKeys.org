/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package cluster

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

const maxMessageSkew = 5 * time.Minute

func (c *Cluster) signMessage(msg *ClusterMessage) error {
	payload, err := c.messageMACPayload(msg)
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

func (c *Cluster) messageMACPayload(msg *ClusterMessage) ([]byte, error) {
	return []byte(fmt.Sprintf("%s|%s|%d|%x", msg.Type, msg.From, msg.Timestamp.UnixNano(), []byte(msg.Payload))), nil
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
