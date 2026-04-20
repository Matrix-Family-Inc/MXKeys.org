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
	"crypto/subtle"
	"encoding/hex"
)

// ConsistencyProof proves that an older tree is a prefix of a newer tree.
type ConsistencyProof struct {
	OldSize   int      `json:"old_size"`
	NewSize   int      `json:"new_size"`
	OldRoot   string   `json:"old_root"`
	NewRoot   string   `json:"new_root"`
	ProofPath []string `json:"proof_path"`
}

// GetConsistencyProof generates a proof that the tree grew consistently
// (append-only) from oldSize leaves to the current size.
// Follows the subproof algorithm from RFC 6962 section 2.1.2.
func (t *Tree) GetConsistencyProof(oldSize int) (*ConsistencyProof, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if oldSize <= 0 || oldSize > len(t.leaves) {
		return nil, ErrInvalidIndex
	}

	oldTree := NewFromHashes(t.leaves[:oldSize])

	return &ConsistencyProof{
		OldSize:   oldSize,
		NewSize:   len(t.leaves),
		OldRoot:   oldTree.RootHex(),
		NewRoot:   hex.EncodeToString(t.root),
		ProofPath: consistencyPath(oldSize, len(t.leaves), t.levels),
	}, nil
}

// VerifyConsistencyProof verifies that an old root is consistent with a new root.
// Returns (false, ErrInvalidProof) on malformed input and (false, nil) when the
// reconstructed roots do not match constant-time.
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

	// RFC 6962 verification: reconstruct both roots from the proof path.
	fn := proof.OldSize - 1
	sn := proof.NewSize - 1

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
// Walks the tree structure: for the promote-odd-node scheme, reconstruct both
// subtrees and collect boundary hashes.
func consistencyPath(m, n int, levels [][][]byte) []string {
	if m == n || len(levels) == 0 {
		return nil
	}

	var path []string
	oldTree := NewFromHashes(levels[0][:m])
	path = append(path, oldTree.RootHex())

	subM := m
	subN := n
	level := 0

	for subM < subN && level < len(levels)-1 {
		levelNodes := levels[level]
		if subM < len(levelNodes) && subM%2 == 0 && subM+1 <= len(levelNodes) {
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
