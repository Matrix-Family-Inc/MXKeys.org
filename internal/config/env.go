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
	"os"
	"strconv"
	"strings"
)

func envInt(name string, dst *int) error {
	v := os.Getenv(name)
	if v == "" {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	*dst = i
	return nil
}

func envFloat(name string, dst *float64) error {
	v := os.Getenv(name)
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	*dst = f
	return nil
}

func envBool(name string, dst *bool) error {
	v := os.Getenv(name)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	*dst = b
	return nil
}

// applyEnvOverrides applies MXKEYS_* environment variables.
// Returns an error if any numeric/boolean env value cannot be parsed.
func applyEnvOverrides(c *Config) error {
	if v := os.Getenv("MXKEYS_SERVER_NAME"); v != "" {
		c.Server.Name = v
	}
	if err := envInt("MXKEYS_SERVER_PORT", &c.Server.Port); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_SERVER_BIND_ADDRESS"); v != "" {
		c.Server.BindAddress = v
	}
	if v := os.Getenv("MXKEYS_DATABASE_URL"); v != "" {
		c.Database.URL = v
	}
	if err := envInt("MXKEYS_DATABASE_MAX_CONNECTIONS", &c.Database.MaxConnections); err != nil {
		return err
	}
	if err := envInt("MXKEYS_DATABASE_MAX_IDLE_CONNECTIONS", &c.Database.MaxIdleConnections); err != nil {
		return err
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
	if err := envInt("MXKEYS_KEYS_VALIDITY_HOURS", &c.Keys.ValidityHours); err != nil {
		return err
	}
	if err := envInt("MXKEYS_KEYS_CACHE_TTL_HOURS", &c.Keys.CacheTTLHours); err != nil {
		return err
	}
	if err := envInt("MXKEYS_KEYS_FETCH_TIMEOUT_S", &c.Keys.FetchTimeoutS); err != nil {
		return err
	}
	if err := envInt("MXKEYS_KEYS_CLEANUP_HOURS", &c.Keys.CleanupHours); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_TRUSTED_SERVERS_FALLBACK"); v != "" {
		c.TrustedServers.Fallback = splitCSV(v)
	}
	if err := envFloat("MXKEYS_RATE_LIMIT_REQUESTS_PER_SECOND", &c.RateLimit.RequestsPerSecond); err != nil {
		return err
	}
	if err := envInt("MXKEYS_RATE_LIMIT_BURST", &c.RateLimit.Burst); err != nil {
		return err
	}
	if err := envFloat("MXKEYS_RATE_LIMIT_QUERY_PER_SECOND", &c.RateLimit.QueryPerSecond); err != nil {
		return err
	}
	if err := envInt("MXKEYS_RATE_LIMIT_QUERY_BURST", &c.RateLimit.QueryBurst); err != nil {
		return err
	}

	if err := envInt("MXKEYS_SECURITY_MAX_SERVER_NAME_LENGTH", &c.Security.MaxServerNameLength); err != nil {
		return err
	}
	if err := envInt("MXKEYS_SECURITY_MAX_SERVERS_PER_QUERY", &c.Security.MaxServersPerQuery); err != nil {
		return err
	}
	if err := envInt("MXKEYS_SECURITY_MAX_JSON_DEPTH", &c.Security.MaxJSONDepth); err != nil {
		return err
	}
	if err := envInt("MXKEYS_SECURITY_MAX_SIGNATURES_PER_KEY", &c.Security.MaxSignaturesPerKey); err != nil {
		return err
	}
	if err := envBool("MXKEYS_SECURITY_REQUIRE_REQUEST_ID", &c.Security.RequireRequestID); err != nil {
		return err
	}
	if err := envBool("MXKEYS_SECURITY_TRUST_FORWARDED_HEADERS", &c.Security.TrustForwardedHeaders); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_SECURITY_TRUSTED_PROXIES"); v != "" {
		c.Security.TrustedProxies = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_SECURITY_ENTERPRISE_ACCESS_TOKEN"); v != "" {
		c.Security.EnterpriseAccessToken = v
	}

	if err := envBool("MXKEYS_TRUST_POLICY_ENABLED", &c.TrustPolicy.Enabled); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_DENY_LIST"); v != "" {
		c.TrustPolicy.DenyList = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_TRUST_POLICY_ALLOW_LIST"); v != "" {
		c.TrustPolicy.AllowList = splitCSV(v)
	}
	if err := envInt("MXKEYS_TRUST_POLICY_REQUIRE_NOTARY_SIGNATURES", &c.TrustPolicy.RequireNotarySignatures); err != nil {
		return err
	}
	if err := envInt("MXKEYS_TRUST_POLICY_MAX_KEY_AGE_HOURS", &c.TrustPolicy.MaxKeyAgeHours); err != nil {
		return err
	}
	if err := envBool("MXKEYS_TRUST_POLICY_REQUIRE_WELL_KNOWN", &c.TrustPolicy.RequireWellKnown); err != nil {
		return err
	}
	if err := envBool("MXKEYS_TRUST_POLICY_REQUIRE_VALID_TLS", &c.TrustPolicy.RequireValidTLS); err != nil {
		return err
	}
	if err := envBool("MXKEYS_TRUST_POLICY_BLOCK_PRIVATE_IPS", &c.TrustPolicy.BlockPrivateIPs); err != nil {
		return err
	}

	if err := envBool("MXKEYS_TRANSPARENCY_ENABLED", &c.Transparency.Enabled); err != nil {
		return err
	}
	if err := envBool("MXKEYS_TRANSPARENCY_LOG_ALL_KEYS", &c.Transparency.LogAllKeys); err != nil {
		return err
	}
	if err := envBool("MXKEYS_TRANSPARENCY_LOG_KEY_CHANGES", &c.Transparency.LogKeyChanges); err != nil {
		return err
	}
	if err := envBool("MXKEYS_TRANSPARENCY_LOG_ANOMALIES", &c.Transparency.LogAnomalies); err != nil {
		return err
	}
	if err := envInt("MXKEYS_TRANSPARENCY_RETENTION_DAYS", &c.Transparency.RetentionDays); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_TRANSPARENCY_TABLE_NAME"); v != "" {
		c.Transparency.TableName = v
	}

	if err := envBool("MXKEYS_CLUSTER_ENABLED", &c.Cluster.Enabled); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_CLUSTER_NODE_ID"); v != "" {
		c.Cluster.NodeID = v
	}
	if v := os.Getenv("MXKEYS_CLUSTER_BIND_ADDRESS"); v != "" {
		c.Cluster.BindAddress = v
	}
	if err := envInt("MXKEYS_CLUSTER_BIND_PORT", &c.Cluster.BindPort); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_CLUSTER_ADVERTISE_ADDRESS"); v != "" {
		c.Cluster.AdvertiseAddress = v
	}
	if err := envInt("MXKEYS_CLUSTER_ADVERTISE_PORT", &c.Cluster.AdvertisePort); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_CLUSTER_SEEDS"); v != "" {
		c.Cluster.Seeds = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_CLUSTER_CONSENSUS_MODE"); v != "" {
		c.Cluster.ConsensusMode = v
	}
	if err := envInt("MXKEYS_CLUSTER_SYNC_INTERVAL", &c.Cluster.SyncInterval); err != nil {
		return err
	}
	if v := os.Getenv("MXKEYS_CLUSTER_SHARED_SECRET"); v != "" {
		c.Cluster.SharedSecret = v
	}

	if v := os.Getenv("MXKEYS_TRUSTED_NOTARIES"); v != "" {
		c.Security.TrustedNotaries = parseTrustedNotariesEnv(v)
	}
	return nil
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
