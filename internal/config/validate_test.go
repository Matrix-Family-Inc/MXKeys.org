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
