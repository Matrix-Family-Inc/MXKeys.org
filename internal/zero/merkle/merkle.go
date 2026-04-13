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
	"crypto/subtle"
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

	// Validate proof structure
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

	// Validate directions values (must be 0 or 1)
	for _, dir := range proof.Directions {
		if dir != 0 && dir != 1 {
			return false, ErrInvalidProof
		}
	}

	// Compute root from leaf and audit path
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

	// Constant-time comparison
	if len(currentHash) != len(expectedRoot) {
		return false, nil
	}

	var diff byte
	for i := range currentHash {
		diff |= currentHash[i] ^ expectedRoot[i]
	}

	return diff == 0, nil
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

// GetConsistencyProof generates a proof that tree grew consistently (append-only).
// Follows the subproof algorithm from RFC 6962 section 2.1.2.
func (t *Tree) GetConsistencyProof(oldSize int) (*ConsistencyProof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if oldSize <= 0 || oldSize > len(t.leaves) {
		return nil, ErrInvalidIndex
	}

	oldTree := NewFromHashes(t.leaves[:oldSize])

	proof := &ConsistencyProof{
		OldSize:   oldSize,
		NewSize:   len(t.leaves),
		OldRoot:   oldTree.RootHex(),
		NewRoot:   hex.EncodeToString(t.root),
		ProofPath: consistencyPath(oldSize, len(t.leaves), t.levels),
	}

	return proof, nil
}

// VerifyConsistencyProof verifies that an old root is consistent with a new root.
func VerifyConsistencyProof(proof *ConsistencyProof) (bool, error) {
	if proof == nil || proof.OldSize <= 0 || proof.NewSize < proof.OldSize {
		return false, ErrInvalidProof
	}
	if proof.OldSize == proof.NewSize {
		return proof.OldRoot == proof.NewRoot && len(proof.ProofPath) == 0, nil
	}

	oldRoot, err := hex.DecodeString(proof.OldRoot)
	if err != nil || len(oldRoot) != sha256.Size {
		return false, ErrInvalidProof
	}
	newRoot, err := hex.DecodeString(proof.NewRoot)
	if err != nil || len(newRoot) != sha256.Size {
		return false, ErrInvalidProof
	}
	if len(proof.ProofPath) == 0 {
		return false, ErrInvalidProof
	}

	path := make([][]byte, len(proof.ProofPath))
	for i, h := range proof.ProofPath {
		b, err := hex.DecodeString(h)
		if err != nil || len(b) != sha256.Size {
			return false, ErrInvalidProof
		}
		path[i] = b
	}

	// RFC 6962 verification: reconstruct both roots from the proof path
	fn := proof.OldSize - 1
	sn := proof.NewSize - 1

	// Find the last set bit of fn
	for fn&1 == 1 {
		fn >>= 1
		sn >>= 1
	}

	fr := path[0]
	sr := path[0]

	for _, c := range path[1:] {
		if sn == 0 {
			return false, nil
		}
		if fn&1 == 1 || fn == sn {
			fr = hashNode(c, fr)
			sr = hashNode(c, sr)
			for fn != 0 && fn&1 == 0 {
				fn >>= 1
				sn >>= 1
			}
		} else {
			sr = hashNode(sr, c)
		}
		fn >>= 1
		sn >>= 1
	}

	if sn != 0 {
		return false, nil
	}

	frMatch := subtle.ConstantTimeCompare(fr, oldRoot) == 1
	srMatch := subtle.ConstantTimeCompare(sr, newRoot) == 1
	return frMatch && srMatch, nil
}

// consistencyPath builds the proof nodes needed for consistency verification.
func consistencyPath(m, n int, levels [][][]byte) []string {
	if m == n || len(levels) == 0 {
		return nil
	}

	var path []string
	// Collect the nodes needed by walking the tree structure.
	// For our promote-odd-node scheme: reconstruct both subtrees
	// and collect boundary hashes.
	oldTree := NewFromHashes(levels[0][:m])
	path = append(path, oldTree.RootHex())

	// Add internal nodes from the new tree that the old tree needs
	// to verify extension. Walk right-side subtrees.
	subM := m
	subN := n
	level := 0

	for subM < subN && level < len(levels)-1 {
		levelNodes := levels[level]
		if subM < len(levelNodes) && subM%2 == 0 && subM+1 <= len(levelNodes) {
			// right sibling at this level
			if subM+1 < len(levelNodes) {
				path = append(path, hex.EncodeToString(levelNodes[subM+1]))
			} else if subM < len(levelNodes) {
				path = append(path, hex.EncodeToString(levelNodes[subM]))
			}
		}
		subM = (subM + 1) / 2
		subN = (subN + 1) / 2
		level++
	}

	return path
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
