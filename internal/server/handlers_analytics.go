/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"
	"sort"
	"strconv"

	"mxkeys/internal/keys"
)

// handleAnalyticsSummary returns analytics summary.
// GET /_mxkeys/analytics/summary
func (s *Server) handleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.analytics == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	writeJSON(w, s.analytics.Summary())
}

// handleAnalyticsServers returns per-server analytics.
// GET /_mxkeys/analytics/servers?limit=50
func (s *Server) handleAnalyticsServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.analytics == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 500 {
			limit = 50
		}
	}

	stats := s.analytics.GetStats()

	// Convert to slice and sort by LastSeen descending.
	servers := make([]interface{}, 0, len(stats.ServerStats))
	ordered := make([]*keys.ServerAnalytics, 0, len(stats.ServerStats))
	for _, serverStats := range stats.ServerStats {
		ordered = append(ordered, serverStats)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].LastSeen.After(ordered[j].LastSeen)
	})
	for _, serverStats := range ordered {
		servers = append(servers, serverStats)
	}

	if len(servers) > limit {
		servers = servers[:limit]
	}

	writeJSON(w, map[string]interface{}{
		"servers": servers,
		"count":   len(servers),
		"total":   len(stats.ServerStats),
	})
}

// handleAnalyticsAnomalies returns detected anomalies.
// GET /_mxkeys/analytics/anomalies
func (s *Server) handleAnalyticsAnomalies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.analytics == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	anomalous := s.analytics.GetAnomalousServers()

	writeJSON(w, map[string]interface{}{
		"servers": anomalous,
		"count":   len(anomalous),
	})
}

// handleAnalyticsTopRotators returns servers with most key rotations.
// GET /_mxkeys/analytics/rotators?limit=10
func (s *Server) handleAnalyticsTopRotators(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.analytics == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 100 {
			limit = 10
		}
	}

	rotators := s.analytics.GetTopRotators(limit)

	writeJSON(w, map[string]interface{}{
		"servers": rotators,
		"count":   len(rotators),
	})
}
