/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package config

import (
	"crypto/ed25519"
	"encoding/base64"
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
		if isPlaceholderSecret(c.Cluster.SharedSecret) {
			return fmt.Errorf("cluster.shared_secret contains a placeholder value; set a long, random secret")
		}
		if len(c.Cluster.SharedSecret) < 32 {
			return fmt.Errorf("cluster.shared_secret must be at least 32 characters (recommended: 64+ random bytes base64-encoded)")
		}
		if c.Cluster.ConsensusMode != "crdt" && c.Cluster.ConsensusMode != "raft" {
			return fmt.Errorf("cluster.consensus_mode must be 'crdt' or 'raft'")
		}
		if err := validateClusterTLS(c.Cluster.TLS); err != nil {
			return err
		}
	}

	if err := validateTrustedNotaries(c.Security.TrustedNotaries); err != nil {
		return err
	}

	return nil
}

// validateTrustedNotaries checks that every pinned-notary entry has the
// operator actually filled in: server_name, key_id, and a base64 public
// key of the right ed25519 length. The example file ships
// `public_key: "base64-encoded-public-key"` as an obvious placeholder;
// copying the example verbatim used to boot a notary with a broken trust
// pin that only failed later on first fetch. Fail fast here instead.
func validateTrustedNotaries(notaries []TrustedNotary) error {
	for i, n := range notaries {
		label := fmt.Sprintf("trusted_notaries[%d]", i)
		if strings.TrimSpace(n.ServerName) == "" {
			return fmt.Errorf("%s.server_name is required", label)
		}
		if strings.TrimSpace(n.KeyID) == "" {
			return fmt.Errorf("%s.key_id is required (for %q)", label, n.ServerName)
		}
		raw := strings.TrimSpace(n.PublicKey)
		if raw == "" {
			return fmt.Errorf("%s.public_key is required (for %q)", label, n.ServerName)
		}
		if isPlaceholderPublicKey(raw) {
			return fmt.Errorf("%s.public_key is a placeholder (%q); set the real ed25519 public key for %s", label, raw, n.ServerName)
		}
		decoded, err := decodeNotaryPublicKey(raw)
		if err != nil {
			return fmt.Errorf("%s.public_key for %s: %w", label, n.ServerName, err)
		}
		if len(decoded) != ed25519.PublicKeySize {
			return fmt.Errorf("%s.public_key for %s has length %d, want %d", label, n.ServerName, len(decoded), ed25519.PublicKeySize)
		}
	}
	return nil
}

// decodeNotaryPublicKey accepts base64 in either raw-std or std encoding,
// mirroring the runtime decoding in internal/server/server.go so validation
// and runtime agree on what is acceptable.
func decodeNotaryPublicKey(raw string) ([]byte, error) {
	if b, err := base64.RawStdEncoding.DecodeString(raw); err == nil {
		return b, nil
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 public key")
	}
	return b, nil
}

// placeholderPublicKeys are obvious example strings that must never
// survive into a real config. Case-insensitive, whitespace-trimmed match.
var placeholderPublicKeys = map[string]struct{}{
	"base64-encoded-public-key": {},
	"replace-with-public-key":   {},
	"your-public-key-here":      {},
	"example":                   {},
}

func isPlaceholderPublicKey(raw string) bool {
	_, ok := placeholderPublicKeys[strings.ToLower(strings.TrimSpace(raw))]
	return ok
}

// validateClusterTLS ensures that when cluster TLS is enabled the three
// mandatory paths are present, the min_version (if set) is recognized,
// and that files exist on disk at startup (rather than deferring to the
// first connection attempt).
func validateClusterTLS(t ClusterTLSConfig) error {
	if !t.Enabled {
		return nil
	}
	if strings.TrimSpace(t.CertFile) == "" {
		return fmt.Errorf("cluster.tls.cert_file is required when cluster.tls.enabled=true")
	}
	if strings.TrimSpace(t.KeyFile) == "" {
		return fmt.Errorf("cluster.tls.key_file is required when cluster.tls.enabled=true")
	}
	if strings.TrimSpace(t.CAFile) == "" {
		return fmt.Errorf("cluster.tls.ca_file is required when cluster.tls.enabled=true")
	}
	if v := strings.TrimSpace(t.MinVersion); v != "" && v != "1.2" && v != "1.3" {
		return fmt.Errorf("cluster.tls.min_version must be '1.2' or '1.3', got %q", v)
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

// placeholderSecrets are distributed in config.example.yaml as obvious
// placeholders and must never survive into a real deployment.
var placeholderSecrets = map[string]struct{}{
	"replace-with-long-random-secret": {},
	"replace-with-long-random-token":  {},
	"change-me":                       {},
	"changeme":                        {},
	"default":                         {},
	"secret":                          {},
}

// isPlaceholderSecret reports whether the given secret looks like one of the
// documented example placeholders from config.example.yaml.
func isPlaceholderSecret(secret string) bool {
	_, ok := placeholderSecrets[strings.ToLower(strings.TrimSpace(secret))]
	return ok
}
