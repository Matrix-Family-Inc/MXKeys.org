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

// Proof contains a Merkle proof of inclusion.
type Proof struct {
	LeafIndex  int      `json:"leaf_index"`
	LeafHash   string   `json:"leaf_hash"`
	TreeSize   int      `json:"tree_size"`
	RootHash   string   `json:"root_hash"`
	AuditPath  []string `json:"audit_path"`
	Directions []int    `json:"directions"` // 0 = left sibling, 1 = right sibling
}

// GetProof generates a Merkle proof for a leaf at the given index.
func (t *Tree) GetProof(index int) (*Proof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.leaves) == 0 {
		return nil, ErrEmptyTree
	}

	if index < 0 || index >= len(t.leaves) {
		return nil, ErrInvalidIndex
	}

	proof := &Proof{
		LeafIndex:  index,
		LeafHash:   hex.EncodeToString(t.leaves[index]),
		TreeSize:   len(t.leaves),
		RootHash:   hex.EncodeToString(t.root),
		AuditPath:  make([]string, 0),
		Directions: make([]int, 0),
	}

	idx := index
	for level := 0; level < len(t.levels)-1; level++ {
		levelNodes := t.levels[level]

		var siblingIdx int
		var direction int

		if idx%2 == 0 {
			siblingIdx = idx + 1
			direction = 1
		} else {
			siblingIdx = idx - 1
			direction = 0
		}

		if siblingIdx < len(levelNodes) {
			proof.AuditPath = append(proof.AuditPath, hex.EncodeToString(levelNodes[siblingIdx]))
			proof.Directions = append(proof.Directions, direction)
		}

		idx = idx / 2
	}

	return proof, nil
}

// VerifyProof verifies a Merkle proof of inclusion against its declared root.
// Returns (false, ErrInvalidProof) on malformed input and (false, nil) on
// structurally valid proofs whose recomputed root mismatches.
func VerifyProof(proof *Proof) (bool, error) {
	if proof == nil {
		return false, ErrInvalidProof
	}

	if len(proof.AuditPath) != len(proof.Directions) {
		return false, ErrInvalidProof
	}

	if proof.LeafIndex < 0 || proof.TreeSize <= 0 || proof.LeafIndex >= proof.TreeSize {
		return false, ErrInvalidProof
	}

	leafHash, err := hex.DecodeString(proof.LeafHash)
	if err != nil {
		return false, ErrInvalidProof
	}

	if len(leafHash) != sha256.Size {
		return false, ErrInvalidProof
	}

	expectedRoot, err := hex.DecodeString(proof.RootHash)
	if err != nil {
		return false, ErrInvalidProof
	}

	if len(expectedRoot) != sha256.Size {
		return false, ErrInvalidProof
	}

	for _, dir := range proof.Directions {
		if dir != 0 && dir != 1 {
			return false, ErrInvalidProof
		}
	}

	currentHash := leafHash
	for i, siblingHex := range proof.AuditPath {
		sibling, err := hex.DecodeString(siblingHex)
		if err != nil {
			return false, ErrInvalidProof
		}

		if len(sibling) != sha256.Size {
			return false, ErrInvalidProof
		}

		if proof.Directions[i] == 0 {
			currentHash = hashNode(sibling, currentHash)
		} else {
			currentHash = hashNode(currentHash, sibling)
		}
	}

	if len(currentHash) != len(expectedRoot) {
		return false, nil
	}

	var diff byte
	for i := range currentHash {
		diff |= currentHash[i] ^ expectedRoot[i]
	}

	return diff == 0, nil
}
