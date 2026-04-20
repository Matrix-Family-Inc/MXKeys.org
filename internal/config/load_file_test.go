package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExtendedSectionsFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configData := `server:
  name: mxkeys.test
  port: 8448
  bind_address: "127.0.0.1"

database:
  url: postgres://test:test@localhost/test
  max_connections: 15
  max_idle_connections: 3

logging:
  level: debug
  format: json

keys:
  storage_path: /tmp/mxkeys-keys
  validity_hours: 12
  cache_ttl_hours: 2
  fetch_timeout_s: 15
  cleanup_hours: 4

trusted_servers:
  fallback:
    - matrix.org
    - example.org

rate_limit:
  requests_per_second: 75.5
  burst: 150
  query_per_second: 12.5
  query_burst: 30

security:
  max_server_name_length: 200
  max_servers_per_query: 40
  max_json_depth: 8
  max_signatures_per_key: 6
  require_request_id: true
  trust_forwarded_headers: true
  enterprise_access_token: enterprise-token
  trusted_proxies:
    - 127.0.0.1/32

trust_policy:
  enabled: true
  deny_list:
    - "*.invalid"
  allow_list:
    - "matrix.org"
  require_notary_signatures: 1
  max_key_age_hours: 48
  require_well_known: true
  require_valid_tls: true
  block_private_ips: true

transparency:
  enabled: true
  log_all_keys: true
  log_key_changes: true
  log_anomalies: true
  retention_days: 90
  table_name: key_transparency_log

cluster:
  enabled: true
  node_id: node-a
  bind_address: 127.0.0.1
  bind_port: 7946
  advertise_address: 10.0.0.10
  advertise_port: 7946
  seeds:
    - 127.0.0.1:7947
    - 127.0.0.1:7948
  consensus_mode: raft
  sync_interval: 9
  shared_secret: cluster-secret

trusted_notaries:
  - server_name: matrix.org
    key_id: ed25519:auto
    public_key: "cHVibGljX2tleQ"
`
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.RateLimit.RequestsPerSecond != 75.5 {
		t.Fatalf("unexpected rate_limit.requests_per_second: %v", cfg.RateLimit.RequestsPerSecond)
	}
	if !cfg.Security.RequireRequestID {
		t.Fatalf("security.require_request_id should be true")
	}
	if !cfg.Security.TrustForwardedHeaders || len(cfg.Security.TrustedProxies) != 1 || cfg.Security.TrustedProxies[0] != "127.0.0.1/32" {
		t.Fatalf("security trusted proxy settings not loaded")
	}
	if cfg.Security.EnterpriseAccessToken != "enterprise-token" {
		t.Fatalf("security.enterprise_access_token not loaded")
	}
	if !cfg.TrustPolicy.Enabled || cfg.TrustPolicy.RequireNotarySignatures != 1 {
		t.Fatalf("trust_policy values not loaded")
	}
	if !cfg.Transparency.Enabled || cfg.Transparency.RetentionDays != 90 {
		t.Fatalf("transparency values not loaded")
	}
	if !cfg.Cluster.Enabled || cfg.Cluster.ConsensusMode != "raft" || len(cfg.Cluster.Seeds) != 2 {
		t.Fatalf("cluster values not loaded")
	}
	if cfg.Cluster.AdvertiseAddress != "10.0.0.10" || cfg.Cluster.SharedSecret != "cluster-secret" {
		t.Fatalf("extended cluster values not loaded")
	}
	if len(cfg.Security.TrustedNotaries) != 1 {
		t.Fatalf("trusted_notaries not parsed from YAML section")
	}
	if cfg.Security.TrustedNotaries[0].ServerName != "matrix.org" {
		t.Fatalf("trusted_notaries server_name mismatch")
	}
}
