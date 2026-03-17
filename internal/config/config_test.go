package config

import (
	"testing"
)

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
	}
}

func TestValidConfigPasses(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid config should pass: %v", err)
	}
}

func TestEmptyServerNameReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Name = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty server.name should be rejected")
	}
}

func TestServerPortZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Error("server.port=0 should be rejected")
	}
}

func TestServerPortNegativeReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = -1
	if err := cfg.Validate(); err == nil {
		t.Error("negative server.port should be rejected")
	}
}

func TestServerPortTooHighReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 65536
	if err := cfg.Validate(); err == nil {
		t.Error("server.port > 65535 should be rejected")
	}
}

func TestEmptyBindAddressReject(t *testing.T) {
	cfg := validConfig()
	cfg.Server.BindAddress = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty bind_address should be rejected")
	}
}

func TestEmptyDatabaseURLReject(t *testing.T) {
	cfg := validConfig()
	cfg.Database.URL = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty database.url should be rejected")
	}
}

func TestEmptyKeysStoragePathReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.StoragePath = ""
	if err := cfg.Validate(); err == nil {
		t.Error("empty keys.storage_path should be rejected")
	}
}

func TestValidityHoursZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.ValidityHours = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.validity_hours=0 should be rejected")
	}
}

func TestValidityHoursNegativeReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.ValidityHours = -1
	if err := cfg.Validate(); err == nil {
		t.Error("negative keys.validity_hours should be rejected")
	}
}

func TestCacheTTLHoursZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.CacheTTLHours = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.cache_ttl_hours=0 should be rejected")
	}
}

func TestFetchTimeoutZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.FetchTimeoutS = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.fetch_timeout_s=0 should be rejected")
	}
}

func TestCleanupHoursZeroReject(t *testing.T) {
	cfg := validConfig()
	cfg.Keys.CleanupHours = 0
	if err := cfg.Validate(); err == nil {
		t.Error("keys.cleanup_hours=0 should be rejected")
	}
}

func TestValidPortBoundaries(t *testing.T) {
	tests := []struct {
		port    int
		isValid bool
	}{
		{1, true},
		{80, true},
		{443, true},
		{8448, true},
		{65535, true},
		{0, false},
		{-1, false},
		{65536, false},
		{100000, false},
	}

	for _, tt := range tests {
		cfg := validConfig()
		cfg.Server.Port = tt.port
		err := cfg.Validate()
		if tt.isValid && err != nil {
			t.Errorf("port %d should be valid, got error: %v", tt.port, err)
		}
		if !tt.isValid && err == nil {
			t.Errorf("port %d should be invalid", tt.port)
		}
	}
}
