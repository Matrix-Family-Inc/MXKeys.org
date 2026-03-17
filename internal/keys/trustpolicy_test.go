package keys

import (
	"net"
	"testing"
	"time"
)

func TestTrustPolicyDisabled(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{Enabled: false})

	if v := tp.CheckServer("evil.server"); v != nil {
		t.Error("disabled policy should not reject")
	}
}

func TestTrustPolicyDenyList(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:  true,
		DenyList: []string{"evil.server", "bad.actor"},
	})

	if v := tp.CheckServer("evil.server"); v == nil {
		t.Error("should reject denied server")
	}

	if v := tp.CheckServer("bad.actor"); v == nil {
		t.Error("should reject denied server")
	}

	if v := tp.CheckServer("good.server"); v != nil {
		t.Error("should allow non-denied server")
	}
}

func TestTrustPolicyDenyListWildcard(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:  true,
		DenyList: []string{"*.spam.domain", "evil.*"},
	})

	if v := tp.CheckServer("foo.spam.domain"); v == nil {
		t.Error("should reject wildcard match *.spam.domain")
	}

	if v := tp.CheckServer("evil.server"); v == nil {
		t.Error("should reject wildcard match evil.*")
	}

	if v := tp.CheckServer("good.server"); v != nil {
		t.Error("should allow non-matching server")
	}
}

func TestTrustPolicyAllowList(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:   true,
		AllowList: []string{"matrix.org", "example.com"},
	})

	if v := tp.CheckServer("matrix.org"); v != nil {
		t.Error("should allow listed server")
	}

	if v := tp.CheckServer("other.server"); v == nil {
		t.Error("should reject non-listed server when allow list is set")
	}
}

func TestTrustPolicyAllowListWildcard(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:   true,
		AllowList: []string{"*.matrix.org"},
	})

	if v := tp.CheckServer("synapse.matrix.org"); v != nil {
		t.Error("should allow wildcard match")
	}

	if v := tp.CheckServer("matrix.org"); v == nil {
		t.Error("should reject non-matching (no subdomain)")
	}
}

func TestTrustPolicyBlockPrivateIPs(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:         true,
		BlockPrivateIPs: true,
	})

	tests := []struct {
		server  string
		blocked bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"127.0.0.1", true},
		{"172.16.0.1", true},
		{"8.8.8.8", false},
		{"matrix.org", false},
		{"[::1]", true},
		{"[2001:db8::1]", false},
	}

	for _, tt := range tests {
		v := tp.CheckServer(tt.server)
		if tt.blocked && v == nil {
			t.Errorf("%s should be blocked", tt.server)
		}
		if !tt.blocked && v != nil {
			t.Errorf("%s should not be blocked: %v", tt.server, v)
		}
	}
}

func TestTrustPolicyRequireWellKnown(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:          true,
		RequireWellKnown: true,
	})

	if v := tp.CheckServer("matrix.org"); v != nil {
		t.Fatalf("expected matrix.org allowed, got %v", v)
	}

	if v := tp.CheckServer("matrix.org:8448"); v == nil || v.Rule != "require_well_known" {
		t.Fatalf("expected require_well_known violation for explicit port, got %v", v)
	}

	if v := tp.CheckServer("1.2.3.4"); v == nil || v.Rule != "require_well_known" {
		t.Fatalf("expected require_well_known violation for IP literal, got %v", v)
	}
}

func TestTrustPolicyRequireValidTLS(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:         true,
		RequireValidTLS: true,
	})

	if v := tp.CheckServer("federation.matrix.org"); v != nil {
		t.Fatalf("expected federation.matrix.org allowed, got %v", v)
	}

	if v := tp.CheckServer("8.8.8.8"); v == nil || v.Rule != "require_valid_tls" {
		t.Fatalf("expected require_valid_tls violation for IP literal, got %v", v)
	}

	if v := tp.CheckServer("localhost"); v == nil || v.Rule != "require_valid_tls" {
		t.Fatalf("expected require_valid_tls violation for localhost, got %v", v)
	}

	if v := tp.CheckServer("node.local"); v == nil || v.Rule != "require_valid_tls" {
		t.Fatalf("expected require_valid_tls violation for .local domain, got %v", v)
	}
}

func TestTrustPolicyMaxKeyAge(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:        true,
		MaxKeyAgeHours: 168, // 7 days
	})

	resp := &ServerKeysResponse{
		ServerName:   "test.server",
		ValidUntilTS: time.Now().Add(30 * 24 * time.Hour).UnixMilli(), // 30 days
	}

	if v := tp.CheckResponse("test.server", resp); v == nil {
		t.Error("should reject key with validity exceeding max")
	}

	resp.ValidUntilTS = time.Now().Add(24 * time.Hour).UnixMilli() // 1 day
	if v := tp.CheckResponse("test.server", resp); v != nil {
		t.Error("should allow key with acceptable validity")
	}
}

