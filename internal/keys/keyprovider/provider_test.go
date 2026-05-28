package keyprovider

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestFileProviderGeneratesAndReloads(t *testing.T) {
	dir := t.TempDir()

	p1, err := New(Config{Kind: KindFile, StoragePath: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	priv1, kid1, err := p1.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("first LoadOrGenerate: %v", err)
	}
	if kid1 != KeyID {
		t.Fatalf("expected key id %s, got %s", KeyID, kid1)
	}

	info, err := os.Stat(filepath.Join(dir, keyFileName))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 file perms, got %o", info.Mode().Perm())
	}

	p2, err := New(Config{Kind: KindFile, StoragePath: dir})
	if err != nil {
		t.Fatalf("New (reload): %v", err)
	}
	priv2, _, err := p2.LoadOrGenerate(context.Background())
	if err != nil {
		t.Fatalf("second LoadOrGenerate: %v", err)
	}
	if !equalBytes(priv1, priv2) {
		t.Fatal("reloaded key differs from original")
	}

	sig, err := p1.Sign(context.Background(), []byte("hello"))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !ed25519.Verify(p1.PublicKey(), []byte("hello"), sig) {
		t.Fatal("signature does not verify against own public key")
	}
}

func TestEnvProviderDecodesSeedAndFullKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	t.Run("full key base64", func(t *testing.T) {
		enc := base64.RawStdEncoding.EncodeToString(priv)
		t.Setenv("MXKEYS_TEST_FULL", enc)

		p, err := New(Config{Kind: KindEnv, EnvVar: "MXKEYS_TEST_FULL"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		got, _, err := p.LoadOrGenerate(context.Background())
		if err != nil {
			t.Fatalf("LoadOrGenerate: %v", err)
		}
		if !equalBytes(got, priv) {
			t.Fatal("env full key round-trip mismatch")
		}
		if !p.PublicKey().Equal(pub) {
			t.Fatal("env public key mismatch")
		}
	})

	t.Run("seed base64", func(t *testing.T) {
		seed := priv.Seed()
		enc := base64.RawStdEncoding.EncodeToString(seed)
		t.Setenv("MXKEYS_TEST_SEED", enc)

		p, err := New(Config{Kind: KindEnv, EnvVar: "MXKEYS_TEST_SEED"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		got, _, err := p.LoadOrGenerate(context.Background())
		if err != nil {
			t.Fatalf("LoadOrGenerate: %v", err)
		}
		if !equalBytes(got, priv) {
			t.Fatal("env seed round-trip mismatch")
		}
	})

	t.Run("missing env", func(t *testing.T) {
		p, err := New(Config{Kind: KindEnv, EnvVar: "MXKEYS_TEST_MISSING"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if _, _, err := p.LoadOrGenerate(context.Background()); err == nil {
			t.Fatal("expected error for unset env var")
		}
	})
}

func TestKMSStubIsNotImplemented(t *testing.T) {
	p, err := New(Config{Kind: KindKMS, KMSEndpoint: "https://kms.example"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, _, err := p.LoadOrGenerate(context.Background()); err != ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestUnknownKindRejected(t *testing.T) {
	if _, err := New(Config{Kind: "bogus"}); err == nil {
		t.Fatal("expected error for unknown kind")
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
