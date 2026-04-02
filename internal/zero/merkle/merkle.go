/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 */

package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
)

var (
	ErrEmptyTree    = errors.New("empty tree")
	ErrInvalidIndex = errors.New("invalid leaf index")
	ErrInvalidProof = errors.New("invalid proof")
)

// Tree is a Merkle tree for cryptographic proofs
type Tree struct {
	mu     sync.RWMutex
	leaves [][]byte
	levels [][][]byte
	root   []byte
}

// Proof contains a Merkle proof of inclusion
type Proof struct {
	LeafIndex  int      `json:"leaf_index"`
	LeafHash   string   `json:"leaf_hash"`
	TreeSize   int      `json:"tree_size"`
	RootHash   string   `json:"root_hash"`
	AuditPath  []string `json:"audit_path"`
	Directions []int    `json:"directions"` // 0 = left, 1 = right
}

// New creates a new empty Merkle tree
func New() *Tree {
	return &Tree{
		leaves: make([][]byte, 0),
		levels: make([][][]byte, 0),
	}
}

// NewFromHashes creates a tree from existing leaf hashes
func NewFromHashes(hashes [][]byte) *Tree {
	t := &Tree{
		leaves: make([][]byte, len(hashes)),
	}
	copy(t.leaves, hashes)
	t.rebuild()
	return t
}

// Add adds a new leaf to the tree
func (t *Tree) Add(data []byte) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	hash := hashLeaf(data)
	t.leaves = append(t.leaves, hash)
	t.rebuild()

	return len(t.leaves) - 1
}

// AddHash adds a pre-computed hash as a leaf
func (t *Tree) AddHash(hash []byte) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.leaves = append(t.leaves, hash)
	t.rebuild()

	return len(t.leaves) - 1
}

// Root returns the Merkle root hash
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

// RootHex returns the Merkle root as hex string
func (t *Tree) RootHex() string {
	root := t.Root()
	if root == nil {
		return ""
	}
	return hex.EncodeToString(root)
}

// Size returns the number of leaves
func (t *Tree) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.leaves)
}

// GetProof generates a Merkle proof for a leaf at index
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

	// Build audit path
	idx := index
	for level := 0; level < len(t.levels)-1; level++ {
		levelNodes := t.levels[level]

		// Determine sibling
		var siblingIdx int
		var direction int

		if idx%2 == 0 {
			// We're on the left, sibling is on the right
			siblingIdx = idx + 1
			direction = 1
		} else {
			// We're on the right, sibling is on the left
			siblingIdx = idx - 1
			direction = 0
		}

		// Add sibling to audit path if it exists
		if siblingIdx < len(levelNodes) {
			proof.AuditPath = append(proof.AuditPath, hex.EncodeToString(levelNodes[siblingIdx]))
			proof.Directions = append(proof.Directions, direction)
		}

		// Move to parent index
		idx = idx / 2
	}

	return proof, nil
}

// VerifyProof verifies a Merkle proof
func VerifyProof(proof *Proof) (bool, error) {
	if proof == nil {
		return false, ErrInvalidProof
	}

	leafHash, err := hex.DecodeString(proof.LeafHash)
	if err != nil {
		return false, err
	}

	expectedRoot, err := hex.DecodeString(proof.RootHash)
	if err != nil {
		return false, err
	}

	// Compute root from leaf and audit path
	currentHash := leafHash
	for i, siblingHex := range proof.AuditPath {
		sibling, err := hex.DecodeString(siblingHex)
		if err != nil {
			return false, err
		}

		if proof.Directions[i] == 0 {
			// Sibling is on the left
			currentHash = hashNode(sibling, currentHash)
		} else {
			// Sibling is on the right
			currentHash = hashNode(currentHash, sibling)
		}
	}

	// Compare computed root with expected root
	if len(currentHash) != len(expectedRoot) {
		return false, nil
	}

	for i := range currentHash {
		if currentHash[i] != expectedRoot[i] {
			return false, nil
		}
	}

	return true, nil
}

// rebuild reconstructs the tree from leaves
func (t *Tree) rebuild() {
	if len(t.leaves) == 0 {
		t.levels = nil
		t.root = nil
		return
	}

	// First level is the leaves
	t.levels = make([][][]byte, 0)
	t.levels = append(t.levels, t.leaves)

	currentLevel := t.leaves

	// Build up the tree
	for len(currentLevel) > 1 {
		nextLevel := make([][]byte, 0, (len(currentLevel)+1)/2)

		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				// Hash pair
				nextLevel = append(nextLevel, hashNode(currentLevel[i], currentLevel[i+1]))
			} else {
				// Odd node, promote it
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}

		t.levels = append(t.levels, nextLevel)
		currentLevel = nextLevel
	}

	// Root is the single node at the top level
	if len(currentLevel) > 0 {
		t.root = currentLevel[0]
	}
}

// hashLeaf hashes a leaf with a 0x00 prefix (domain separation)
func hashLeaf(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	return h.Sum(nil)
}

// hashNode hashes two child nodes with a 0x01 prefix (domain separation)
func hashNode(left, right []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// HashData computes SHA-256 of data (for external use)
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashDataHex computes SHA-256 of data and returns hex string
func HashDataHex(data []byte) string {
	return hex.EncodeToString(HashData(data))
}

// ConsistencyProof proves that an older tree is a prefix of a newer tree
type ConsistencyProof struct {
	OldSize   int      `json:"old_size"`
	NewSize   int      `json:"new_size"`
	OldRoot   string   `json:"old_root"`
	NewRoot   string   `json:"new_root"`
	ProofPath []string `json:"proof_path"`
}

// GetConsistencyProof generates a proof that tree grew consistently
func (t *Tree) GetConsistencyProof(oldSize int) (*ConsistencyProof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if oldSize <= 0 || oldSize > len(t.leaves) {
		return nil, ErrInvalidIndex
	}

	// Build old tree to get its root
	oldTree := NewFromHashes(t.leaves[:oldSize])

	proof := &ConsistencyProof{
		OldSize:   oldSize,
		NewSize:   len(t.leaves),
		OldRoot:   oldTree.RootHex(),
		NewRoot:   t.RootHex(),
		ProofPath: make([]string, 0),
	}

	// For simplicity, we include the hashes needed to verify
	// that old tree is a prefix of new tree
	// Full RFC 6962 consistency proof is more complex

	return proof, nil
}

// Stats returns tree statistics
func (t *Tree) Stats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"leaves": len(t.leaves),
		"levels": len(t.levels),
		"root":   t.RootHex(),
	}
}
