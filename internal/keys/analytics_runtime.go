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
	"fmt"
	"time"
)

// RecordKeyObservation records a key observation for analytics
func (a *Analytics) RecordKeyObservation(serverName string, resp *ServerKeysResponse) []Anomaly {
	if !a.enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	var anomalies []Anomaly

	// Get or create server stats
	serverStats, exists := a.stats.ServerStats[serverName]
	if !exists {
		serverStats = &ServerAnalytics{
			ServerName: serverName,
			FirstSeen:  now,
		}
		a.stats.ServerStats[serverName] = serverStats
		a.stats.TotalServersObserved++
		a.serversObserved.Set(float64(a.stats.TotalServersObserved))
	}

	previousSeen := serverStats.LastSeen
	serverStats.LastSeen = now
	serverStats.LastFetchStatus = "success"
	elapsedSinceLastSeen := time.Duration(0)
	if !previousSeen.IsZero() {
		elapsedSinceLastSeen = now.Sub(previousSeen)
	}

	// Analyze each key
	for keyID, verifyKey := range resp.VerifyKeys {
		_ = verifyKey // Used for key analysis

		// Check for key rotation
		if serverStats.CurrentKeyID != "" && serverStats.CurrentKeyID != keyID {
			a.stats.TotalKeyRotations++
			serverStats.RotationCount++
			a.keyRotationsTotal.Inc()

			// Check rotation frequency
			if elapsedSinceLastSeen < 24*time.Hour {
				anomaly := Anomaly{
					Type:       AnomalyRapidRotation,
					ServerName: serverName,
					KeyID:      keyID,
					Severity:   "medium",
					Details:    fmt.Sprintf("Key rotated after only %v", elapsedSinceLastSeen),
					DetectedAt: now,
				}
				anomalies = append(anomalies, anomaly)
				serverStats.AnomalyCount++
				a.stats.TotalAnomalies++
				a.anomaliesDetected.Inc()
			}
		}

		serverStats.CurrentKeyID = keyID
		serverStats.KeyCount++
		a.stats.TotalKeysObserved++
	}

	// Analyze key validity
	validUntil := time.UnixMilli(resp.ValidUntilTS)
	validity := validUntil.Sub(now)

	// Update validity statistics
	if a.stats.ShortestKeyValidity == 0 || validity < a.stats.ShortestKeyValidity {
		a.stats.ShortestKeyValidity = validity
	}
	if validity > a.stats.LongestKeyValidity {
		a.stats.LongestKeyValidity = validity
	}

	serverStats.AvgKeyValidity = validity
	serverStats.CurrentKeyAge = elapsedSinceLastSeen

	// Check for anomalous validity periods
	if validity < time.Hour {
		anomaly := Anomaly{
			Type:       AnomalyShortValidity,
			ServerName: serverName,
			Severity:   "high",
			Details:    fmt.Sprintf("Key validity only %v", validity),
			DetectedAt: now,
		}
		anomalies = append(anomalies, anomaly)
		serverStats.AnomalyCount++
		a.stats.TotalAnomalies++
		a.anomaliesDetected.Inc()
	}

	if validity > 365*24*time.Hour {
		anomaly := Anomaly{
			Type:       AnomalyLongValidity,
			ServerName: serverName,
			Severity:   "low",
			Details:    fmt.Sprintf("Key validity %v (over 1 year)", validity),
			DetectedAt: now,
		}
		anomalies = append(anomalies, anomaly)
		serverStats.AnomalyCount++
		a.stats.TotalAnomalies++
	}

	// Check for multiple active keys
	if len(resp.VerifyKeys) > 1 {
		anomaly := Anomaly{
			Type:       AnomalyMultipleKeys,
			ServerName: serverName,
			Severity:   "low",
			Details:    fmt.Sprintf("Server has %d active keys", len(resp.VerifyKeys)),
			DetectedAt: now,
		}
		anomalies = append(anomalies, anomaly)
		serverStats.AnomalyCount++
		a.stats.TotalAnomalies++
	}

	// Check for self-signature
	if _, hasSelfSig := resp.Signatures[serverName]; !hasSelfSig {
		anomaly := Anomaly{
			Type:       AnomalySignatureMissing,
			ServerName: serverName,
			Severity:   "critical",
			Details:    "Response missing self-signature",
			DetectedAt: now,
		}
		anomalies = append(anomalies, anomaly)
		serverStats.AnomalyCount++
		a.stats.TotalAnomalies++
		a.anomaliesDetected.Inc()
	}

	// Update most active server
	if serverStats.RotationCount > a.stats.MostActiveRotations {
		a.stats.MostActiveServer = serverName
		a.stats.MostActiveRotations = serverStats.RotationCount
	}

	a.stats.LastUpdated = now

	return anomalies
}

// RecordFetchFailure records a fetch failure
func (a *Analytics) RecordFetchFailure(serverName string, reason string) {
	if !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	serverStats, exists := a.stats.ServerStats[serverName]
	if !exists {
		serverStats = &ServerAnalytics{
			ServerName: serverName,
			FirstSeen:  time.Now(),
		}
		a.stats.ServerStats[serverName] = serverStats
	}

	serverStats.LastSeen = time.Now()
	serverStats.LastFetchStatus = "failed: " + reason
}
