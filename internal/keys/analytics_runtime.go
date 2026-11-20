/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package keys

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// maxAnalyticsServers bounds per-server statistics; servers beyond it are
// not tracked.
const maxAnalyticsServers = 50000

// RecordKeyObservation records a served key document. Statistics change
// only when the document differs from the previous one seen for the server,
// so repeated queries for an unchanged document leave them as they are. A
// rotation is a change of the set of verify key IDs.
func (a *Analytics) RecordKeyObservation(serverName string, resp *ServerKeysResponse) []Anomaly {
	if !a.enabled || resp == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	stats, exists := a.stats.ServerStats[serverName]
	if !exists {
		if len(a.stats.ServerStats) >= maxAnalyticsServers {
			return nil
		}
		stats = &ServerAnalytics{ServerName: serverName, FirstSeen: now, KeysChangedAt: now}
		a.stats.ServerStats[serverName] = stats
		a.stats.TotalServersObserved++
		a.serversObserved.Set(float64(a.stats.TotalServersObserved))
	}
	stats.LastSeen = now
	stats.LastFetchStatus = "success"

	keyIDs := make([]string, 0, len(resp.VerifyKeys))
	for keyID := range resp.VerifyKeys {
		keyIDs = append(keyIDs, keyID)
	}
	sort.Strings(keyIDs)
	keySet := strings.Join(keyIDs, ",")
	version := keySet + "|" + strconv.FormatInt(resp.ValidUntilTS, 10)
	if version == stats.version {
		return nil
	}
	stats.version = version

	var anomalies []Anomaly
	anomaly := func(kind AnomalyType, severity, keyID, details string) {
		anomalies = append(anomalies, Anomaly{Type: kind, ServerName: serverName, KeyID: keyID, Severity: severity, Details: details, DetectedAt: now})
		stats.AnomalyCount++
		a.stats.TotalAnomalies++
		a.anomaliesDetected.Inc()
	}

	if keySet != stats.CurrentKeyID {
		if stats.CurrentKeyID != "" {
			a.stats.TotalKeyRotations++
			stats.RotationCount++
			a.keyRotationsTotal.Inc()
			if since := now.Sub(stats.KeysChangedAt); since < 24*time.Hour {
				anomaly(AnomalyRapidRotation, "medium", keySet, fmt.Sprintf("Key set changed %v after the previous change", since.Round(time.Second)))
			}
		}
		stats.CurrentKeyID = keySet
		stats.KeysChangedAt = now
		stats.KeyCount = len(keyIDs)
		if stats.seen == nil {
			stats.seen = make(map[string]struct{}, len(keyIDs))
		}
		for _, keyID := range keyIDs {
			if _, ok := stats.seen[keyID]; !ok {
				stats.seen[keyID] = struct{}{}
				a.stats.TotalKeysObserved++
			}
		}
		if len(keyIDs) > 1 {
			anomaly(AnomalyMultipleKeys, "low", "", fmt.Sprintf("Server has %d active keys", len(keyIDs)))
		}
	}

	validity := time.UnixMilli(resp.ValidUntilTS).Sub(now)
	if a.stats.ShortestKeyValidity == 0 || validity < a.stats.ShortestKeyValidity {
		a.stats.ShortestKeyValidity = validity
	}
	if validity > a.stats.LongestKeyValidity {
		a.stats.LongestKeyValidity = validity
	}
	stats.AvgKeyValidity = validity
	stats.CurrentKeyAge = now.Sub(stats.KeysChangedAt)

	if validity < time.Hour {
		anomaly(AnomalyShortValidity, "high", "", fmt.Sprintf("Key validity only %v", validity.Round(time.Second)))
	}
	if validity > 365*24*time.Hour {
		anomaly(AnomalyLongValidity, "low", "", fmt.Sprintf("Key validity %v (over 1 year)", validity.Round(time.Hour)))
	}
	if len(resp.Signatures[serverName]) == 0 {
		anomaly(AnomalySignatureMissing, "critical", "", "Response missing self-signature")
	}

	if stats.RotationCount > a.stats.MostActiveRotations {
		a.stats.MostActiveServer = serverName
		a.stats.MostActiveRotations = stats.RotationCount
	}
	a.stats.LastUpdated = now

	return anomalies
}

// RecordFetchFailure marks a failed lookup for a server that is already
// tracked. Unknown names are not added: they come from anonymous queries.
func (a *Analytics) RecordFetchFailure(serverName string) {
	if !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if stats, exists := a.stats.ServerStats[serverName]; exists {
		stats.LastSeen = time.Now()
		stats.LastFetchStatus = "failed"
	}
}
