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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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

		if v := zeroconfig.GetFloat(m, "rate_limit.requests_per_second"); v > 0 {
			config.RateLimit.RequestsPerSecond = v
		}
		if v := zeroconfig.GetInt(m, "rate_limit.burst"); v > 0 {
			config.RateLimit.Burst = v
		}
		if v := zeroconfig.GetFloat(m, "rate_limit.query_per_second"); v > 0 {
			config.RateLimit.QueryPerSecond = v
		}
		if v := zeroconfig.GetInt(m, "rate_limit.query_burst"); v > 0 {
			config.RateLimit.QueryBurst = v
		}

		if v := zeroconfig.GetInt(m, "security.max_server_name_length"); v > 0 {
			config.Security.MaxServerNameLength = v
		}
		if v := zeroconfig.GetInt(m, "security.max_servers_per_query"); v > 0 {
			config.Security.MaxServersPerQuery = v
		}
		if v := zeroconfig.GetInt(m, "security.max_json_depth"); v > 0 {
			config.Security.MaxJSONDepth = v
		}
		if v := zeroconfig.GetInt(m, "security.max_signatures_per_key"); v > 0 {
			config.Security.MaxSignaturesPerKey = v
		}
		if zeroconfig.Has(m, "security.require_request_id") {
			config.Security.RequireRequestID = zeroconfig.GetBool(m, "security.require_request_id")
		}

		if zeroconfig.Has(m, "trust_policy.enabled") {
			config.TrustPolicy.Enabled = zeroconfig.GetBool(m, "trust_policy.enabled")
		}
		if v := zeroconfig.GetStringSlice(m, "trust_policy.deny_list"); len(v) > 0 {
			config.TrustPolicy.DenyList = v
		}
		if v := zeroconfig.GetStringSlice(m, "trust_policy.allow_list"); len(v) > 0 {
			config.TrustPolicy.AllowList = v
		}
		if v := zeroconfig.GetInt(m, "trust_policy.require_notary_signatures"); v >= 0 {
			config.TrustPolicy.RequireNotarySignatures = v
		}
		if v := zeroconfig.GetInt(m, "trust_policy.max_key_age_hours"); v >= 0 {
			config.TrustPolicy.MaxKeyAgeHours = v
		}
		if zeroconfig.Has(m, "trust_policy.require_well_known") {
			config.TrustPolicy.RequireWellKnown = zeroconfig.GetBool(m, "trust_policy.require_well_known")
		}
		if zeroconfig.Has(m, "trust_policy.require_valid_tls") {
			config.TrustPolicy.RequireValidTLS = zeroconfig.GetBool(m, "trust_policy.require_valid_tls")
		}
		if zeroconfig.Has(m, "trust_policy.block_private_ips") {
			config.TrustPolicy.BlockPrivateIPs = zeroconfig.GetBool(m, "trust_policy.block_private_ips")
		}

		if zeroconfig.Has(m, "transparency.enabled") {
			config.Transparency.Enabled = zeroconfig.GetBool(m, "transparency.enabled")
		}
		if zeroconfig.Has(m, "transparency.log_all_keys") {
			config.Transparency.LogAllKeys = zeroconfig.GetBool(m, "transparency.log_all_keys")
		}
		if zeroconfig.Has(m, "transparency.log_key_changes") {
			config.Transparency.LogKeyChanges = zeroconfig.GetBool(m, "transparency.log_key_changes")
		}
		if zeroconfig.Has(m, "transparency.log_anomalies") {
			config.Transparency.LogAnomalies = zeroconfig.GetBool(m, "transparency.log_anomalies")
		}
		if v := zeroconfig.GetInt(m, "transparency.retention_days"); v > 0 {
			config.Transparency.RetentionDays = v
		}
		if v := zeroconfig.GetString(m, "transparency.table_name"); v != "" {
			config.Transparency.TableName = v
		}

		if zeroconfig.Has(m, "cluster.enabled") {
			config.Cluster.Enabled = zeroconfig.GetBool(m, "cluster.enabled")
		}
		if v := zeroconfig.GetString(m, "cluster.node_id"); v != "" {
			config.Cluster.NodeID = v
		}
		if v := zeroconfig.GetString(m, "cluster.bind_address"); v != "" {
			config.Cluster.BindAddress = v
		}
		if v := zeroconfig.GetInt(m, "cluster.bind_port"); v > 0 {
			config.Cluster.BindPort = v
		}
		if v := zeroconfig.GetStringSlice(m, "cluster.seeds"); len(v) > 0 {
			config.Cluster.Seeds = v
		}
		if v := zeroconfig.GetString(m, "cluster.consensus_mode"); v != "" {
			config.Cluster.ConsensusMode = v
		}
		if v := zeroconfig.GetInt(m, "cluster.sync_interval"); v > 0 {
			config.Cluster.SyncInterval = v
		}

		if trusted := parseTrustedNotariesFromYAML(configPath); len(trusted) > 0 {
			config.Security.TrustedNotaries = trusted
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
		c.TrustedServers.Fallback = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_RATE_LIMIT_REQUESTS_PER_SECOND"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.RateLimit.RequestsPerSecond = f
		}
	}
	if v := os.Getenv("MXKEYS_RATE_LIMIT_BURST"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.RateLimit.Burst = i
		}
	}
	if v := os.Getenv("MXKEYS_RATE_LIMIT_QUERY_PER_SECOND"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.RateLimit.QueryPerSecond = f
		}
	}
	if v := os.Getenv("MXKEYS_RATE_LIMIT_QUERY_BURST"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.RateLimit.QueryBurst = i
		}
	}

	if v := os.Getenv("MXKEYS_SECURITY_MAX_SERVER_NAME_LENGTH"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Security.MaxServerNameLength = i
		}
	}
	if v := os.Getenv("MXKEYS_SECURITY_MAX_SERVERS_PER_QUERY"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Security.MaxServersPerQuery = i
		}
	}
	if v := os.Getenv("MXKEYS_SECURITY_MAX_JSON_DEPTH"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Security.MaxJSONDepth = i
		}
	}
	if v := os.Getenv("MXKEYS_SECURITY_MAX_SIGNATURES_PER_KEY"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Security.MaxSignaturesPerKey = i
		}
	}
	if v := os.Getenv("MXKEYS_SECURITY_REQUIRE_REQUEST_ID"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Security.RequireRequestID = b
		}
	}

	if v := os.Getenv("MXKEYS_TRUST_POLICY_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.TrustPolicy.Enabled = b
		}
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_DENY_LIST"); v != "" {
		c.TrustPolicy.DenyList = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_ALLOW_LIST"); v != "" {
		c.TrustPolicy.AllowList = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_REQUIRE_NOTARY_SIGNATURES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.TrustPolicy.RequireNotarySignatures = i
		}
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_MAX_KEY_AGE_HOURS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.TrustPolicy.MaxKeyAgeHours = i
		}
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_REQUIRE_WELL_KNOWN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.TrustPolicy.RequireWellKnown = b
		}
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_REQUIRE_VALID_TLS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.TrustPolicy.RequireValidTLS = b
		}
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_BLOCK_PRIVATE_IPS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.TrustPolicy.BlockPrivateIPs = b
		}
	}

	if v := os.Getenv("MXKEYS_TRANSPARENCY_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Transparency.Enabled = b
		}
	}
	if v := os.Getenv("MXKEYS_TRANSPARENCY_LOG_ALL_KEYS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Transparency.LogAllKeys = b
		}
	}
	if v := os.Getenv("MXKEYS_TRANSPARENCY_LOG_KEY_CHANGES"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Transparency.LogKeyChanges = b
		}
	}
	if v := os.Getenv("MXKEYS_TRANSPARENCY_LOG_ANOMALIES"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Transparency.LogAnomalies = b
		}
	}
	if v := os.Getenv("MXKEYS_TRANSPARENCY_RETENTION_DAYS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Transparency.RetentionDays = i
		}
	}
	if v := os.Getenv("MXKEYS_TRANSPARENCY_TABLE_NAME"); v != "" {
		c.Transparency.TableName = v
	}

	if v := os.Getenv("MXKEYS_CLUSTER_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Cluster.Enabled = b
		}
	}
	if v := os.Getenv("MXKEYS_CLUSTER_NODE_ID"); v != "" {
		c.Cluster.NodeID = v
	}
	if v := os.Getenv("MXKEYS_CLUSTER_BIND_ADDRESS"); v != "" {
		c.Cluster.BindAddress = v
	}
	if v := os.Getenv("MXKEYS_CLUSTER_BIND_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Cluster.BindPort = i
		}
	}
	if v := os.Getenv("MXKEYS_CLUSTER_SEEDS"); v != "" {
		c.Cluster.Seeds = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_CLUSTER_CONSENSUS_MODE"); v != "" {
		c.Cluster.ConsensusMode = v
	}
	if v := os.Getenv("MXKEYS_CLUSTER_SYNC_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Cluster.SyncInterval = i
		}
	}

	if v := os.Getenv("MXKEYS_TRUSTED_NOTARIES"); v != "" {
		c.Security.TrustedNotaries = parseTrustedNotariesEnv(v)
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
	if c.RateLimit.RequestsPerSecond <= 0 {
		return fmt.Errorf("rate_limit.requests_per_second must be positive")
	}
	if c.RateLimit.Burst <= 0 {
		return fmt.Errorf("rate_limit.burst must be positive")
	}
	if c.RateLimit.QueryPerSecond <= 0 {
		return fmt.Errorf("rate_limit.query_per_second must be positive")
	}
	if c.RateLimit.QueryBurst <= 0 {
		return fmt.Errorf("rate_limit.query_burst must be positive")
	}
	if c.Security.MaxServerNameLength <= 0 {
		return fmt.Errorf("security.max_server_name_length must be positive")
	}
	if c.Security.MaxServersPerQuery <= 0 {
		return fmt.Errorf("security.max_servers_per_query must be positive")
	}
	if c.Security.MaxJSONDepth <= 0 {
		return fmt.Errorf("security.max_json_depth must be positive")
	}
	if c.Security.MaxSignaturesPerKey <= 0 {
		return fmt.Errorf("security.max_signatures_per_key must be positive")
	}
	if c.TrustPolicy.RequireNotarySignatures < 0 {
		return fmt.Errorf("trust_policy.require_notary_signatures must be non-negative")
	}
	if c.TrustPolicy.MaxKeyAgeHours < 0 {
		return fmt.Errorf("trust_policy.max_key_age_hours must be non-negative")
	}
	if c.Transparency.RetentionDays <= 0 {
		return fmt.Errorf("transparency.retention_days must be positive")
	}
	if c.Transparency.TableName == "" {
		return fmt.Errorf("transparency.table_name is required")
	}
	if c.Cluster.Enabled {
		if c.Cluster.BindAddress == "" {
			return fmt.Errorf("cluster.bind_address is required when cluster.enabled=true")
		}
		if c.Cluster.BindPort < 1 || c.Cluster.BindPort > 65535 {
			return fmt.Errorf("cluster.bind_port must be between 1 and 65535 when cluster.enabled=true")
		}
		if c.Cluster.SyncInterval <= 0 {
			return fmt.Errorf("cluster.sync_interval must be positive when cluster.enabled=true")
		}
		if c.Cluster.ConsensusMode != "crdt" && c.Cluster.ConsensusMode != "raft" {
			return fmt.Errorf("cluster.consensus_mode must be 'crdt' or 'raft'")
		}
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
	c.Cluster.BindAddress = "0.0.0.0"
	c.Cluster.BindPort = 7946
	c.Cluster.ConsensusMode = "crdt"
	c.Cluster.SyncInterval = 5
}

func splitCSV(v string) []string {
	raw := strings.Split(v, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func parseTrustedNotariesEnv(v string) []TrustedNotary {
	// Format:
	// MXKEYS_TRUSTED_NOTARIES="server|key_id|public_key;server2|key_id|public_key"
	chunks := strings.Split(v, ";")
	out := make([]TrustedNotary, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.Split(chunk, "|")
		if len(parts) != 3 {
			continue
		}
		entry := TrustedNotary{
			ServerName: strings.TrimSpace(parts[0]),
			KeyID:      strings.TrimSpace(parts[1]),
			PublicKey:  strings.TrimSpace(parts[2]),
		}
		if entry.ServerName != "" && entry.KeyID != "" && entry.PublicKey != "" {
			out = append(out, entry)
		}
	}
	return out
}

func parseTrustedNotariesFromYAML(configPath string) []TrustedNotary {
	if configPath == "" {
		return nil
	}
	// Do not attempt parsing non-yaml files.
	ext := strings.ToLower(filepath.Ext(configPath))
	if ext != ".yaml" && ext != ".yml" {
		return nil
	}

	f, err := os.Open(configPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var (
		result        []TrustedNotary
		inSection     bool
		sectionIndent int
		current       *TrustedNotary
	)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := countIndent(line)
		if !inSection {
			if trimmed == "trusted_notaries:" {
				inSection = true
				sectionIndent = indent
			}
			continue
		}

		// End of trusted_notaries section.
		if indent <= sectionIndent {
			if current != nil && isCompleteTrustedNotary(*current) {
				result = append(result, *current)
			}
			inSection = false
			current = nil
			if trimmed == "trusted_notaries:" {
				inSection = true
				sectionIndent = indent
			}
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			if current != nil && isCompleteTrustedNotary(*current) {
				result = append(result, *current)
			}
			current = &TrustedNotary{}
			inline := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if k, v, ok := parseKeyValue(inline); ok {
				assignTrustedNotaryField(current, k, v)
			}
			continue
		}

		if current == nil {
			current = &TrustedNotary{}
		}
		if k, v, ok := parseKeyValue(trimmed); ok {
			assignTrustedNotaryField(current, k, v)
		}
	}

	if current != nil && isCompleteTrustedNotary(*current) {
		result = append(result, *current)
	}

	return result
}

func countIndent(s string) int {
	count := 0
	for _, r := range s {
		if r == ' ' {
			count++
			continue
		}
		if r == '\t' {
			count += 2
			continue
		}
		break
	}
	return count
}

func parseKeyValue(s string) (string, string, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	value = strings.Trim(value, `"`)
	return key, value, key != ""
}

func assignTrustedNotaryField(target *TrustedNotary, key, value string) {
	switch key {
	case "server_name":
		target.ServerName = value
	case "key_id":
		target.KeyID = value
	case "public_key":
		target.PublicKey = value
	}
}

func isCompleteTrustedNotary(n TrustedNotary) bool {
	return n.ServerName != "" && n.KeyID != "" && n.PublicKey != ""
}
