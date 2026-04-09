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
	zeroconfig "mxkeys/internal/zero/config"
)

func applyMapConfig(config *Config, m map[string]interface{}, _ string) {
	// Map values to config struct.
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

	applyRateLimitMap(config, m)
	applySecurityMap(config, m)
	applyTrustPolicyMap(config, m)
	applyTransparencyMap(config, m)
	applyClusterMap(config, m)

	if trusted := parseTrustedNotariesFromMap(m); len(trusted) > 0 {
		config.Security.TrustedNotaries = trusted
	}
}

func applyRateLimitMap(config *Config, m map[string]interface{}) {
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
}

func applySecurityMap(config *Config, m map[string]interface{}) {
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
	if zeroconfig.Has(m, "security.trust_forwarded_headers") {
		config.Security.TrustForwardedHeaders = zeroconfig.GetBool(m, "security.trust_forwarded_headers")
	}
	if v := zeroconfig.GetStringSlice(m, "security.trusted_proxies"); len(v) > 0 {
		config.Security.TrustedProxies = v
	}
	if v := zeroconfig.GetString(m, "security.enterprise_access_token"); v != "" {
		config.Security.EnterpriseAccessToken = v
	}
}

func applyTrustPolicyMap(config *Config, m map[string]interface{}) {
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
}

func applyTransparencyMap(config *Config, m map[string]interface{}) {
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
}

func applyClusterMap(config *Config, m map[string]interface{}) {
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
	if v := zeroconfig.GetString(m, "cluster.advertise_address"); v != "" {
		config.Cluster.AdvertiseAddress = v
	}
	if v := zeroconfig.GetInt(m, "cluster.advertise_port"); v > 0 {
		config.Cluster.AdvertisePort = v
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
	if v := zeroconfig.GetString(m, "cluster.shared_secret"); v != "" {
		config.Cluster.SharedSecret = v
	}
}
