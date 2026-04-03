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

package server

import (
	"net/http"
	"strconv"
	"time"
)

// handleTransparencyLog returns transparency log entries
// GET /_mxkeys/transparency/log?server=matrix.org&since=2026-01-01&limit=100
func (s *Server) handleTransparencyLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.transparency == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{
			"errcode": "M_NOT_FOUND",
			"error":   "Transparency log not enabled",
		})
		return
	}

	serverName := r.URL.Query().Get("server")
	sinceStr := r.URL.Query().Get("since")
	limitStr := r.URL.Query().Get("limit")

	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse("2006-01-02", sinceStr)
		if err != nil {
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				writeJSON(w, map[string]string{
					"errcode": "M_INVALID_PARAM",
					"error":   "Invalid since format. Use YYYY-MM-DD or RFC3339",
				})
				return
			}
		}
	} else {
		since = time.Now().AddDate(0, 0, -7) // Default: last 7 days
	}

	limit := 100
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > 1000 {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{
				"errcode": "M_INVALID_PARAM",
				"error":   "Invalid limit. Must be 1-1000",
			})
			return
		}
	}

	entries, err := s.transparency.Query(r.Context(), serverName, since, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to query transparency log",
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
		"query": map[string]interface{}{
			"server": serverName,
			"since":  since,
			"limit":  limit,
		},
	})
}

// handleTransparencyVerify verifies hash chain integrity
// GET /_mxkeys/transparency/verify?limit=1000
func (s *Server) handleTransparencyVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.transparency == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{
			"errcode": "M_NOT_FOUND",
			"error":   "Transparency log not enabled",
		})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10000
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10000
		}
	}

	valid, err := s.transparency.VerifyChain(r.Context(), limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{
			"errcode": "M_UNKNOWN",
			"error":   "Chain verification failed",
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"valid":           valid,
		"entries_checked": limit,
		"verified_at":     time.Now().Format(time.RFC3339),
	})
}

// handleTransparencyStats returns transparency log statistics
// GET /_mxkeys/transparency/stats
func (s *Server) handleTransparencyStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.transparency == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	stats, err := s.transparency.Stats(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		writeJSON(w, map[string]string{
			"errcode": "M_UNKNOWN",
			"error":   "Failed to get stats",
		})
		return
	}

	writeJSON(w, stats)
}

// handleTransparencyProof returns a Merkle proof for an entry
// GET /_mxkeys/transparency/proof?index=123
func (s *Server) handleTransparencyProof(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.merkleTree == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{
			"errcode": "M_NOT_FOUND",
			"error":   "Merkle tree not enabled",
		})
		return
	}

	indexStr := r.URL.Query().Get("index")
	if indexStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{
			"errcode": "M_INVALID_PARAM",
			"error":   "index parameter required",
		})
		return
	}

	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{
			"errcode": "M_INVALID_PARAM",
			"error":   "Invalid index",
		})
		return
	}

	proof, err := s.merkleTree.GetProof(index)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{
			"errcode": "M_NOT_FOUND",
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, proof)
}

// handleAnalyticsSummary returns analytics summary
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

// handleAnalyticsServers returns per-server analytics
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

	// Convert to slice and sort by LastSeen
	servers := make([]interface{}, 0, len(stats.ServerStats))
	for _, serverStats := range stats.ServerStats {
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

// handleAnalyticsAnomalies returns detected anomalies
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

// handleAnalyticsTopRotators returns servers with most key rotations
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

// handleClusterStatus returns cluster status
// GET /_mxkeys/cluster/status
func (s *Server) handleClusterStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.cluster == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	writeJSON(w, s.cluster.Stats())
}

// handleClusterNodes returns cluster nodes
// GET /_mxkeys/cluster/nodes
func (s *Server) handleClusterNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.cluster == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	nodes := s.cluster.Nodes()

	writeJSON(w, map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// handleTrustPolicyStatus returns trust policy status
// GET /_mxkeys/policy/status
func (s *Server) handleTrustPolicyStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.trustPolicy == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	writeJSON(w, s.trustPolicy.Stats())
}

// handleTrustPolicyCheck checks a server against trust policy
// GET /_mxkeys/policy/check?server=matrix.org
func (s *Server) handleTrustPolicyCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.trustPolicy == nil {
		writeJSON(w, map[string]interface{}{
			"enabled": false,
		})
		return
	}

	serverName := r.URL.Query().Get("server")
	if serverName == "" {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]string{
			"errcode": "M_INVALID_PARAM",
			"error":   "server parameter required",
		})
		return
	}

	violation := s.trustPolicy.CheckServer(serverName)
	if violation != nil {
		writeJSON(w, map[string]interface{}{
			"server":  serverName,
			"allowed": false,
			"violation": map[string]string{
				"rule":    violation.Rule,
				"details": violation.Details,
			},
		})
		return
	}

	writeJSON(w, map[string]interface{}{
		"server":  serverName,
		"allowed": true,
	})
}

// registerTransparencyRoutes registers transparency and analytics routes
func (s *Server) registerTransparencyRoutes() {
	// Transparency log
	s.mux.HandleFunc("GET /_mxkeys/transparency/log", s.handleTransparencyLog)
	s.mux.HandleFunc("GET /_mxkeys/transparency/verify", s.handleTransparencyVerify)
	s.mux.HandleFunc("GET /_mxkeys/transparency/stats", s.handleTransparencyStats)
	s.mux.HandleFunc("GET /_mxkeys/transparency/proof", s.handleTransparencyProof)

	// Analytics
	s.mux.HandleFunc("GET /_mxkeys/analytics/summary", s.handleAnalyticsSummary)
	s.mux.HandleFunc("GET /_mxkeys/analytics/servers", s.handleAnalyticsServers)
	s.mux.HandleFunc("GET /_mxkeys/analytics/anomalies", s.handleAnalyticsAnomalies)
	s.mux.HandleFunc("GET /_mxkeys/analytics/rotators", s.handleAnalyticsTopRotators)

	// Cluster
	s.mux.HandleFunc("GET /_mxkeys/cluster/status", s.handleClusterStatus)
	s.mux.HandleFunc("GET /_mxkeys/cluster/nodes", s.handleClusterNodes)

	// Trust policy
	s.mux.HandleFunc("GET /_mxkeys/policy/status", s.handleTrustPolicyStatus)
	s.mux.HandleFunc("GET /_mxkeys/policy/check", s.handleTrustPolicyCheck)
}
