package config

// validConfig returns a Config populated with valid values used as a baseline
// for validation tests. Individual tests mutate a field and re-run Validate().
func validConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Name:        "mxkeys.test",
			Port:        8448,
			BindAddress: "0.0.0.0",
		},
		Database: DatabaseConfig{
			URL:                "postgres://test:test@localhost/test",
			MaxConnections:     10,
			MaxIdleConnections: 2,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Keys: KeysConfig{
			StoragePath:   "/var/lib/mxkeys/keys",
			ValidityHours: 24,
			CacheTTLHours: 1,
			FetchTimeoutS: 30,
			CleanupHours:  6,
		},
		TrustedServers: TrustedServersConfig{
			Fallback: []string{"matrix.org"},
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 50,
			Burst:             100,
			QueryPerSecond:    10,
			QueryBurst:        20,
		},
		Security: SecurityConfig{
			MaxServerNameLength:   255,
			MaxServersPerQuery:    100,
			MaxJSONDepth:          10,
			MaxSignaturesPerKey:   10,
			TrustForwardedHeaders: false,
			TrustedProxies:        []string{"127.0.0.1/32"},
			EnterpriseAccessToken: "enterprise-token",
		},
		Transparency: TransparencyConfig{
			Enabled:       false,
			LogAllKeys:    true,
			LogKeyChanges: true,
			LogAnomalies:  true,
			RetentionDays: 365,
			TableName:     "key_transparency_log",
		},
		Cluster: ClusterConfig{
			Enabled:          false,
			BindAddress:      "127.0.0.1",
			BindPort:         7946,
			AdvertiseAddress: "127.0.0.1",
			AdvertisePort:    7946,
			ConsensusMode:    "crdt",
			SyncInterval:     5,
			SharedSecret:     "cluster-secret",
		},
	}
}
