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
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
)

func TestNotarySigningKeyBackupRestoreDrill(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mxkeys-backup-drill-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	primaryDir := filepath.Join(tmpDir, "primary")
	backupDir := filepath.Join(tmpDir, "backup")
	restoreDir := filepath.Join(tmpDir, "restore")

	n1 := &Notary{}
	if err := n1.initSigningKey(primaryDir); err != nil {
		t.Fatalf("failed to initialize primary key: %v", err)
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

	n2 := &Notary{}
	if err := n2.initSigningKey(restoreDir); err != nil {
		t.Fatalf("failed to initialize restored key: %v", err)
	}

	pub1 := n1.serverKeyPair.Public().(ed25519.PublicKey)
	pub2 := n2.serverKeyPair.Public().(ed25519.PublicKey)
	if !pub1.Equal(pub2) {
		t.Fatal("backup/restore drill failed: restored signing key does not match original")
	}
}
