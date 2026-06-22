/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

package keys

import (
	"testing"

	"mxkeys/internal/zero/merkle"
)

func TestTransparencyMerkleProofLifecycle(t *testing.T) {
	tl := &TransparencyLog{
		enabled:    true,
		merkleTree: merkle.New(),
	}

	if err := tl.addMerkleHash(hashKey("entry-1")); err != nil {
		t.Fatalf("addMerkleHash(entry-1) failed: %v", err)
	}
	if err := tl.addMerkleHash(hashKey("entry-2")); err != nil {
		t.Fatalf("addMerkleHash(entry-2) failed: %v", err)
	}

	proof, err := tl.GetProof(1)
	if err != nil {
		t.Fatalf("GetProof(1) failed: %v", err)
	}
	if proof.TreeSize != 2 {
		t.Fatalf("proof.TreeSize = %d, want 2", proof.TreeSize)
	}
	ok, err := merkle.VerifyProof(proof)
	if err != nil {
		t.Fatalf("VerifyProof() failed: %v", err)
	}
	if !ok {
		t.Fatal("expected Merkle proof to verify")
	}
}

func TestTransparencyAddMerkleHashRejectsInvalidHex(t *testing.T) {
	tl := &TransparencyLog{
		enabled:    true,
		merkleTree: merkle.New(),
	}

	if err := tl.addMerkleHash("not-hex"); err == nil {
		t.Fatal("expected invalid hex hash to be rejected")
	}
}
