//go:build integration
// +build integration

/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package integration

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"

	"mxkeys/internal/keys/keyprovider"
)

// TestKeyRotationEndToEnd exercises the documented key rotation
// procedure at the provider layer:
//
//  1. Plaintext install generates a key.
//  2. Operator enables at-rest encryption by setting a passphrase
//     and restarting; the legacy plaintext key is upgraded in place
//     without regenerating (critical: a regenerated key is a server
//     identity change, not a rotation).
//  3. After the upgrade, the plaintext file is gone and only the
//     encrypted file remains.
//  4. Subsequent loads with the correct passphrase return the SAME
//     private key bytes.
//  5. A passphrase change (rotation of the KEK) is performed by
//     re-encrypting the decrypted seed under a fresh passphrase.
//     The signing-key identity remains stable across the KEK
//     rotation.
func TestKeyRotationEndToEnd(t *testing.T) {
	dir := t.TempDir()

	// Step 1: plaintext install.
	p0, err := keyprovider.New(keyprovider.Config{
		Kind:        keyprovider.KindFile,
		StoragePath: dir,
	})
	if err != nil {
		t.Fatalf("plain New: %v", err)
	}
	priv0, _, err := p0.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("plain LoadOrGenerate: %v", err)
	}

	// Step 2: operator enables encryption.
	pass1 := []byte("first-operator-passphrase")
	p1, err := keyprovider.New(keyprovider.Config{
		Kind:        keyprovider.KindFile,
		StoragePath: dir,
		Passphrase:  pass1,
	})
	if err != nil {
		t.Fatalf("enc New: %v", err)
	}
	priv1, _, err := p1.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("enc LoadOrGenerate: %v", err)
	}
	if !ed25519.PrivateKey(priv1).Public().(ed25519.PublicKey).Equal(
		ed25519.PrivateKey(priv0).Public().(ed25519.PublicKey),
	) {
		t.Fatal("upgrade must preserve the server identity (public key changed)")
	}

	// Step 3: filesystem layout after upgrade.
	if _, err := os.Stat(filepath.Join(dir, "mxkeys_ed25519.key")); !os.IsNotExist(err) {
		t.Fatalf("plaintext key must be removed after upgrade, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mxkeys_ed25519.key.enc")); err != nil {
		t.Fatalf("encrypted key must exist after upgrade: %v", err)
	}

	// Step 4: reload with same passphrase returns same bytes.
	p1b, err := keyprovider.New(keyprovider.Config{
		Kind:        keyprovider.KindFile,
		StoragePath: dir,
		Passphrase:  pass1,
	})
	if err != nil {
		t.Fatalf("reload New: %v", err)
	}
	priv1b, _, err := p1b.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("reload LoadOrGenerate: %v", err)
	}
	if !equalBytes(priv1, priv1b) {
		t.Fatal("reload must return the same private key bytes")
	}

	// Step 5: KEK rotation. Simulate the documented procedure:
	//   a) decrypt with the old passphrase,
	//   b) save the seed to a staging file,
	//   c) re-encrypt under a new passphrase by rewriting in a new
	//      temp dir seeded with the plaintext seed file, then
	//      enabling encryption with the new passphrase.
	// The resulting provider MUST return the same private key as
	// priv1 so relying homeservers see no identity change.
	stage := t.TempDir()
	seed := ed25519.PrivateKey(priv1).Seed()
	if err := os.WriteFile(filepath.Join(stage, "mxkeys_ed25519.key"), seed, 0o600); err != nil {
		t.Fatalf("seed staging: %v", err)
	}

	pass2 := []byte("rotated-operator-passphrase")
	p2, err := keyprovider.New(keyprovider.Config{
		Kind:        keyprovider.KindFile,
		StoragePath: stage,
		Passphrase:  pass2,
	})
	if err != nil {
		t.Fatalf("rotated New: %v", err)
	}
	priv2, _, err := p2.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("rotated LoadOrGenerate: %v", err)
	}
	if !equalBytes(priv1, priv2) {
		t.Fatal("KEK rotation must preserve signing-key identity")
	}

	// Reload the rotated store with the OLD passphrase must fail.
	pfail, err := keyprovider.New(keyprovider.Config{
		Kind:        keyprovider.KindFile,
		StoragePath: stage,
		Passphrase:  pass1,
	})
	if err != nil {
		t.Fatalf("rotated New (old pass): %v", err)
	}
	if _, _, err := pfail.LoadOrGenerate(context.Background()); err == nil {
		t.Fatal("rotated store must refuse the old passphrase")
	}
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
