/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package keyprovider

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPBKDF2Matches_RFC6070Vector(t *testing.T) {
	// RFC 6070 PBKDF2-HMAC-SHA256 test vector
	// P = "password", S = "salt", c = 1, dkLen = 32
	//   expected output prefix =
	//   12 0f b6 cf fc f8 b3 2c 43 e7 22 52 56 c4 f8 37 a8 65 48 c9
	got := pbkdf2SHA256([]byte("password"), []byte("salt"), 1, 32)
	want := []byte{
		0x12, 0x0f, 0xb6, 0xcf, 0xfc, 0xf8, 0xb3, 0x2c,
		0x43, 0xe7, 0x22, 0x52, 0x56, 0xc4, 0xf8, 0x37,
		0xa8, 0x65, 0x48, 0xc9,
	}
	if !bytes.Equal(got[:20], want) {
		t.Fatalf("pbkdf2(password, salt, 1) prefix mismatch:\n got  %x\n want %x", got[:20], want)
	}
}

// testIterations is the minimum iteration count accepted at read time.
// Using the floor keeps tests fast (milliseconds per PBKDF2 call) while
// still exercising the real KDF and AEAD paths.
const testIterations = minPBKDF2Iterations

// TestMain overrides the global iteration count for every test in this
// package. The one test that needs the real production default
// restores it explicitly.
func TestMain(m *testing.M) {
	pbkdf2Iterations = testIterations
	os.Exit(m.Run())
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := []byte("this is the 32-byte ed25519 seed")
	passphrase := []byte("s3cret-p@ssphr@se-long-enough")

	blob, err := encryptKeyWithIterations(plaintext, passphrase, testIterations)
	if err != nil {
		t.Fatalf("encryptKey: %v", err)
	}
	if !hasEncryptedMagic(blob) {
		t.Fatal("encrypted blob must start with MXKENC01 magic")
	}
	if len(blob) < encHeaderLen+len(plaintext) {
		t.Fatalf("blob too short: %d bytes", len(blob))
	}

	recovered, err := decryptKey(blob, passphrase)
	if err != nil {
		t.Fatalf("decryptKey: %v", err)
	}
	if !bytes.Equal(recovered, plaintext) {
		t.Fatalf("round-trip mismatch: %q != %q", recovered, plaintext)
	}
}

func TestDecryptRejectsWrongPassphrase(t *testing.T) {
	blob, err := encryptKeyWithIterations([]byte("payload"), []byte("right-passphrase"), testIterations)
	if err != nil {
		t.Fatalf("encryptKey: %v", err)
	}
	_, err = decryptKey(blob, []byte("wrong-passphrase"))
	if !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed, got %v", err)
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	blob, err := encryptKeyWithIterations([]byte("payload bytes"), []byte("passphrase"), testIterations)
	if err != nil {
		t.Fatalf("encryptKey: %v", err)
	}
	blob[encHeaderLen+1] ^= 0xFF
	if _, err := decryptKey(blob, []byte("passphrase")); !errors.Is(err, ErrDecryptFailed) {
		t.Fatalf("expected ErrDecryptFailed after tamper, got %v", err)
	}
}

func TestDecryptRejectsTamperedHeader(t *testing.T) {
	blob, err := encryptKeyWithIterations([]byte("payload"), []byte("passphrase"), testIterations)
	if err != nil {
		t.Fatalf("encryptKey: %v", err)
	}
	// Flip a bit inside the salt. Salt is both the KDF input and part
	// of the AAD; tampering either way must cause Open to fail.
	blob[encSaltOffset] ^= 0x01
	if _, err := decryptKey(blob, []byte("passphrase")); err == nil {
		t.Fatal("expected decryption failure after header tamper")
	}
}

func TestDecryptRejectsNoMagic(t *testing.T) {
	_, err := decryptKey([]byte("short junk"), []byte("pass"))
	if !errors.Is(err, ErrBadMagic) {
		t.Fatalf("expected ErrBadMagic, got %v", err)
	}
	plain := []byte("legacy plaintext key that does not start with magic...")
	_, err = decryptKey(plain, []byte("pass"))
	if !errors.Is(err, ErrBadMagic) {
		t.Fatalf("expected ErrBadMagic for plaintext, got %v", err)
	}
}

func TestDecryptRejectsNoPassphraseWhenEncrypted(t *testing.T) {
	blob, err := encryptKeyWithIterations([]byte("payload"), []byte("pass"), testIterations)
	if err != nil {
		t.Fatalf("encryptKey: %v", err)
	}
	if _, err := decryptKey(blob, nil); err == nil {
		t.Fatal("expected error when passphrase missing")
	}
}

func TestDecryptRejectsLowIterationCount(t *testing.T) {
	blob, err := encryptKeyWithIterations([]byte("payload"), []byte("pass"), testIterations)
	if err != nil {
		t.Fatalf("encryptKey: %v", err)
	}
	// Poke iterations=100 (below the 10k threshold). The implausible-
	// iteration guard runs before AEAD.Open, so it returns first.
	binary.BigEndian.PutUint32(blob[encIterationsOffset:encSaltOffset], 100)
	if _, err := decryptKey(blob, []byte("pass")); err == nil {
		t.Fatal("expected rejection of implausible iteration count")
	}
}

