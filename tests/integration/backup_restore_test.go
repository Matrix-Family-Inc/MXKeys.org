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
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackupScriptProducesExpectedLayout exercises scripts/mxkeys-backup.sh
// against a synthetic filesystem layout and verifies the output tarball
// contains every mandatory path. Postgres is not required: the test
// uses --skip-database fallback by passing an empty DB URL and
// patching the script invocation to bypass pg_dump when the URL
// equals "fake". Instead of modifying the script, we shell-stub
// pg_dump via a PATH override so the script genuinely runs the
// documented command path.
func TestBackupScriptProducesExpectedLayout(t *testing.T) {
	scriptDir, err := findRepoRoot(t)
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	backupSh := filepath.Join(scriptDir, "scripts", "mxkeys-backup.sh")
	if _, err := os.Stat(backupSh); err != nil {
		t.Skipf("backup script missing: %v", err)
	}

	root := t.TempDir()

	// Stub a fake pg_dump on PATH that writes a deterministic file.
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	fakeDump := filepath.Join(binDir, "pg_dump")
	if err := os.WriteFile(fakeDump, []byte("#!/usr/bin/env bash\nshift 0\nfor a in \"$@\"; do\n  case \"$a\" in --file=*) out=${a#--file=} ;; esac\ndone\necho 'SELECT 1;' > \"$out\"\n"), 0o755); err != nil {
		t.Fatalf("write fake pg_dump: %v", err)
	}

	keysDir := filepath.Join(root, "keys")
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		t.Fatalf("mkdir keys: %v", err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, "mxkeys_ed25519.key"), []byte("fake-key-bytes"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	raftDir := filepath.Join(root, "raft")
	if err := os.MkdirAll(raftDir, 0o700); err != nil {
		t.Fatalf("mkdir raft: %v", err)
	}
	if err := os.WriteFile(filepath.Join(raftDir, "raft.wal"), []byte("fake-wal"), 0o600); err != nil {
		t.Fatalf("write wal: %v", err)
	}

	outputDir := filepath.Join(root, "backups")

	cmd := exec.Command(backupSh,
		"--database-url", "postgres://stub:stub@127.0.0.1:1/stub",
		"--keys-dir", keysDir,
		"--raft-dir", raftDir,
		"--output-dir", outputDir,
	)
	// Prepend our stub bin dir so the script picks up fake pg_dump.
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("backup script failed: %v\noutput:\n%s", err, string(out))
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected exactly one tarball in %s, got %v (err=%v)", outputDir, entries, err)
	}
	tarball := filepath.Join(outputDir, entries[0].Name())
	if !strings.HasSuffix(tarball, ".tar.gz") {
		t.Fatalf("unexpected backup name: %s", tarball)
	}
	info, err := os.Stat(tarball)
	if err != nil {
		t.Fatalf("stat tarball: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 perms on tarball, got %o", info.Mode().Perm())
	}

	got := readTarballEntries(t, tarball)
	// The archive has a top-level dir like mxkeys-backup-<TS>/.
	want := []string{
		"database.sql",
		"keys/mxkeys_ed25519.key",
		"raft/raft.wal",
		"MANIFEST.txt",
	}
	for _, needle := range want {
		if !hasSuffixAny(got, "/"+needle) {
			t.Errorf("tarball missing expected entry %q\nactual entries:\n%s", needle, strings.Join(got, "\n"))
		}
	}
}

func hasSuffixAny(all []string, suffix string) bool {
	for _, s := range all {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func readTarballEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open tarball: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var entries []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		entries = append(entries, h.Name)
	}
	return entries
}

// findRepoRoot walks upward from CWD until it finds a directory that
// contains a go.mod file with the "module mxkeys" declaration. The
// integration tests are allowed to run from the repo root or from
// nested ./tests/integration; this resolves both.
func findRepoRoot(t *testing.T) (string, error) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		gomod := filepath.Join(dir, "go.mod")
		if b, err := os.ReadFile(gomod); err == nil && strings.Contains(string(b), "module mxkeys") {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
