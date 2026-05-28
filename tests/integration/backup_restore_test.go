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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stubDBBinaries drops fake pg_dump and psql onto binDir. The stubs
// understand the single `--file=` flag mxkeys-backup.sh and the
// -f / < STDIN redirection patterns mxkeys-restore.sh use. The fake
// dump is a trivial `SELECT 1;` text file; restore writes it back
// under a caller-supplied sentinel name so the round-trip test can
// confirm "the exact bytes travelled from backup to restore".
func stubDBBinaries(t *testing.T, binDir, sentinelDir string) {
	t.Helper()
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	pgDump := `#!/usr/bin/env bash
for a in "$@"; do
  case "$a" in --file=*) out=${a#--file=} ;; esac
done
printf 'SELECT 1;\n' > "$out"
`
	if err := os.WriteFile(filepath.Join(binDir, "pg_dump"), []byte(pgDump), 0o755); err != nil {
		t.Fatalf("write pg_dump stub: %v", err)
	}
	// psql stub: reads SQL from stdin and writes it as-is to a sentinel
	// file in sentinelDir so the round-trip test can verify the data
	// crossed the backup tarball unchanged.
	psql := fmt.Sprintf(`#!/usr/bin/env bash
cat > %q/restored.sql
`, sentinelDir)
	if err := os.WriteFile(filepath.Join(binDir, "psql"), []byte(psql), 0o755); err != nil {
		t.Fatalf("write psql stub: %v", err)
	}
}

// TestBackupScriptProducesExpectedLayout exercises scripts/mxkeys-backup.sh
// against a synthetic filesystem layout and verifies the output tarball
// contains every mandatory path.
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

	binDir := filepath.Join(root, "bin")
	sentinelDir := filepath.Join(root, "sentinel")
	if err := os.MkdirAll(sentinelDir, 0o755); err != nil {
		t.Fatalf("mkdir sentinel: %v", err)
	}
	stubDBBinaries(t, binDir, sentinelDir)

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

// TestBackupRestoreRoundTrip is the end-to-end proof of the operator
// contract: seed a filesystem layout, back it up, wipe the layout,
// restore the tarball, and verify every file comes back with the
// exact byte contents the backup captured. The SQL portion travels
// through a stubbed psql-reads-stdin path so the test does not need
// a live PostgreSQL.
func TestBackupRestoreRoundTrip(t *testing.T) {
	scriptDir, err := findRepoRoot(t)
	if err != nil {
		t.Skipf("cannot locate repo root: %v", err)
	}
	backupSh := filepath.Join(scriptDir, "scripts", "mxkeys-backup.sh")
	restoreSh := filepath.Join(scriptDir, "scripts", "mxkeys-restore.sh")
	for _, p := range []string{backupSh, restoreSh} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("script missing: %v", err)
		}
	}

	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	sentinelDir := filepath.Join(root, "sentinel")
	if err := os.MkdirAll(sentinelDir, 0o755); err != nil {
		t.Fatalf("mkdir sentinel: %v", err)
	}
	stubDBBinaries(t, binDir, sentinelDir)

	// Seed filesystem state that the backup tarball must carry through.
	keysDir := filepath.Join(root, "keys")
	raftDir := filepath.Join(root, "raft")
	for _, d := range []string{keysDir, raftDir} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	keyBytes := []byte("fake-seed-bytes-for-roundtrip-test")
	walBytes := []byte("fake-wal-bytes\x00\x01\x02")
	if err := os.WriteFile(filepath.Join(keysDir, "mxkeys_ed25519.key"), keyBytes, 0o600); err != nil {
		t.Fatalf("seed key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(raftDir, "raft.wal"), walBytes, 0o600); err != nil {
		t.Fatalf("seed wal: %v", err)
	}

	outputDir := filepath.Join(root, "backups")

	// 1. Backup.
	backupCmd := exec.Command(backupSh,
		"--database-url", "postgres://stub:stub@127.0.0.1:1/stub",
		"--keys-dir", keysDir,
		"--raft-dir", raftDir,
		"--output-dir", outputDir,
	)
	backupCmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	if out, err := backupCmd.CombinedOutput(); err != nil {
		t.Fatalf("backup: %v\n%s", err, string(out))
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected one tarball, got %v (err=%v)", entries, err)
	}
	tarball := filepath.Join(outputDir, entries[0].Name())

	// 2. Wipe the source dirs to prove the restore actually populates.
	if err := os.RemoveAll(keysDir); err != nil {
		t.Fatalf("wipe keys: %v", err)
	}
	if err := os.RemoveAll(raftDir); err != nil {
		t.Fatalf("wipe raft: %v", err)
	}

	// 3. Restore.
	restoreCmd := exec.Command(restoreSh,
		"--input", tarball,
		"--database-url", "postgres://stub:stub@127.0.0.1:1/stub",
		"--keys-dir", keysDir,
		"--raft-dir", raftDir,
	)
	restoreCmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	if out, err := restoreCmd.CombinedOutput(); err != nil {
		t.Fatalf("restore: %v\n%s", err, string(out))
	}

	// 4. Verify byte identity of every restored path.
	type checkCase struct {
		path string
		want []byte
	}
	checks := []checkCase{
		{filepath.Join(keysDir, "mxkeys_ed25519.key"), keyBytes},
		{filepath.Join(raftDir, "raft.wal"), walBytes},
		{filepath.Join(sentinelDir, "restored.sql"), []byte("SELECT 1;\n")},
	}
	for _, c := range checks {
		got, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatalf("restored file missing: %s: %v", c.path, err)
		}
		if string(got) != string(c.want) {
			t.Fatalf("round-trip mismatch at %s\n got:  %q\n want: %q", c.path, got, c.want)
		}
	}

	// 5. Permissions must be preserved: keys dir 0700, key file 0600.
	if info, err := os.Stat(keysDir); err != nil {
		t.Fatalf("stat restored keys dir: %v", err)
	} else if info.Mode().Perm() != 0o700 {
		t.Fatalf("restored keys dir perm = %o, want 0700", info.Mode().Perm())
	}
	if info, err := os.Stat(filepath.Join(keysDir, "mxkeys_ed25519.key")); err != nil {
		t.Fatalf("stat restored key: %v", err)
	} else if info.Mode().Perm() != 0o600 {
		t.Fatalf("restored key perm = %o, want 0600", info.Mode().Perm())
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
