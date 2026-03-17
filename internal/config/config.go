/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 */

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

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
	MaxServerNameLength int
	MaxServersPerQuery  int
	MaxJSONDepth        int
	MaxSignaturesPerKey int
	RequireRequestID    bool
	TrustedNotaries     []TrustedNotary
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
	Enabled       bool
	NodeID        string   // unique identifier for this node
	BindAddress   string   // cluster communication address
	BindPort      int      // cluster communication port
	Seeds         []string // seed nodes for discovery
	ConsensusMode string   // "raft" or "crdt"
	SyncInterval  int      // seconds between state sync
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

		// Map values to config struct
		if v := zeroconfig.GetString(m, "server.name"); v != "" {
			config.Server.Name = v
		}
		if v := zeroconfig.GetInt(m, "server.port"); v != 0 {
			config.Server.Port = v
		}
		if v := zeroconfig.GetString(m, "server.bind_address"); v != "" {
			config.Server.BindAddress = v
		}

		if v := zeroconfig.GetString(m, "database.url"); v != "" {
			config.Database.URL = v
		}
		if v := zeroconfig.GetInt(m, "database.max_connections"); v != 0 {
			config.Database.MaxConnections = v
		}
		if v := zeroconfig.GetInt(m, "database.max_idle_connections"); v != 0 {
			config.Database.MaxIdleConnections = v
		}

		if v := zeroconfig.GetString(m, "logging.level"); v != "" {
			config.Logging.Level = v
		}
		if v := zeroconfig.GetString(m, "logging.format"); v != "" {
			config.Logging.Format = v
		}

		if v := zeroconfig.GetString(m, "keys.storage_path"); v != "" {
			config.Keys.StoragePath = v
		}
		if v := zeroconfig.GetInt(m, "keys.validity_hours"); v != 0 {
			config.Keys.ValidityHours = v
		}
		if v := zeroconfig.GetInt(m, "keys.cache_ttl_hours"); v != 0 {
			config.Keys.CacheTTLHours = v
		}
		if v := zeroconfig.GetInt(m, "keys.fetch_timeout_s"); v != 0 {
			config.Keys.FetchTimeoutS = v
		}
		if v := zeroconfig.GetInt(m, "keys.cleanup_hours"); v != 0 {
			config.Keys.CleanupHours = v
		}

		if v := zeroconfig.GetStringSlice(m, "trusted_servers.fallback"); len(v) > 0 {
			config.TrustedServers.Fallback = v
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// applyEnvOverrides applies MXKEYS_* environment variables
func applyEnvOverrides(c *Config) {
	if v := os.Getenv("MXKEYS_SERVER_NAME"); v != "" {
		c.Server.Name = v
	}
	if v := os.Getenv("MXKEYS_SERVER_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Server.Port = i
		}
	}
	if v := os.Getenv("MXKEYS_SERVER_BIND_ADDRESS"); v != "" {
		c.Server.BindAddress = v
	}
	if v := os.Getenv("MXKEYS_DATABASE_URL"); v != "" {
		c.Database.URL = v
	}
	if v := os.Getenv("MXKEYS_DATABASE_MAX_CONNECTIONS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Database.MaxConnections = i
		}
	}
	if v := os.Getenv("MXKEYS_DATABASE_MAX_IDLE_CONNECTIONS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Database.MaxIdleConnections = i
		}
	}
	if v := os.Getenv("MXKEYS_LOGGING_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("MXKEYS_LOGGING_FORMAT"); v != "" {
		c.Logging.Format = v
	}
	if v := os.Getenv("MXKEYS_KEYS_STORAGE_PATH"); v != "" {
		c.Keys.StoragePath = v
	}
	if v := os.Getenv("MXKEYS_KEYS_VALIDITY_HOURS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Keys.ValidityHours = i
		}
	}
	if v := os.Getenv("MXKEYS_KEYS_CACHE_TTL_HOURS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Keys.CacheTTLHours = i
		}
	}
	if v := os.Getenv("MXKEYS_KEYS_FETCH_TIMEOUT_S"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Keys.FetchTimeoutS = i
		}
	}
	if v := os.Getenv("MXKEYS_KEYS_CLEANUP_HOURS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Keys.CleanupHours = i
		}
	}
	if v := os.Getenv("MXKEYS_TRUSTED_SERVERS_FALLBACK"); v != "" {
		c.TrustedServers.Fallback = strings.Split(v, ",")
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Name == "" {
		return fmt.Errorf("server.name is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.BindAddress == "" {
		return fmt.Errorf("server.bind_address is required")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}
	if c.Keys.StoragePath == "" {
		return fmt.Errorf("keys.storage_path is required")
	}
	if c.Keys.ValidityHours <= 0 {
		return fmt.Errorf("keys.validity_hours must be positive")
	}
	if c.Keys.CacheTTLHours <= 0 {
		return fmt.Errorf("keys.cache_ttl_hours must be positive")
	}
	if c.Keys.FetchTimeoutS <= 0 {
		return fmt.Errorf("keys.fetch_timeout_s must be positive")
	}
	if c.Keys.CleanupHours <= 0 {
		return fmt.Errorf("keys.cleanup_hours must be positive")
	}
	return nil
}

func setDefaults(c *Config) {
	c.Server.Port = 8448
	c.Server.Name = "mxkeys.org"
	c.Server.BindAddress = "0.0.0.0"

	c.Database.URL = "postgres://mxkeys:mxkeys@localhost/mxkeys?sslmode=disable"
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
	c.Cluster.BindPort = 7946
	c.Cluster.ConsensusMode = "crdt"
	c.Cluster.SyncInterval = 5
}
