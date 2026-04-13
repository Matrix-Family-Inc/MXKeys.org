/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package config

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var safeSQLIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Validate validates the configuration.
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
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error")
	}
	switch c.Logging.Format {
	case "text", "json":
	default:
		return fmt.Errorf("logging.format must be one of: text, json")
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
	if c.Security.TrustForwardedHeaders && len(c.Security.TrustedProxies) == 0 {
		return fmt.Errorf("security.trusted_proxies is required when security.trust_forwarded_headers=true")
	}
	for _, proxy := range c.Security.TrustedProxies {
		if ip := net.ParseIP(strings.TrimSpace(proxy)); ip != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(strings.TrimSpace(proxy)); err != nil {
			return fmt.Errorf("security.trusted_proxies contains invalid CIDR or IP: %s", proxy)
		}
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
	if !safeSQLIdentifierPattern.MatchString(c.Transparency.TableName) {
		return fmt.Errorf("transparency.table_name must be a safe SQL identifier")
	}
	if c.Cluster.Enabled {
		if c.Cluster.BindAddress == "" {
			return fmt.Errorf("cluster.bind_address is required when cluster.enabled=true")
		}
		if c.Cluster.BindPort < 1 || c.Cluster.BindPort > 65535 {
			return fmt.Errorf("cluster.bind_port must be between 1 and 65535 when cluster.enabled=true")
		}
		if c.Cluster.AdvertisePort != 0 && (c.Cluster.AdvertisePort < 1 || c.Cluster.AdvertisePort > 65535) {
			return fmt.Errorf("cluster.advertise_port must be between 1 and 65535 when specified")
		}
		if c.Cluster.AdvertiseAddress == "" && isWildcardAddress(c.Cluster.BindAddress) {
			return fmt.Errorf("cluster.advertise_address is required when cluster.bind_address is wildcard")
		}
		if strings.TrimSpace(c.Cluster.AdvertiseAddress) != "" && isWildcardAddress(c.Cluster.AdvertiseAddress) {
			return fmt.Errorf("cluster.advertise_address must not be a wildcard address")
		}
		if c.Cluster.SyncInterval <= 0 {
			return fmt.Errorf("cluster.sync_interval must be positive when cluster.enabled=true")
		}
		if strings.TrimSpace(c.Cluster.SharedSecret) == "" {
			return fmt.Errorf("cluster.shared_secret is required when cluster.enabled=true")
		}
		if c.Cluster.ConsensusMode != "crdt" && c.Cluster.ConsensusMode != "raft" {
			return fmt.Errorf("cluster.consensus_mode must be 'crdt' or 'raft'")
		}
	}
	return nil
}

func isWildcardAddress(addr string) bool {
	switch strings.TrimSpace(addr) {
	case "", "0.0.0.0", "::", "[::]":
		return true
	default:
		return false
	}
}
