/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package keys

import (
	"testing"
	"time"
)

func analyticsDocument(server string, validUntil time.Time, keyIDs ...string) *ServerKeysResponse {
	resp := &ServerKeysResponse{
		ServerName:   server,
		ValidUntilTS: validUntil.UnixMilli(),
		VerifyKeys:   map[string]VerifyKeyResponse{},
		Signatures:   map[string]map[string]string{server: {}},
	}
	for _, keyID := range keyIDs {
		resp.VerifyKeys[keyID] = VerifyKeyResponse{Key: "AQ"}
		resp.Signatures[server][keyID] = "sig"
	}
	return resp
}

func TestAnalyticsRapidRotationUsesTimeSincePreviousChange(t *testing.T) {
	a := NewAnalytics(AnalyticsConfig{Enabled: true})
	const server = "analytics.test"
	validUntil := time.Now().Add(2 * time.Hour)

	a.RecordKeyObservation(server, analyticsDocument(server, validUntil, "ed25519:key1"))
	a.mu.Lock()
	a.stats.ServerStats[server].KeysChangedAt = time.Now().Add(-48 * time.Hour)
	a.mu.Unlock()

	for _, anomaly := range a.RecordKeyObservation(server, analyticsDocument(server, validUntil, "ed25519:key2")) {
		if anomaly.Type == AnomalyRapidRotation {
			t.Fatalf("a change 48h after the previous one is not rapid: %+v", anomaly)
		}
	}
	if got := a.GetServerStats(server).RotationCount; got != 1 {
		t.Fatalf("RotationCount = %d, want 1", got)
	}
}

func TestAnalyticsIgnoresRepeatedObservationsOfOneDocument(t *testing.T) {
	a := NewAnalytics(AnalyticsConfig{Enabled: true})
	const server = "multi.test"
	doc := analyticsDocument(server, time.Now().Add(2*time.Hour), "ed25519:key1", "ed25519:key2")

	for i := 0; i < 5; i++ {
		a.RecordKeyObservation(server, doc)
	}
	stats := a.GetStats()
	if stats.TotalKeyRotations != 0 {
		t.Fatalf("an unchanged two-key document must not count as rotations, got %d", stats.TotalKeyRotations)
	}
	if stats.TotalAnomalies != 1 {
		t.Fatalf("the multiple-keys anomaly must be counted once per document, got %d", stats.TotalAnomalies)
	}
	if stats.TotalKeysObserved != 2 {
		t.Fatalf("TotalKeysObserved = %d, want 2", stats.TotalKeysObserved)
	}
}

func TestAnalyticsDoesNotTrackUnknownFailures(t *testing.T) {
	a := NewAnalytics(AnalyticsConfig{Enabled: true})
	for i := 0; i < 100; i++ {
		a.RecordFetchFailure("random-" + string(rune('a'+i%26)) + ".invalid")
	}
	if got := len(a.GetStats().ServerStats); got != 0 {
		t.Fatalf("failed lookups of unknown names must not be tracked, got %d entries", got)
	}
}

func TestAnalyticsSnapshotsAreDetached(t *testing.T) {
	a := NewAnalytics(AnalyticsConfig{Enabled: true})
	const server = "copy.test"
	a.RecordKeyObservation(server, analyticsDocument(server, time.Now().Add(time.Hour), "ed25519:a"))

	snapshot := a.GetStats().ServerStats[server]
	a.RecordKeyObservation(server, analyticsDocument(server, time.Now().Add(2*time.Hour), "ed25519:b"))
	if snapshot.CurrentKeyID != "ed25519:a" {
		t.Fatalf("a returned snapshot must not change with later observations")
	}
}
