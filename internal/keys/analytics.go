/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"database/sql"
	"sort"
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

	if limit <= 0 {
		return nil
	}

	servers := make([]*ServerAnalytics, 0, len(a.stats.ServerStats))
	for _, stats := range a.stats.ServerStats {
		servers = append(servers, stats)
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].RotationCount > servers[j].RotationCount
	})

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
	var totalServers, totalKeys int64
	var avgValidityMs float64
	row := a.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(DISTINCT server_name),
			COUNT(*),
			COALESCE(AVG(EXTRACT(EPOCH FROM (valid_until - NOW())) * 1000), 0)
		FROM server_keys
		WHERE valid_until > NOW()
	`)

	err := row.Scan(&totalServers, &totalKeys, &avgValidityMs)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	a.mu.Lock()
	a.stats.TotalServersObserved = totalServers
	a.stats.TotalKeysObserved = totalKeys
	a.stats.AvgKeyValidityHours = avgValidityMs / (1000 * 60 * 60)
	a.serversObserved.Set(float64(totalServers))
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
