package config

import (
	"os"
	"path/filepath"
	"testing"
)

func validConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:        "mxkeys.test",
			Port:        8448,
			BindAddress: "0.0.0.0",
		},
		Database: DatabaseConfig{
			URL:                "postgres://test:test@localhost/test",
			MaxConnections:     10,
			MaxIdleConnections: 2,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Keys: KeysConfig{
			StoragePath:   "/var/lib/mxkeys/keys",
			ValidityHours: 24,
			CacheTTLHours: 1,
			FetchTimeoutS: 30,
			CleanupHours:  6,
		},
		TrustedServers: TrustedServersConfig{
			Fallback: []string{"matrix.org"},
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 50,
			Burst:             100,
			QueryPerSecond:    10,
			QueryBurst:        20,
		},
		Security: SecurityConfig{
			MaxServerNameLength: 255,
			MaxServersPerQuery:  100,
			MaxJSONDepth:        10,
			MaxSignaturesPerKey: 10,
		},
		Transparency: TransparencyConfig{
			Enabled:       false,
			LogAllKeys:    true,
			LogKeyChanges: true,
			LogAnomalies:  true,
			RetentionDays: 365,
			TableName:     "key_transparency_log",
		},
		Cluster: ClusterConfig{
			Enabled:       false,
			BindAddress:   "0.0.0.0",
			BindPort:      7946,
			ConsensusMode: "crdt",
			SyncInterval:  5,
		},
	}
}

func TestValidConfigPasses(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
}

func TestEmptyServerNameReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Name = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty server.name should be rejected")
	}
}

func TestServerPortZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Error("server.port=0 should be rejected")
	}
}

func TestServerPortNegativeReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = -1
	if err := cfg.Validate(); err == nil {
		t.Error("negative server.port should be rejected")
	}
}

func TestServerPortTooHighReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 65536
	if err := cfg.Validate(); err == nil {
		t.Error("server.port > 65535 should be rejected")
	}
}

func TestEmptyBindAddressReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.BindAddress = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty bind_address should be rejected")
	}
}

func TestEmptyDatabaseURLReject(t *testing.T) {
	cfg := validConfig()
	cfg.Database.URL = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty database.url should be rejected")
	}
}

func TestEmptyKeysStoragePathReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.StoragePath = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty keys.storage_path should be rejected")
	}
}

func TestValidityHoursZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.ValidityHours = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.validity_hours=0 should be rejected")
	}
}

func TestValidityHoursNegativeReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.ValidityHours = -1
	if err := cfg.Validate(); err == nil {
		t.Error("negative keys.validity_hours should be rejected")
	}
}

func TestCacheTTLHoursZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.CacheTTLHours = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.cache_ttl_hours=0 should be rejected")
	}
}

func TestFetchTimeoutZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.FetchTimeoutS = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.fetch_timeout_s=0 should be rejected")
	}
}

func TestCleanupHoursZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.CleanupHours = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.cleanup_hours=0 should be rejected")
	}
}

func TestValidPortBoundaries(t *testing.T) {
	tests := []struct {
		port    int
		isValid bool
	}{
		{1, true},
		{80, true},
		{443, true},
		{8448, true},
		{65535, true},
		{0, false},
		{-1, false},
		{65536, false},
		{100000, false},
	}

	for _, tt := range tests {
		cfg := validConfig()
		cfg.Server.Port = tt.port
		err := cfg.Validate()
		if tt.isValid && err != nil {
			t.Errorf("port %d should be valid, got error: %v", tt.port, err)
		}
		if !tt.isValid && err == nil {
			t.Errorf("port %d should be invalid", tt.port)
		}
	}
}

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
  seeds:
    - 127.0.0.1:7947
    - 127.0.0.1:7948
  consensus_mode: raft
  sync_interval: 9

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
	if !cfg.TrustPolicy.Enabled || cfg.TrustPolicy.RequireNotarySignatures != 1 {
		t.Fatalf("trust_policy values not loaded")
	}
	if !cfg.Transparency.Enabled || cfg.Transparency.RetentionDays != 90 {
		t.Fatalf("transparency values not loaded")
	}
	if !cfg.Cluster.Enabled || cfg.Cluster.ConsensusMode != "raft" || len(cfg.Cluster.Seeds) != 2 {
		t.Fatalf("cluster values not loaded")
	}
	if len(cfg.Security.TrustedNotaries) != 1 {
		t.Fatalf("trusted_notaries not parsed from YAML section")
	}
	if cfg.Security.TrustedNotaries[0].ServerName != "matrix.org" {
		t.Fatalf("trusted_notaries server_name mismatch")
	}
}

func TestApplyEnvOverridesTrustedNotaries(t *testing.T) {
	cfg := validConfig()
	setDefaults(cfg)

	t.Setenv("MXKEYS_TRUSTED_NOTARIES", "matrix.org|ed25519:auto|cHVibGljX2tleQ;example.org|ed25519:ex|ZXhhbXBsZQ")
	applyEnvOverrides(cfg)

	if len(cfg.Security.TrustedNotaries) != 2 {
		t.Fatalf("expected 2 trusted notaries from env, got %d", len(cfg.Security.TrustedNotaries))
	}
	if cfg.Security.TrustedNotaries[1].ServerName != "example.org" {
		t.Fatalf("second trusted notary mismatch")
	}
}
