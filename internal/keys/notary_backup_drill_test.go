/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Mar 16 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mxkeys/internal/keys/keyprovider"
)

// TestNotarySigningKeyBackupRestoreDrill exercises the operator runbook for
// backing up and restoring the notary signing key: generate in primary dir,
// copy the on-disk file to a backup dir, restore into a fresh dir, reload via
// the file provider, and assert the public key is bit-identical to the
// original.
func TestNotarySigningKeyBackupRestoreDrill(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mxkeys-backup-drill-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	primaryDir := filepath.Join(tmpDir, "primary")
	backupDir := filepath.Join(tmpDir, "backup")
	restoreDir := filepath.Join(tmpDir, "restore")

	p1, err := keyprovider.New(keyprovider.Config{Kind: keyprovider.KindFile, StoragePath: primaryDir})
	if err != nil {
		t.Fatalf("primary provider: %v", err)
	}
	if _, _, err := p1.LoadOrGenerate(context.Background()); err != nil {
		t.Fatalf("primary LoadOrGenerate: %v", err)
	}

	primaryPath := filepath.Join(primaryDir, "mxkeys_ed25519.key")
	backupPath := filepath.Join(backupDir, "mxkeys_ed25519.key")
	restorePath := filepath.Join(restoreDir, "mxkeys_ed25519.key")

	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	keyBytes, err := os.ReadFile(primaryPath)
	if err != nil {
		t.Fatalf("failed to read primary key: %v", err)
	}
	if err := os.WriteFile(backupPath, keyBytes, 0o600); err != nil {
		t.Fatalf("failed to write backup key: %v", err)
	}

	if err := os.Remove(primaryPath); err != nil {
		t.Fatalf("failed to remove primary key for restore simulation: %v", err)
	}

	if err := os.MkdirAll(restoreDir, 0o700); err != nil {
		t.Fatalf("failed to create restore dir: %v", err)
	}
	backupBytes, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup key: %v", err)
	}
	if err := os.WriteFile(restorePath, backupBytes, 0o600); err != nil {
		t.Fatalf("failed to restore key from backup: %v", err)
	}

	p2, err := keyprovider.New(keyprovider.Config{Kind: keyprovider.KindFile, StoragePath: restoreDir})
	if err != nil {
		t.Fatalf("restore provider: %v", err)
	}
	if _, _, err := p2.LoadOrGenerate(context.Background()); err != nil {
		t.Fatalf("restore LoadOrGenerate: %v", err)
	}

	if !p1.PublicKey().Equal(p2.PublicKey()) {
		t.Fatal("backup/restore drill failed: restored signing key does not match original")
	}
}
