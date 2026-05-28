package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyEnvOverridesTrustedNotaries(t *testing.T) {
	cfg := validConfig()
	setDefaults(cfg)

	t.Setenv("MXKEYS_TRUSTED_NOTARIES", "matrix.org|ed25519:auto|cHVibGljX2tleQ;example.org|ed25519:ex|ZXhhbXBsZQ")
	if err := applyEnvOverrides(cfg); err != nil {
		t.Fatal(err)
	}

	if len(cfg.Security.TrustedNotaries) != 2 {
		t.Fatalf("expected 2 trusted notaries from env, got %d", len(cfg.Security.TrustedNotaries))
	}
	if cfg.Security.TrustedNotaries[1].ServerName != "example.org" {
		t.Fatalf("second trusted notary mismatch")
	}
}

func TestLoadEnvOverridesFileValues(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configData := `server:
  name: from-file.example
  port: 8448
  bind_address: "127.0.0.1"

database:
  url: postgres://test:test@localhost/test

keys:
  storage_path: /tmp/mxkeys-keys
  validity_hours: 12
  cache_ttl_hours: 2
  fetch_timeout_s: 15
  cleanup_hours: 4

security:
  max_server_name_length: 255
  max_servers_per_query: 100
  max_json_depth: 10
  max_signatures_per_key: 10
  require_request_id: true

rate_limit:
  requests_per_second: 10
  burst: 20
  query_per_second: 2
  query_burst: 4
`
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Setenv("MXKEYS_SERVER_NAME", "from-env.example")
	t.Setenv("MXKEYS_RATE_LIMIT_REQUESTS_PER_SECOND", "99.5")
	t.Setenv("MXKEYS_SECURITY_REQUIRE_REQUEST_ID", "false")
	t.Setenv("MXKEYS_SECURITY_TRUST_FORWARDED_HEADERS", "true")
	t.Setenv("MXKEYS_SECURITY_TRUSTED_PROXIES", "10.0.0.0/8,127.0.0.1/32")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Name != "from-env.example" {
		t.Fatalf("expected env override for server.name, got %q", cfg.Server.Name)
	}
	if cfg.RateLimit.RequestsPerSecond != 99.5 {
		t.Fatalf("expected env override for rate limit, got %v", cfg.RateLimit.RequestsPerSecond)
	}
	if cfg.Security.RequireRequestID {
		t.Fatalf("expected env override to set require_request_id=false")
	}
	if !cfg.Security.TrustForwardedHeaders || len(cfg.Security.TrustedProxies) != 2 {
		t.Fatalf("expected env override to set trusted proxy policy")
	}
}

func TestParseTrustedNotariesEnvSkipsInvalidChunks(t *testing.T) {
	got := parseTrustedNotariesEnv("matrix.org|ed25519:auto|cHVibGlj;broken;example.org|ed25519:ex|ZXhhbXBsZQ;|||")
	if len(got) != 2 {
		t.Fatalf("expected 2 valid entries, got %d", len(got))
	}
	if got[0].ServerName != "matrix.org" || got[1].ServerName != "example.org" {
		t.Fatalf("unexpected parsed order/content: %#v", got)
	}
}

func TestParseTrustedNotariesFromYAMLSkipsIncompleteEntries(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configData := `trusted_notaries:
  - server_name: matrix.org
    key_id: ed25519:auto
    public_key: "cHVibGljX2tleQ"
  - server_name: incomplete.example
    key_id: ed25519:missing-public-key
`
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	notaries := parseTrustedNotariesFromYAML(configPath)
	if len(notaries) != 1 {
		t.Fatalf("expected only complete entries to be parsed, got %d", len(notaries))
	}
	if notaries[0].ServerName != "matrix.org" {
		t.Fatalf("unexpected entry parsed: %#v", notaries[0])
	}
}