// TestProductionDefaultMatchesOWASPRecommendation is a regression
// guard: a refactor must not silently weaken the at-rest KDF below
// the OWASP 2023 recommendation for PBKDF2-HMAC-SHA256.
func TestProductionDefaultMatchesOWASPRecommendation(t *testing.T) {
	if productionPBKDF2Iterations < 600_000 {
		t.Fatalf("productionPBKDF2Iterations=%d is below OWASP 2023 floor of 600 000",
			productionPBKDF2Iterations)
	}
}

// TestFileProviderEncryptedRoundTrip drives the public FileProvider API
// end-to-end with encryption enabled.
func TestFileProviderEncryptedRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pass := []byte("operator-passphrase-length-ok")

	p1, err := New(Config{Kind: KindFile, StoragePath: dir, Passphrase: pass})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	priv1, kid, err := p1.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}
	if kid != KeyID {
		t.Fatalf("unexpected key id %q", kid)
	}

	encPath := filepath.Join(dir, keyFileEncName)
	blob, err := os.ReadFile(encPath)
	if err != nil {
		t.Fatalf("read encrypted file: %v", err)
	}
	if !hasEncryptedMagic(blob) {
		t.Fatalf("encrypted file does not carry MXKENC01 magic")
	}
	info, err := os.Stat(encPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(dir, keyFileName)); !os.IsNotExist(err) {
		t.Fatalf("plaintext key file must not exist alongside encrypted one, got %v", err)
	}

	// Reload in a fresh provider instance with the same passphrase.
	p2, err := New(Config{Kind: KindFile, StoragePath: dir, Passphrase: pass})
	if err != nil {
		t.Fatalf("New (reload): %v", err)
	}
	priv2, _, err := p2.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("reload LoadOrGenerate: %v", err)
	}
	if !bytes.Equal(priv1, priv2) {
		t.Fatal("encrypted reload produced a different private key")
	}

	// Wrong passphrase on reload must fail.
	p3, err := New(Config{Kind: KindFile, StoragePath: dir, Passphrase: []byte("wrong-passphrase-sure")})
	if err != nil {
		t.Fatalf("New (wrong pass): %v", err)
	}
	if _, _, err := p3.LoadOrGenerate(context.Background()); err == nil {
		t.Fatal("expected load with wrong passphrase to fail")
	}

	// Signing still works after reload.
	sig, err := p2.Sign(context.Background(), []byte("hello"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !ed25519.Verify(p2.PublicKey(), []byte("hello"), sig) {
		t.Fatal("signature verification failed post-reload")
	}
}

// TestFileProviderLegacyUpgradesOnEncryptionEnable documents the
// upgrade path: a plaintext key on disk is transparently re-encrypted
// on first load once a passphrase is configured, and the plaintext
// file is removed.
func TestFileProviderLegacyUpgradesOnEncryptionEnable(t *testing.T) {
	dir := t.TempDir()

	// Step 1: write a plaintext key (old install).
	old, err := New(Config{Kind: KindFile, StoragePath: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	oldPriv, _, err := old.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("legacy LoadOrGenerate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, keyFileName)); err != nil {
		t.Fatalf("plaintext key must exist after legacy load: %v", err)
	}

	// Step 2: enable encryption and reload. The plaintext must be
	// upgraded, not regenerated, so the server identity survives.
	pass := []byte("upgrade-passphrase-ok")
	upg, err := New(Config{Kind: KindFile, StoragePath: dir, Passphrase: pass})
	if err != nil {
		t.Fatalf("New (upgrade): %v", err)
	}
	newPriv, _, err := upg.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("upgrade LoadOrGenerate: %v", err)
	}
	if !bytes.Equal(oldPriv, newPriv) {
		t.Fatal("upgrade must preserve the private key (else server identity rotates silently)")
	}
	if _, err := os.Stat(filepath.Join(dir, keyFileName)); !os.IsNotExist(err) {
		t.Fatalf("plaintext file must be removed after upgrade, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, keyFileEncName)); err != nil {
		t.Fatalf("encrypted file must exist after upgrade: %v", err)
	}
}

// TestFileProviderRefusesToSilentlyDowngrade asserts the safety
// property that a plaintext-mode provider will not read (and thus
// effectively ignore) an encrypted key file that exists on disk.
func TestFileProviderRefusesToSilentlyDowngrade(t *testing.T) {
	dir := t.TempDir()

	p, err := New(Config{Kind: KindFile, StoragePath: dir, Passphrase: []byte("p")})
	if err != nil {
		t.Fatalf("New (encrypted): %v", err)
	}
	if _, _, err := p.LoadOrGenerate(context.Background()); err != nil {
		t.Fatalf("LoadOrGenerate: %v", err)
	}

	plain, err := New(Config{Kind: KindFile, StoragePath: dir})
	if err != nil {
		t.Fatalf("New (plaintext): %v", err)
	}
	if _, _, err := plain.LoadOrGenerate(context.Background()); err == nil {
		t.Fatal("plaintext mode must refuse to bypass an existing encrypted file")
	}
}
