package config

import (
	"strings"
	"testing"
)

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
			name: "cluster shared secret rejects placeholder",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.SharedSecret = "replace-with-long-random-secret"
			},
			errMatch: "cluster.shared_secret contains a placeholder value",
		},
		{
			name: "cluster shared secret enforces minimum length",
			mutate: func(c *Config) {
				c.Cluster.Enabled = true
				c.Cluster.SharedSecret = "too-short"
			},
			errMatch: "cluster.shared_secret must be at least 32 characters",
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

func TestValidateTrustedNotaries(t *testing.T) {
	// A valid ed25519 public key (32 bytes) in raw base64. Any deterministic
	// byte stream of length 32 is a valid placeholder for this test since
	// validation only checks length and shape, not curve-point validity.
	validKey := "Nzxs2Mh0Fb+Uhv3uTE47iWBoCGY8oSa11BZX9S7W6RE"

	tests := []struct {
		name     string
		notaries []TrustedNotary
		errMatch string
	}{
		{
			name:     "placeholder rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: "base64-encoded-public-key"}},
			errMatch: "placeholder",
		},
		{
			name:     "empty server_name rejected",
			notaries: []TrustedNotary{{ServerName: "", KeyID: "ed25519:auto", PublicKey: validKey}},
			errMatch: "server_name is required",
		},
		{
			name:     "empty key_id rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "", PublicKey: validKey}},
			errMatch: "key_id is required",
		},
		{
			name:     "empty public_key rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: ""}},
			errMatch: "public_key is required",
		},
		{
			name:     "non-base64 public_key rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: "not base64!!"}},
			errMatch: "invalid base64",
		},
		{
			name:     "wrong-length public_key rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: "YWJj"}}, // base64("abc"), 3 bytes
			errMatch: "has length",
		},
		{
			name:     "valid entry accepted",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: validKey}},
		},
		{
			name:     "empty list accepted",
			notaries: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Security.TrustedNotaries = tt.notaries
			err := cfg.Validate()
			if tt.errMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errMatch)
			}
			if !strings.Contains(err.Error(), tt.errMatch) {
				t.Fatalf("unexpected error %q, want substring %q", err.Error(), tt.errMatch)
			}
		})
	}
}

func TestValidateClusterTLS(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Config)
		errMatch string
	}{
		{
			name:     "disabled TLS always valid",
			mutate:   func(c *Config) { c.Cluster.TLS.Enabled = false },
			errMatch: "",
		},
		{
			name: "missing cert_file rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = ""
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
			},
			errMatch: "cert_file is required",
		},
		{
			name: "missing key_file rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = ""
				c.Cluster.TLS.CAFile = "/ca"
			},
			errMatch: "key_file is required",
		},
		{
			name: "missing ca_file rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = ""
			},
			errMatch: "ca_file is required",
		},
		{
			name: "invalid min_version rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
				c.Cluster.TLS.MinVersion = "1.0"
			},
			errMatch: "min_version must be",
		},
		{
			name: "min_version 1.2 rejected (cluster is TLS 1.3 only)",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
				c.Cluster.TLS.MinVersion = "1.2"
			},
			errMatch: "does not support TLS 1.2",
		},
		{
			name: "min_version 1.3 accepted",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
				c.Cluster.TLS.MinVersion = "1.3"
			},
			errMatch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			// Enable cluster so the TLS branch is exercised.
			cfg.Cluster.Enabled = true
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.errMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errMatch)
			}
			if !strings.Contains(err.Error(), tt.errMatch) {
				t.Fatalf("unexpected error %q, want substring %q", err.Error(), tt.errMatch)
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
