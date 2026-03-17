/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 *
 * Federation Key Analytics
 * Metrics, statistics, and anomaly detection for observed keys.
 */

package keys

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/metrics"
)

// Analytics tracks federation key statistics and anomalies
type Analytics struct {
	db      *sql.DB
	enabled bool

	mu    sync.RWMutex
	stats *AnalyticsStats

	// Metrics
	serversObserved    *metrics.Gauge
	keyRotationsTotal  *metrics.Counter
	anomaliesDetected  *metrics.Counter
	avgKeyValidityDays *metrics.Gauge
}

// AnalyticsStats holds aggregated statistics
type AnalyticsStats struct {
	TotalServersObserved int64
	TotalKeysObserved    int64
	TotalKeyRotations    int64
	TotalAnomalies       int64
	AvgKeyValidityHours  float64
	ShortestKeyValidity  time.Duration
	LongestKeyValidity   time.Duration
	MostActiveServer     string
	MostActiveRotations  int
	LastUpdated          time.Time

	// Per-server stats
	ServerStats map[string]*ServerAnalytics
}

// ServerAnalytics holds per-server statistics
type ServerAnalytics struct {
	ServerName       string
	FirstSeen        time.Time
	LastSeen         time.Time
	KeyCount         int
	RotationCount    int
	AvgKeyValidity   time.Duration
	CurrentKeyID     string
	CurrentKeyAge    time.Duration
	AnomalyCount     int
	FetchSuccessRate float64
	LastFetchStatus  string
}

// AnomalyType describes detected anomalies
type AnomalyType string

const (
	AnomalyRapidRotation    AnomalyType = "rapid_rotation"
	AnomalyShortValidity    AnomalyType = "short_validity"
	AnomalyLongValidity     AnomalyType = "long_validity"
	AnomalyMultipleKeys     AnomalyType = "multiple_keys"
	AnomalyBackdatedKey     AnomalyType = "backdated_key"
	AnomalySignatureMissing AnomalyType = "signature_missing"
	AnomalyUnexpectedChange AnomalyType = "unexpected_change"
)

// Anomaly describes a detected anomaly
type Anomaly struct {
	Type       AnomalyType
	ServerName string
	KeyID      string
	Severity   string // "low", "medium", "high", "critical"
	Details    string
	DetectedAt time.Time
}

// AnalyticsConfig holds analytics configuration
type AnalyticsConfig struct {
	Enabled bool
}

// NewAnalytics creates a new analytics engine
func NewAnalytics(db *sql.DB, cfg AnalyticsConfig) *Analytics {
	a := &Analytics{
		db:      db,
		enabled: cfg.Enabled,
		stats: &AnalyticsStats{
			ServerStats: make(map[string]*ServerAnalytics),
		},
		serversObserved: metrics.NewGauge(metrics.GaugeOpts{
			Namespace: "mxkeys",
			Subsystem: "analytics",
			Name:      "servers_observed",
			Help:      "Total unique servers observed",
		}),
		keyRotationsTotal: metrics.NewCounter(metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "analytics",
			Name:      "key_rotations_total",
			Help:      "Total key rotations observed",
		}),
		anomaliesDetected: metrics.NewCounter(metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "analytics",
			Name:      "anomalies_detected_total",
			Help:      "Total anomalies detected",
		}),
		avgKeyValidityDays: metrics.NewGauge(metrics.GaugeOpts{
			Namespace: "mxkeys",
			Subsystem: "analytics",
			Name:      "avg_key_validity_days",
			Help:      "Average key validity in days",
		}),
	}

	if cfg.Enabled {
		log.Info("Analytics engine initialized")
	}

	return a
}

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

