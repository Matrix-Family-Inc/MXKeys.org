/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

package config

import (
	"fmt"
	"os"

	zeroconfig "mxkeys/internal/zero/config"
)

// Config server configuration
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Logging        LoggingConfig
	Keys           KeysConfig
	TrustedServers TrustedServersConfig
	RateLimit      RateLimitConfig
	Security       SecurityConfig
	TrustPolicy    TrustPolicyConfig
	Transparency   TransparencyConfig
	Cluster        ClusterConfig
}

// ServerConfig server settings
type ServerConfig struct {
	Name        string
	Port        int
	BindAddress string
}

// DatabaseConfig database settings
type DatabaseConfig struct {
	URL                string
	MaxConnections     int
	MaxIdleConnections int
}

// LoggingConfig logging settings
type LoggingConfig struct {
	Level  string
	Format string
}

// KeysConfig key management settings
type KeysConfig struct {
	StoragePath   string
	ValidityHours int
	CacheTTLHours int
	FetchTimeoutS int
	CleanupHours  int
}

// TrustedServersConfig upstream notary servers
type TrustedServersConfig struct {
	Fallback []string
}

// TrustedNotary pinned notary key
type TrustedNotary struct {
	ServerName string
	KeyID      string
	PublicKey  string
}

// RateLimitConfig rate limiting settings
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
	QueryPerSecond    float64
	QueryBurst        int
}

// SecurityConfig security settings
type SecurityConfig struct {
	MaxServerNameLength   int
	MaxServersPerQuery    int
	MaxJSONDepth          int
	MaxSignaturesPerKey   int
	RequireRequestID      bool
	TrustForwardedHeaders bool
	TrustedProxies        []string
	EnterpriseAccessToken string
	TrustedNotaries       []TrustedNotary
}

// TrustPolicyConfig federation trust policies
type TrustPolicyConfig struct {
	Enabled                 bool
	DenyList                []string // blocked servers (supports wildcards like *.spam.domain)
	AllowList               []string // if non-empty, only these servers are allowed
	RequireNotarySignatures int      // minimum notary signatures required (0 = disabled)
	MaxKeyAgeHours          int      // reject keys older than this (0 = disabled)
	RequireWellKnown        bool     // require .well-known for non-IP servers
	RequireValidTLS         bool     // require valid TLS certificate
	BlockPrivateIPs         bool     // block requests to private IP ranges
}

// TransparencyConfig key transparency log settings
type TransparencyConfig struct {
	Enabled       bool
	LogAllKeys    bool   // log all observed keys
	LogKeyChanges bool   // log key rotations
	LogAnomalies  bool   // log suspicious activity
	RetentionDays int    // how long to keep log entries
	TableName     string // PostgreSQL table name
}

// ClusterConfig distributed notary cluster settings
type ClusterConfig struct {
	Enabled          bool
	NodeID           string   // unique identifier for this node
	BindAddress      string   // local cluster bind address
	BindPort         int      // local cluster bind port
	AdvertiseAddress string   // address announced to peers
	AdvertisePort    int      // port announced to peers
	Seeds            []string // seed nodes for discovery / raft peers
	ConsensusMode    string   // "raft" or "crdt"
	SyncInterval     int      // seconds between state sync
	SharedSecret     string   // required HMAC secret for cluster transport
}

// Load loads config from file or environment variables
func Load() (*Config, error) {
	config := &Config{}
	setDefaults(config)

	// Try config file paths
	var configPath string
	paths := []string{"config.yaml", "/etc/mxkeys/config.yaml"}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath != "" {
		m, err := zeroconfig.Load(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}

		// Apply env overrides
		zeroconfig.WithEnvOverride(m, "MXKEYS")
		applyMapConfig(config, m, configPath)
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func setDefaults(c *Config) {
	c.Server.Port = 8448
	c.Server.Name = "mxkeys.org"
	c.Server.BindAddress = "0.0.0.0"

	c.Database.URL = ""
	c.Database.MaxConnections = 10
	c.Database.MaxIdleConnections = 2

	c.Logging.Level = "info"
	c.Logging.Format = "text"

	c.Keys.StoragePath = "/var/lib/mxkeys/keys"
	c.Keys.ValidityHours = 24
	c.Keys.CacheTTLHours = 1
	c.Keys.FetchTimeoutS = 30
	c.Keys.CleanupHours = 6

	c.TrustedServers.Fallback = []string{"matrix.org"}

	c.RateLimit.RequestsPerSecond = 50
	c.RateLimit.Burst = 100
	c.RateLimit.QueryPerSecond = 10
	c.RateLimit.QueryBurst = 20

	c.Security.MaxServerNameLength = 255
	c.Security.MaxServersPerQuery = 100
	c.Security.MaxJSONDepth = 10
	c.Security.MaxSignaturesPerKey = 10
	c.Security.RequireRequestID = false
	c.Security.TrustForwardedHeaders = false
	c.Security.TrustedProxies = nil
	c.Security.EnterpriseAccessToken = ""

	// Trust policies (disabled by default)
	c.TrustPolicy.Enabled = false
	c.TrustPolicy.RequireNotarySignatures = 0
	c.TrustPolicy.MaxKeyAgeHours = 0
	c.TrustPolicy.RequireWellKnown = false
	c.TrustPolicy.RequireValidTLS = false
	c.TrustPolicy.BlockPrivateIPs = true

	// Transparency log (disabled by default)
	c.Transparency.Enabled = false
	c.Transparency.LogAllKeys = true
	c.Transparency.LogKeyChanges = true
	c.Transparency.LogAnomalies = true
	c.Transparency.RetentionDays = 365
	c.Transparency.TableName = "key_transparency_log"

	// Cluster (disabled by default)
	c.Cluster.Enabled = false
	c.Cluster.BindAddress = "0.0.0.0"
	c.Cluster.BindPort = 7946
	c.Cluster.AdvertiseAddress = ""
	c.Cluster.AdvertisePort = 0
	c.Cluster.ConsensusMode = "crdt"
	c.Cluster.SyncInterval = 5
	c.Cluster.SharedSecret = ""
}
