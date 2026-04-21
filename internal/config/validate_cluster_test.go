package config

import (
	"strings"
	"testing"
	"time"
)

// TrustedNotary / cluster-TLS / raft-specific validation cases live
// here so validate_test.go stays focused on the server / database /
// keys / rate_limit / security / trust_policy / transparency core
// and the aggregate per-file budget stays under the hard ceiling
// (see ADR-0010).

func TestValidateTrustedNotaries(t *testing.T) {
	// A valid ed25519 public key (32 bytes) in raw base64. Any deterministic
	// byte stream of length 32 is a valid placeholder for this test since
	// validation only checks length and shape, not curve-point validity.
	validKey := "Nzxs2Mh0Fb+Uhv3uTE47iWBoCGY8oSa11BZX9S7W6RE"

	tests := []struct {
		name     string
		notaries []TrustedNotary
		errMatch string
	}{
		{
			name:     "placeholder rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: "base64-encoded-public-key"}},
			errMatch: "placeholder",
		},
		{
			name:     "empty server_name rejected",
			notaries: []TrustedNotary{{ServerName: "", KeyID: "ed25519:auto", PublicKey: validKey}},
			errMatch: "server_name is required",
		},
		{
			name:     "empty key_id rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "", PublicKey: validKey}},
			errMatch: "key_id is required",
		},
		{
			name:     "empty public_key rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: ""}},
			errMatch: "public_key is required",
		},
		{
			name:     "non-base64 public_key rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: "not base64!!"}},
			errMatch: "invalid base64",
		},
		{
			name:     "wrong-length public_key rejected",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: "YWJj"}}, // base64("abc"), 3 bytes
			errMatch: "has length",
		},
		{
			name:     "valid entry accepted",
			notaries: []TrustedNotary{{ServerName: "matrix.org", KeyID: "ed25519:auto", PublicKey: validKey}},
		},
		{
			name:     "empty list accepted",
			notaries: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.Security.TrustedNotaries = tt.notaries
			err := cfg.Validate()
			if tt.errMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errMatch)
			}
			if !strings.Contains(err.Error(), tt.errMatch) {
				t.Fatalf("unexpected error %q, want substring %q", err.Error(), tt.errMatch)
			}
		})
	}
}

func TestValidateClusterTLS(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*Config)
		errMatch string
	}{
		{
			name:     "disabled TLS always valid",
			mutate:   func(c *Config) { c.Cluster.TLS.Enabled = false },
			errMatch: "",
		},
		{
			name: "missing cert_file rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = ""
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
			},
			errMatch: "cert_file is required",
		},
		{
			name: "missing key_file rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = ""
				c.Cluster.TLS.CAFile = "/ca"
			},
			errMatch: "key_file is required",
		},
		{
			name: "missing ca_file rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = ""
			},
			errMatch: "ca_file is required",
		},
		{
			name: "invalid min_version rejected",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
				c.Cluster.TLS.MinVersion = "1.0"
			},
			errMatch: "min_version must be",
		},
		{
			name: "min_version 1.2 rejected (cluster is TLS 1.3 only)",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
				c.Cluster.TLS.MinVersion = "1.2"
			},
			errMatch: "does not support TLS 1.2",
		},
		{
			name: "min_version 1.3 accepted",
			mutate: func(c *Config) {
				c.Cluster.TLS.Enabled = true
				c.Cluster.TLS.CertFile = "/c"
				c.Cluster.TLS.KeyFile = "/k"
				c.Cluster.TLS.CAFile = "/ca"
				c.Cluster.TLS.MinVersion = "1.3"
			},
			errMatch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			// Enable cluster so the TLS branch is exercised.
			cfg.Cluster.Enabled = true
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.errMatch == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errMatch)
			}
			if !strings.Contains(err.Error(), tt.errMatch) {
				t.Fatalf("unexpected error %q, want substring %q", err.Error(), tt.errMatch)
			}
		})
	}
}

func TestValidateRaftRequiresStateDir(t *testing.T) {
	cfg := validConfig()
	cfg.Cluster.Enabled = true
	cfg.Cluster.ConsensusMode = "raft"
	cfg.Cluster.RaftStateDir = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("raft mode without raft_state_dir must fail validation")
	}
	if !strings.Contains(err.Error(), "cluster.raft_state_dir is required") {
		t.Fatalf("unexpected error %q", err.Error())
	}

	cfg.Cluster.RaftStateDir = "/var/lib/mxkeys/raft"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("raft mode with raft_state_dir should pass: %v", err)
	}
}

func TestValidateRaftCompactionTunables(t *testing.T) {
	// Zero values MUST be accepted; they mean "use the built-in
	// default" on the cluster side.
	cfg := validConfig()
	cfg.Cluster.Enabled = true
	cfg.Cluster.ConsensusMode = "raft"
	cfg.Cluster.RaftStateDir = "/var/lib/mxkeys/raft"
	cfg.Cluster.RaftCompactionInterval = 0
	cfg.Cluster.RaftCompactionLogThreshold = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("zero compaction tunables must fall back to defaults, got %v", err)
	}

	// Negative values MUST be rejected on both knobs.
	cfg.Cluster.RaftCompactionInterval = -1 * time.Second
	if err := cfg.Validate(); err == nil ||
		!strings.Contains(err.Error(), "raft_compaction_interval") {
		t.Fatalf("negative interval must be rejected, got %v", err)
	}
	cfg.Cluster.RaftCompactionInterval = 0
	cfg.Cluster.RaftCompactionLogThreshold = -5
	if err := cfg.Validate(); err == nil ||
		!strings.Contains(err.Error(), "raft_compaction_log_threshold") {
		t.Fatalf("negative threshold must be rejected, got %v", err)
	}

	// Sub-second interval MUST be rejected; the ticker floor keeps
	// the loop from going pathological under a config typo like
	// "5ms".
	cfg.Cluster.RaftCompactionLogThreshold = 0
	cfg.Cluster.RaftCompactionInterval = 500 * time.Millisecond
	if err := cfg.Validate(); err == nil ||
		!strings.Contains(err.Error(), ">= 1s") {
		t.Fatalf("sub-second interval must be rejected, got %v", err)
	}

	// A legitimate override (2s / 32) MUST pass.
	cfg.Cluster.RaftCompactionInterval = 2 * time.Second
	cfg.Cluster.RaftCompactionLogThreshold = 32
	if err := cfg.Validate(); err != nil {
		t.Fatalf("legitimate override must pass, got %v", err)
	}
}