// GetStats returns current analytics statistics
func (a *Analytics) GetStats() *AnalyticsStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy
	stats := &AnalyticsStats{
		TotalServersObserved: a.stats.TotalServersObserved,
		TotalKeysObserved:    a.stats.TotalKeysObserved,
		TotalKeyRotations:    a.stats.TotalKeyRotations,
		TotalAnomalies:       a.stats.TotalAnomalies,
		AvgKeyValidityHours:  a.stats.AvgKeyValidityHours,
		ShortestKeyValidity:  a.stats.ShortestKeyValidity,
		LongestKeyValidity:   a.stats.LongestKeyValidity,
		MostActiveServer:     a.stats.MostActiveServer,
		MostActiveRotations:  a.stats.MostActiveRotations,
		LastUpdated:          a.stats.LastUpdated,
		ServerStats:          make(map[string]*ServerAnalytics),
	}

	for k, v := range a.stats.ServerStats {
		stats.ServerStats[k] = v
	}

	return stats
}

// GetServerStats returns statistics for a specific server
func (a *Analytics) GetServerStats(serverName string) *ServerAnalytics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.stats.ServerStats[serverName]
}

// GetTopRotators returns servers with most key rotations
func (a *Analytics) GetTopRotators(limit int) []*ServerAnalytics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Collect all servers
	servers := make([]*ServerAnalytics, 0, len(a.stats.ServerStats))
	for _, stats := range a.stats.ServerStats {
		servers = append(servers, stats)
	}

	// Sort by rotation count (simple bubble sort for small lists)
	for i := 0; i < len(servers)-1; i++ {
		for j := 0; j < len(servers)-i-1; j++ {
			if servers[j].RotationCount < servers[j+1].RotationCount {
				servers[j], servers[j+1] = servers[j+1], servers[j]
			}
		}
	}

	if limit > len(servers) {
		limit = len(servers)
	}

	return servers[:limit]
}

// GetAnomalousServers returns servers with detected anomalies
func (a *Analytics) GetAnomalousServers() []*ServerAnalytics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var anomalous []*ServerAnalytics
	for _, stats := range a.stats.ServerStats {
		if stats.AnomalyCount > 0 {
			anomalous = append(anomalous, stats)
		}
	}

	return anomalous
}

// ComputeAggregates updates aggregate statistics from database
func (a *Analytics) ComputeAggregates(ctx context.Context) error {
	if !a.enabled || a.db == nil {
		return nil
	}

	// Query aggregate stats from server_keys table
	var totalKeys, avgValidityMs int64
	row := a.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(DISTINCT server_name),
			COALESCE(AVG(valid_until_ts - EXTRACT(EPOCH FROM NOW()) * 1000), 0)
		FROM server_keys
		WHERE valid_until_ts > EXTRACT(EPOCH FROM NOW()) * 1000
	`)

	err := row.Scan(&totalKeys, &avgValidityMs)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	a.mu.Lock()
	a.stats.TotalKeysObserved = totalKeys
	a.stats.AvgKeyValidityHours = float64(avgValidityMs) / (1000 * 60 * 60)
	a.avgKeyValidityDays.Set(a.stats.AvgKeyValidityHours / 24)
	a.mu.Unlock()

	return nil
}

// Summary returns a JSON-serializable summary
func (a *Analytics) Summary() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return map[string]interface{}{
		"enabled":                a.enabled,
		"total_servers_observed": a.stats.TotalServersObserved,
		"total_keys_observed":    a.stats.TotalKeysObserved,
		"total_key_rotations":    a.stats.TotalKeyRotations,
		"total_anomalies":        a.stats.TotalAnomalies,
		"avg_key_validity_hours": a.stats.AvgKeyValidityHours,
		"shortest_validity":      a.stats.ShortestKeyValidity.String(),
		"longest_validity":       a.stats.LongestKeyValidity.String(),
		"most_active_server":     a.stats.MostActiveServer,
		"most_active_rotations":  a.stats.MostActiveRotations,
		"tracked_servers":        len(a.stats.ServerStats),
		"last_updated":           a.stats.LastUpdated,
	}
}