func TestTrustPolicyRequireNotarySignatures(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:                 true,
		RequireNotarySignatures: 2,
	})

	resp := &ServerKeysResponse{
		ServerName: "test.server",
		Signatures: map[string]map[string]string{
			"test.server": {"ed25519:key": "sig"},
		},
	}

	if v := tp.CheckResponse("test.server", resp); v == nil {
		t.Error("should reject response with no notary signatures")
	}

	resp.Signatures["notary1.org"] = map[string]string{"ed25519:n1": "sig"}
	if v := tp.CheckResponse("test.server", resp); v == nil {
		t.Error("should reject response with only 1 notary signature")
	}

	resp.Signatures["notary2.org"] = map[string]string{"ed25519:n2": "sig"}
	if v := tp.CheckResponse("test.server", resp); v != nil {
		t.Error("should allow response with 2 notary signatures")
	}
}

func TestTrustPolicyReload(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:  true,
		DenyList: []string{"old.denied"},
	})

	if v := tp.CheckServer("old.denied"); v == nil {
		t.Error("should deny old.denied")
	}

	tp.Reload(TrustPolicyConfig{
		Enabled:  true,
		DenyList: []string{"new.denied"},
	})

	if v := tp.CheckServer("old.denied"); v != nil {
		t.Error("should allow old.denied after reload")
	}

	if v := tp.CheckServer("new.denied"); v == nil {
		t.Error("should deny new.denied after reload")
	}
}

func TestTrustPolicyStats(t *testing.T) {
	tp := NewTrustPolicy(TrustPolicyConfig{
		Enabled:                 true,
		DenyList:                []string{"a", "b", "c"},
		AllowList:               []string{"x", "y"},
		RequireNotarySignatures: 2,
		RequireWellKnown:        true,
		RequireValidTLS:         true,
		BlockPrivateIPs:         true,
	})

	stats := tp.Stats()

	if stats["enabled"] != true {
		t.Error("enabled should be true")
	}
	if stats["deny_list_count"] != 3 {
		t.Errorf("deny_list_count = %v, want 3", stats["deny_list_count"])
	}
	if stats["allow_list_count"] != 2 {
		t.Errorf("allow_list_count = %v, want 2", stats["allow_list_count"])
	}
	if stats["require_well_known"] != true {
		t.Errorf("require_well_known = %v, want true", stats["require_well_known"])
	}
	if stats["require_valid_tls"] != true {
		t.Errorf("require_valid_tls = %v, want true", stats["require_valid_tls"])
	}
}

func TestWildcardMatching(t *testing.T) {
	tests := []struct {
		pattern string
		input   string
		match   bool
	}{
		{"*", "anything", true},
		{"*.example.com", "sub.example.com", true},
		{"*.example.com", "example.com", false},
		{"example.*", "example.com", true},
		{"example.*", "example.org", true},
		{"*.spam.*", "foo.spam.bar", true},
		{"prefix*suffix", "prefixmiddlesuffix", true},
		{"exact", "exact", true},
		{"exact", "notexact", false},
	}

	for _, tt := range tests {
		result := matchWildcard(tt.pattern, tt.input)
		if result != tt.match {
			t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.pattern, tt.input, result, tt.match)
		}
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Errorf("failed to parse %s", tt.ip)
			continue
		}
		result := isPrivateIP(ip)
		if result != tt.private {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.private)
		}
	}
}

func TestAnalyticsRapidRotationUsesRealElapsedTime(t *testing.T) {
	a := NewAnalytics(nil, AnalyticsConfig{Enabled: true})
	server := "analytics.test"

	first := &ServerKeysResponse{
		ServerName:   server,
		ValidUntilTS: time.Now().Add(2 * time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:key1": {Key: "AQ"},
		},
		Signatures: map[string]map[string]string{
			server: {"ed25519:key1": "sig"},
		},
	}
	a.RecordKeyObservation(server, first)

	a.mu.Lock()
	a.stats.ServerStats[server].LastSeen = time.Now().Add(-48 * time.Hour)
	a.mu.Unlock()

	second := &ServerKeysResponse{
		ServerName:   server,
		ValidUntilTS: time.Now().Add(2 * time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:key2": {Key: "Ag"},
		},
		Signatures: map[string]map[string]string{
			server: {"ed25519:key2": "sig"},
		},
	}
	anomalies := a.RecordKeyObservation(server, second)

	for _, anomaly := range anomalies {
		if anomaly.Type == AnomalyRapidRotation {
			t.Fatalf("unexpected rapid rotation anomaly for 48h interval: %+v", anomaly)
		}
	}
}

func TestAnalyticsMultipleKeysIncrementsTotalAnomalies(t *testing.T) {
	a := NewAnalytics(nil, AnalyticsConfig{Enabled: true})
	server := "multi.test"

	resp := &ServerKeysResponse{
		ServerName:   server,
		ValidUntilTS: time.Now().Add(2 * time.Hour).UnixMilli(),
		VerifyKeys: map[string]VerifyKeyResponse{
			"ed25519:key1": {Key: "AQ"},
			"ed25519:key2": {Key: "Ag"},
		},
		Signatures: map[string]map[string]string{
			server: {"ed25519:key1": "sig"},
		},
	}

	a.RecordKeyObservation(server, resp)
	stats := a.GetStats()

	if stats.TotalAnomalies == 0 {
		t.Fatalf("expected TotalAnomalies to increment for multiple keys anomaly")
	}
}
