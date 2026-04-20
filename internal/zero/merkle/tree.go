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
	"encoding/hex"
	"errors"
	"sync"
)

var (
	ErrEmptyTree    = errors.New("empty tree")
	ErrInvalidIndex = errors.New("invalid leaf index")
	ErrInvalidProof = errors.New("invalid proof")
)

// Tree is a Merkle tree for cryptographic proofs.
type Tree struct {
	mu     sync.RWMutex
	leaves [][]byte
	levels [][][]byte
	root   []byte
}

// New creates a new empty Merkle tree.
func New() *Tree {
	return &Tree{
		leaves: make([][]byte, 0),
		levels: make([][][]byte, 0),
	}
}

// NewFromHashes creates a tree from existing leaf hashes.
func NewFromHashes(hashes [][]byte) *Tree {
	t := &Tree{
		leaves: make([][]byte, len(hashes)),
	}
	copy(t.leaves, hashes)
	t.rebuild()
	return t
}

// Add adds a new leaf to the tree by hashing raw data first.
func (t *Tree) Add(data []byte) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	hash := hashLeaf(data)
	t.leaves = append(t.leaves, hash)
	t.rebuild()

	return len(t.leaves) - 1
}

// AddHash adds a pre-computed hash as a leaf.
func (t *Tree) AddHash(hash []byte) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.leaves = append(t.leaves, hash)
	t.rebuild()

	return len(t.leaves) - 1
}

// Root returns a defensive copy of the Merkle root hash.
func (t *Tree) Root() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.root) == 0 {
		return nil
	}

	result := make([]byte, len(t.root))
	copy(result, t.root)
	return result
}

// RootHex returns the Merkle root as a hex string.
func (t *Tree) RootHex() string {
	root := t.Root()
	if root == nil {
		return ""
	}
	return hex.EncodeToString(root)
}

// Size returns the number of leaves.
func (t *Tree) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.leaves)
}

// Stats returns tree statistics.
func (t *Tree) Stats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"leaves": len(t.leaves),
		"levels": len(t.levels),
		"root":   t.RootHex(),
	}
}

// rebuild reconstructs the tree levels and root from leaves.
// Odd nodes at any level are promoted unchanged; pairs are hashed with
// domain separation (see hashNode).
func (t *Tree) rebuild() {
	if len(t.leaves) == 0 {
		t.levels = nil
		t.root = nil
		return
	}

	t.levels = make([][][]byte, 0)
	t.levels = append(t.levels, t.leaves)

	currentLevel := t.leaves

	for len(currentLevel) > 1 {
		nextLevel := make([][]byte, 0, (len(currentLevel)+1)/2)

		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				nextLevel = append(nextLevel, hashNode(currentLevel[i], currentLevel[i+1]))
			} else {
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}

		t.levels = append(t.levels, nextLevel)
		currentLevel = nextLevel
	}

	if len(currentLevel) > 0 {
		t.root = currentLevel[0]
	}
}
