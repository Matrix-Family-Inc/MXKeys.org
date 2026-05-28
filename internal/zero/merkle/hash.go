/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package merkle

import (
	"crypto/sha256"
	"encoding/hex"
)

// hashLeaf hashes a leaf with a 0x00 prefix for domain separation.
func hashLeaf(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	return h.Sum(nil)
}

// hashNode hashes two child nodes with a 0x01 prefix for domain separation.
func hashNode(left, right []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// HashData computes SHA-256 of data (for external use).
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashDataHex computes SHA-256 of data and returns a hex string.
func HashDataHex(data []byte) string {
	return hex.EncodeToString(HashData(data))
}
