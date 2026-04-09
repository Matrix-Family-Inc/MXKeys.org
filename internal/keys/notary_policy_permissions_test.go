/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNotaryApplyTrustPolicyToRequest(t *testing.T) {
	n := &Notary{
		trustPolicy: NewTrustPolicy(TrustPolicyConfig{
			Enabled:  true,
			DenyList: []string{"denied.example"},
		}),
	}

	req := &KeyQueryRequest{
		ServerKeys: map[string]map[string]KeyCriteria{
			"allowed.example": {},
			"denied.example":  {},
		},
	}

	failures := n.applyTrustPolicyToRequest(req)
	if len(failures) != 1 {
		t.Fatalf("expected 1 policy failure, got %d", len(failures))
	}
	if _, ok := req.ServerKeys["denied.example"]; ok {
		t.Fatalf("denied server should be removed from request")
	}
	if _, ok := req.ServerKeys["allowed.example"]; !ok {
		t.Fatalf("allowed server should remain in request")
	}
}

func TestNotaryCheckResponsePolicy(t *testing.T) {
	n := &Notary{
		trustPolicy: NewTrustPolicy(TrustPolicyConfig{
			Enabled:                 true,
			RequireNotarySignatures: 1,
		}),
	}

	resp := &ServerKeysResponse{
		ServerName: "policy.example",
		Signatures: map[string]map[string]string{
			"policy.example": {"ed25519:key": "sig"},
		},
	}

	violation := n.checkResponsePolicy("policy.example", resp)
	if violation == nil || violation.Rule != "require_notary_signatures" {
		t.Fatalf("expected require_notary_signatures violation, got %#v", violation)
	}
}

func TestSortedServerNames(t *testing.T) {
	serverKeys := map[string]map[string]KeyCriteria{
		"z.example.org": {},
		"a.example.org": {},
		"m.example.org": {},
	}

	names := sortedServerNames(serverKeys)
	expected := []string{"a.example.org", "m.example.org", "z.example.org"}
	if len(names) != len(expected) {
		t.Fatalf("unexpected sorted length: %d", len(names))
	}
	for i := range expected {
		if names[i] != expected[i] {
			t.Fatalf("unexpected order at %d: got %s want %s", i, names[i], expected[i])
		}
	}
}

func TestInitSigningKeyEnforcesSecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not portable on windows")
	}

	tmpDir, err := os.MkdirTemp("", "mxkeys-key-perm-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := &Notary{}
	if err := n.initSigningKey(tmpDir); err != nil {
		t.Fatalf("initSigningKey failed: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "mxkeys_ed25519.key")

	dirInfo, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat key dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("key directory permissions must be 0700, got %04o", dirInfo.Mode().Perm())
	}

	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("key file permissions must be 0600, got %04o", keyInfo.Mode().Perm())
	}
}

func TestInitSigningKeyTightensExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not portable on windows")
	}

	tmpDir, err := os.MkdirTemp("", "mxkeys-key-tighten-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	n := &Notary{}
	if err := n.initSigningKey(tmpDir); err != nil {
		t.Fatalf("initSigningKey failed: %v", err)
	}

	keyPath := filepath.Join(tmpDir, "mxkeys_ed25519.key")

	if err := os.Chmod(tmpDir, 0o755); err != nil {
		t.Fatalf("failed to relax dir perms: %v", err)
	}
	if err := os.Chmod(keyPath, 0o644); err != nil {
		t.Fatalf("failed to relax file perms: %v", err)
	}

	if err := n.initSigningKey(tmpDir); err != nil {
		t.Fatalf("initSigningKey second run failed: %v", err)
	}

	dirInfo, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("failed to stat key dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("key directory permissions must be tightened to 0700, got %04o", dirInfo.Mode().Perm())
	}

	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("key file permissions must be tightened to 0600, got %04o", keyInfo.Mode().Perm())
	}
}
