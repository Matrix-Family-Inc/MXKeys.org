package merkle

import (
	"encoding/hex"
	"testing"
)

func TestNewTree(t *testing.T) {
	tree := New()
	if tree == nil {
		t.Fatal("New() returned nil")
	}
	if tree.Size() != 0 {
		t.Errorf("Size() = %d, want 0", tree.Size())
	}
}

func TestAddLeaf(t *testing.T) {
	tree := New()

	idx := tree.Add([]byte("leaf1"))
	if idx != 0 {
		t.Errorf("first Add returned %d, want 0", idx)
	}

	if tree.Size() != 1 {
		t.Errorf("Size() = %d, want 1", tree.Size())
	}

	idx = tree.Add([]byte("leaf2"))
	if idx != 1 {
		t.Errorf("second Add returned %d, want 1", idx)
	}
}

func TestRootSingleLeaf(t *testing.T) {
	tree := New()
	tree.Add([]byte("single"))

	root := tree.Root()
	if root == nil {
		t.Fatal("Root() returned nil for single leaf")
	}

	if len(root) != 32 {
		t.Errorf("root length = %d, want 32 (SHA-256)", len(root))
	}
}

func TestRootDeterministic(t *testing.T) {
	tree1 := New()
	tree1.Add([]byte("a"))
	tree1.Add([]byte("b"))

	tree2 := New()
	tree2.Add([]byte("a"))
	tree2.Add([]byte("b"))

	if tree1.RootHex() != tree2.RootHex() {
		t.Error("same leaves should produce same root")
	}
}

func TestRootDifferentLeaves(t *testing.T) {
	tree1 := New()
	tree1.Add([]byte("a"))
	tree1.Add([]byte("b"))

	tree2 := New()
	tree2.Add([]byte("a"))
	tree2.Add([]byte("c"))

	if tree1.RootHex() == tree2.RootHex() {
		t.Error("different leaves should produce different root")
	}
}

func TestGetProofSingleLeaf(t *testing.T) {
	tree := New()
	tree.Add([]byte("single"))

	proof, err := tree.GetProof(0)
	if err != nil {
		t.Fatalf("GetProof failed: %v", err)
	}

	if proof.LeafIndex != 0 {
		t.Errorf("LeafIndex = %d, want 0", proof.LeafIndex)
	}

	if proof.TreeSize != 1 {
		t.Errorf("TreeSize = %d, want 1", proof.TreeSize)
	}

	if proof.RootHash == "" {
		t.Error("RootHash should not be empty")
	}
}

func TestGetProofMultipleLeaves(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))
	tree.Add([]byte("b"))
	tree.Add([]byte("c"))
	tree.Add([]byte("d"))

	for i := 0; i < 4; i++ {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Fatalf("GetProof(%d) failed: %v", i, err)
		}

		if proof.LeafIndex != i {
			t.Errorf("proof[%d].LeafIndex = %d", i, proof.LeafIndex)
		}

		if len(proof.AuditPath) == 0 {
			t.Errorf("proof[%d] should have audit path", i)
		}
	}
}

func TestVerifyProof(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))
	tree.Add([]byte("b"))
	tree.Add([]byte("c"))
	tree.Add([]byte("d"))

	for i := 0; i < 4; i++ {
		proof, err := tree.GetProof(i)
		if err != nil {
			t.Fatalf("GetProof(%d) failed: %v", i, err)
		}

		valid, err := VerifyProof(proof)
		if err != nil {
			t.Fatalf("VerifyProof(%d) failed: %v", i, err)
		}

		if !valid {
			t.Errorf("proof for leaf %d should be valid", i)
		}
	}
}

func TestVerifyProofInvalid(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))
	tree.Add([]byte("b"))

	proof, _ := tree.GetProof(0)

	// Tamper with leaf hash
	proof.LeafHash = "0000000000000000000000000000000000000000000000000000000000000000"

	valid, err := VerifyProof(proof)
	if err != nil {
		t.Fatalf("VerifyProof failed: %v", err)
	}

	if valid {
		t.Error("tampered proof should be invalid")
	}
}

func TestGetProofEmptyTree(t *testing.T) {
	tree := New()

	_, err := tree.GetProof(0)
	if err != ErrEmptyTree {
		t.Errorf("expected ErrEmptyTree, got %v", err)
	}
}

