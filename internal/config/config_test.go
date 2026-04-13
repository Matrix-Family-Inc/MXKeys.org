package config

import (
	"os"
	"path/filepath"
	"strings"
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
			MaxServerNameLength:   255,
			MaxServersPerQuery:    100,
			MaxJSONDepth:          10,
			MaxSignaturesPerKey:   10,
			TrustForwardedHeaders: false,
			TrustedProxies:        []string{"127.0.0.1/32"},
			EnterpriseAccessToken: "enterprise-token",
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
			Enabled:          false,
			BindAddress:      "127.0.0.1",
			BindPort:         7946,
			AdvertiseAddress: "127.0.0.1",
			AdvertisePort:    7946,
			ConsensusMode:    "crdt",
			SyncInterval:     5,
			SharedSecret:     "cluster-secret",
		},
	}
}

func TestValidConfigPasses(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config should pass: %v", err)
	}
}

func TestValidateRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*Config)
		errMatch string
	}{
		{
			name:     "server name required",
			mutate:   func(c *Config) { c.Server.Name = "" },
			errMatch: "server.name is required",
		},
		{
			name:     "server port range",
			mutate:   func(c *Config) { c.Server.Port = 65536 },
			errMatch: "server.port must be between 1 and 65535",
		},
		{
			name:     "bind address required",
			mutate:   func(c *Config) { c.Server.BindAddress = "" },
			errMatch: "server.bind_address is required",
		},
		{
			name:     "database url required",
			mutate:   func(c *Config) { c.Database.URL = "" },
			errMatch: "database.url is required",
		},
		{
			name:     "keys storage path required",
			mutate:   func(c *Config) { c.Keys.StoragePath = "" },
			errMatch: "keys.storage_path is required",
		},
		{
			name:     "keys validity positive",
			mutate:   func(c *Config) { c.Keys.ValidityHours = 0 },
			errMatch: "keys.validity_hours must be positive",
		},
		{
			name:     "keys cache ttl positive",
			mutate:   func(c *Config) { c.Keys.CacheTTLHours = 0 },
			errMatch: "keys.cache_ttl_hours must be positive",
		},
		{
			name:     "keys fetch timeout positive",
			mutate:   func(c *Config) { c.Keys.FetchTimeoutS = 0 },
			errMatch: "keys.fetch_timeout_s must be positive",
		},
		{
			name:     "keys cleanup positive",
			mutate:   func(c *Config) { c.Keys.CleanupHours = 0 },
			errMatch: "keys.cleanup_hours must be positive",
		},
		{
			name:     "rate limit requests positive",
			mutate:   func(c *Config) { c.RateLimit.RequestsPerSecond = 0 },
			errMatch: "rate_limit.requests_per_second must be positive",
		},
		{
			name:     "rate limit burst positive",
			mutate:   func(c *Config) { c.RateLimit.Burst = 0 },
			errMatch: "rate_limit.burst must be positive",
		},
		{
			name:     "security max server name positive",
			mutate:   func(c *Config) { c.Security.MaxServerNameLength = 0 },
			errMatch: "security.max_server_name_length must be positive",
		},
		{
			name:     "security max signatures positive",
			mutate:   func(c *Config) { c.Security.MaxSignaturesPerKey = 0 },
			errMatch: "security.max_signatures_per_key must be positive",
		},
		{
			name: "security trusted proxies required when forwarded headers enabled",
			mutate: func(c *Config) {
				c.Security.TrustForwardedHeaders = true
				c.Security.TrustedProxies = nil
			},
			errMatch: "security.trusted_proxies is required when security.trust_forwarded_headers=true",
		},
		{
			name: "security trusted proxies must be valid cidr or ip",
			mutate: func(c *Config) {
				c.Security.TrustedProxies = []string{"not-a-cidr"}
			},
			errMatch: "security.trusted_proxies contains invalid CIDR or IP",
		},
		{
			name:     "trust policy notary signatures non negative",
			mutate:   func(c *Config) { c.TrustPolicy.RequireNotarySignatures = -1 },
			errMatch: "trust_policy.require_notary_signatures must be non-negative",
		},
		{
			name:     "trust policy max key age non negative",
			mutate:   func(c *Config) { c.TrustPolicy.MaxKeyAgeHours = -1 },
			errMatch: "trust_policy.max_key_age_hours must be non-negative",
		},
		{
			name:     "transparency retention positive",
			mutate:   func(c *Config) { c.Transparency.RetentionDays = 0 },
			errMatch: "transparency.retention_days must be positive",
		},
		{
			name:     "transparency table required",
			mutate:   func(c *Config) { c.Transparency.TableName = "" },
			errMatch: "transparency.table_name is required",
		},
		{
			name:     "transparency table safe identifier",
			mutate:   func(c *Config) { c.Transparency.TableName = "key-log;drop" },
			errMatch: "transparency.table_name must be a safe SQL identifier",
		},
		{
			name: "cluster enabled requires bind address",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.BindAddress = ""
			},
			errMatch: "cluster.bind_address is required when cluster.enabled=true",
		},
		{
			name: "cluster enabled requires valid port",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.BindPort = 0
			},
			errMatch: "cluster.bind_port must be between 1 and 65535 when cluster.enabled=true",
		},
		{
			name: "cluster enabled requires positive sync interval",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.SyncInterval = 0
			},
			errMatch: "cluster.sync_interval must be positive when cluster.enabled=true",
		},
		{
			name: "cluster enabled requires shared secret",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.SharedSecret = ""
			},
			errMatch: "cluster.shared_secret is required when cluster.enabled=true",
		},
		{
			name: "cluster enabled allows only supported consensus",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.ConsensusMode = "custom"
			},
			errMatch: "cluster.consensus_mode must be 'crdt' or 'raft'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected validation error containing %q", tt.errMatch)
			}
			if !strings.Contains(err.Error(), tt.errMatch) {
				t.Fatalf("unexpected error %q, expected to contain %q", err.Error(), tt.errMatch)
			}
		})
	}
}

func TestValidateAllowsIncompleteClusterWhenDisabled(t *testing.T) {
	cfg := validConfig()
	cfg.Cluster.Enabled = false
	cfg.Cluster.BindAddress = ""
	cfg.Cluster.BindPort = 0
	cfg.Cluster.AdvertiseAddress = ""
	cfg.Cluster.AdvertisePort = 0
	cfg.Cluster.SyncInterval = 0
	cfg.Cluster.ConsensusMode = ""
	cfg.Cluster.SharedSecret = ""

	if err := cfg.Validate(); err != nil {
		t.Fatalf("cluster fields should not be required when disabled: %v", err)
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
