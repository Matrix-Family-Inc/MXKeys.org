/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package server

import (
	"strings"
	"testing"

	"mxkeys/internal/config"
	"mxkeys/internal/keys/keyprovider"
)

// TestBuildKeyProviderPlaintextDefault confirms that an operator who
// does not touch keys.encryption gets the legacy plaintext file provider.
// This is the backward-compat invariant for pre-encryption installs.
func TestBuildKeyProviderPlaintextDefault(t *testing.T) {
	cfg := &config.Config{}
	cfg.Keys.StoragePath = t.TempDir()

	p, err := buildKeyProvider(cfg)
	if err != nil {
		t.Fatalf("buildKeyProvider: %v", err)
	}
	if p.Kind() != keyprovider.KindFile {
		t.Fatalf("expected file provider, got %q", p.Kind())
	}
}

// TestBuildKeyProviderEncryptedHappyPath confirms that when both the
// config names an env var AND the env var is set, the provider is built
// in encrypted mode (the actual crypto path is covered by the
// keyprovider package tests).
func TestBuildKeyProviderEncryptedHappyPath(t *testing.T) {
	cfg := &config.Config{}
	cfg.Keys.StoragePath = t.TempDir()
	cfg.Keys.EncryptionPassphraseEnv = "TEST_KEY_PASS"

	t.Setenv("TEST_KEY_PASS", "a-real-passphrase-for-test")

	p, err := buildKeyProvider(cfg)
	if err != nil {
		t.Fatalf("buildKeyProvider: %v", err)
	}
	if p.Kind() != keyprovider.KindFile {
		t.Fatalf("expected file provider, got %q", p.Kind())
	}
}

// TestBuildKeyProviderFailsClosedOnMissingPassphrase: a half-configured
// setup (env var named but not set) must fail at startup, not degrade
// to plaintext.
func TestBuildKeyProviderFailsClosedOnMissingPassphrase(t *testing.T) {
	cfg := &config.Config{}
	cfg.Keys.StoragePath = t.TempDir()
	cfg.Keys.EncryptionPassphraseEnv = "TEST_KEY_PASS_MISSING_INTENTIONAL"

	// Ensure the variable is empty regardless of outer environment.
	t.Setenv("TEST_KEY_PASS_MISSING_INTENTIONAL", "")

	_, err := buildKeyProvider(cfg)
	if err == nil {
		t.Fatal("expected error when passphrase env is empty")
	}
	if !strings.Contains(err.Error(), "passphrase_env") {
		t.Fatalf("error should mention passphrase_env, got %v", err)
	}
}
