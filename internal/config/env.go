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
	"os"
	"strconv"
	"strings"
)

// applyEnvOverrides applies MXKEYS_* environment variables.
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
	if v := os.Getenv("MXKEYS_SECURITY_TRUST_FORWARDED_HEADERS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.Security.TrustForwardedHeaders = b
		}
	}
	if v := os.Getenv("MXKEYS_SECURITY_TRUSTED_PROXIES"); v != "" {
		c.Security.TrustedProxies = splitCSV(v)
	}
	if v := os.Getenv("MXKEYS_SECURITY_ENTERPRISE_ACCESS_TOKEN"); v != "" {
		c.Security.EnterpriseAccessToken = v
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
	if v := os.Getenv("MXKEYS_CLUSTER_ADVERTISE_ADDRESS"); v != "" {
		c.Cluster.AdvertiseAddress = v
	}
	if v := os.Getenv("MXKEYS_CLUSTER_ADVERTISE_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			c.Cluster.AdvertisePort = i
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
	if v := os.Getenv("MXKEYS_CLUSTER_SHARED_SECRET"); v != "" {
		c.Cluster.SharedSecret = v
	}

	if v := os.Getenv("MXKEYS_TRUSTED_NOTARIES"); v != "" {
		c.Security.TrustedNotaries = parseTrustedNotariesEnv(v)
	}
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
