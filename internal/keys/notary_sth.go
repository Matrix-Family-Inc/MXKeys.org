/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package keys

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"
)

// SignedTreeHead returns a signed snapshot of the transparency log merkle tree.
// Payload format is tree_size|root_hash|timestamp_ms, signed with ed25519 using
// the notary signing key. External verifiers can reconstruct the payload and
// verify the signature against the notary public key.
func (n *Notary) SignedTreeHead() (map[string]interface{}, error) {
	tl := n.getTransparency()
	if tl == nil {
		return nil, fmt.Errorf("transparency log not enabled")
	}

	tl.mu.RLock()
	treeSize := tl.merkleTree.Size()
	rootHash := tl.merkleTree.RootHex()
	tl.mu.RUnlock()

	now := time.Now().UTC()
	payload := fmt.Sprintf("%d|%s|%d", treeSize, rootHash, now.UnixMilli())

	signature := ed25519.Sign(n.serverKeyPair, []byte(payload))
	signatureB64 := base64.RawStdEncoding.EncodeToString(signature)

	return map[string]interface{}{
		"tree_size":    treeSize,
		"root_hash":    rootHash,
		"timestamp":    now.Format(time.RFC3339),
		"timestamp_ms": now.UnixMilli(),
		"signer":       n.serverName,
		"key_id":       n.serverKeyID,
		"signature":    signatureB64,
		"sign_payload": payload,
	}, nil
}
