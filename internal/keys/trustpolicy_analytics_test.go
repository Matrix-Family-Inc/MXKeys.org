/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"testing"
	"time"
)

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
