/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 21:10:00 UTC
 * Status: Created
 */

package config

import (
	"strings"
	"testing"
	"time"
)

func TestApplyServerInfoMap(t *testing.T) {
	cfg := validConfig()
	setDefaults(cfg)

	applyMapConfig(cfg, map[string]interface{}{
		"server_info": map[string]interface{}{
			"enabled":         true,
			"cache_ttl":       "12h",
			"request_timeout": "3s",
			"whois_enabled":   true,
		},
	}, "")

	if !cfg.ServerInfo.Enabled || !cfg.ServerInfo.WhoisEnabled {
		t.Fatalf("server_info booleans were not applied: %+v", cfg.ServerInfo)
	}
	if cfg.ServerInfo.CacheTTL != 12*time.Hour {
		t.Fatalf("cache_ttl = %s, want 12h", cfg.ServerInfo.CacheTTL)
	}
	if cfg.ServerInfo.RequestTimeout != 3*time.Second {
		t.Fatalf("request_timeout = %s, want 3s", cfg.ServerInfo.RequestTimeout)
	}
}

func TestValidateServerInfo(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ServerInfoConfig
		errMatch string
	}{
		{name: "disabled accepts negative values", cfg: ServerInfoConfig{Enabled: false, CacheTTL: -time.Second}},
		{name: "enabled accepts zero defaults", cfg: ServerInfoConfig{Enabled: true}},
		{name: "enabled accepts explicit values", cfg: ServerInfoConfig{Enabled: true, CacheTTL: time.Minute, RequestTimeout: time.Second}},
		{name: "negative cache ttl rejected", cfg: ServerInfoConfig{Enabled: true, CacheTTL: -time.Second}, errMatch: "cache_ttl"},
		{name: "too small cache ttl rejected", cfg: ServerInfoConfig{Enabled: true, CacheTTL: time.Second}, errMatch: "cache_ttl"},
		{name: "negative timeout rejected", cfg: ServerInfoConfig{Enabled: true, RequestTimeout: -time.Second}, errMatch: "request_timeout"},
		{name: "too small timeout rejected", cfg: ServerInfoConfig{Enabled: true, RequestTimeout: time.Millisecond}, errMatch: "request_timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerInfo(tt.cfg)
			if tt.errMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.errMatch) {
				t.Fatalf("error = %v, want substring %q", err, tt.errMatch)
			}
		})
	}
}