func TestGetProofInvalidIndex(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))

	_, err := tree.GetProof(5)
	if err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex, got %v", err)
	}

	_, err = tree.GetProof(-1)
	if err != ErrInvalidIndex {
		t.Errorf("expected ErrInvalidIndex, got %v", err)
	}
}

func TestOddNumberOfLeaves(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))
	tree.Add([]byte("b"))
	tree.Add([]byte("c"))

	root := tree.Root()
	if root == nil {
		t.Fatal("root should not be nil for odd leaves")
	}

	// Verify all proofs work
	for i := 0; i < 3; i++ {
		proof, _ := tree.GetProof(i)
		valid, _ := VerifyProof(proof)
		if !valid {
			t.Errorf("proof for leaf %d should be valid", i)
		}
	}
}

func TestLargeTree(t *testing.T) {
	tree := New()

	for i := 0; i < 100; i++ {
		tree.Add([]byte{byte(i)})
	}

	if tree.Size() != 100 {
		t.Errorf("Size() = %d, want 100", tree.Size())
	}

	// Verify random proofs
	indices := []int{0, 1, 50, 99}
	for _, idx := range indices {
		proof, err := tree.GetProof(idx)
		if err != nil {
			t.Fatalf("GetProof(%d) failed: %v", idx, err)
		}

		valid, err := VerifyProof(proof)
		if err != nil {
			t.Fatalf("VerifyProof(%d) failed: %v", idx, err)
		}

		if !valid {
			t.Errorf("proof for leaf %d should be valid", idx)
		}
	}
}

func TestHashData(t *testing.T) {
	data := []byte("test")
	hash := HashData(data)

	if len(hash) != 32 {
		t.Errorf("hash length = %d, want 32", len(hash))
	}

	// Same input = same output
	hash2 := HashData(data)
	if hex.EncodeToString(hash) != hex.EncodeToString(hash2) {
		t.Error("HashData should be deterministic")
	}
}

func TestHashDataHex(t *testing.T) {
	data := []byte("test")
	hashHex := HashDataHex(data)

	if len(hashHex) != 64 {
		t.Errorf("hex hash length = %d, want 64", len(hashHex))
	}

	// Verify it's valid hex
	_, err := hex.DecodeString(hashHex)
	if err != nil {
		t.Errorf("invalid hex: %v", err)
	}
}

func TestNewFromHashes(t *testing.T) {
	hashes := [][]byte{
		HashData([]byte("a")),
		HashData([]byte("b")),
		HashData([]byte("c")),
	}

	tree := NewFromHashes(hashes)

	if tree.Size() != 3 {
		t.Errorf("Size() = %d, want 3", tree.Size())
	}

	if tree.Root() == nil {
		t.Error("root should not be nil")
	}
}

func TestConsistencyProof(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))
	tree.Add([]byte("b"))

	oldRoot := tree.RootHex()

	tree.Add([]byte("c"))
	tree.Add([]byte("d"))

	proof, err := tree.GetConsistencyProof(2)
	if err != nil {
		t.Fatalf("GetConsistencyProof failed: %v", err)
	}

	if proof.OldSize != 2 {
		t.Errorf("OldSize = %d, want 2", proof.OldSize)
	}

	if proof.NewSize != 4 {
		t.Errorf("NewSize = %d, want 4", proof.NewSize)
	}

	if proof.OldRoot != oldRoot {
		t.Error("OldRoot mismatch")
	}
}

func TestStats(t *testing.T) {
	tree := New()
	tree.Add([]byte("a"))
	tree.Add([]byte("b"))
	tree.Add([]byte("c"))
	tree.Add([]byte("d"))

	stats := tree.Stats()

	if stats["leaves"].(int) != 4 {
		t.Errorf("leaves = %v, want 4", stats["leaves"])
	}

	if stats["root"].(string) == "" {
		t.Error("root should not be empty")
	}
}

func TestAddHashDirect(t *testing.T) {
	tree := New()

	hash := HashData([]byte("precomputed"))
	idx := tree.AddHash(hash)

	if idx != 0 {
		t.Errorf("AddHash returned %d, want 0", idx)
	}

	if tree.Size() != 1 {
		t.Errorf("Size() = %d, want 1", tree.Size())
	}
}

func TestVerifyProofNil(t *testing.T) {
	valid, err := VerifyProof(nil)
	if err != ErrInvalidProof {
		t.Errorf("expected ErrInvalidProof, got %v", err)
	}
	if valid {
		t.Error("nil proof should be invalid")
	}
}
