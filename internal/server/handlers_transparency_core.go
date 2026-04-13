/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

package server

import (
	"net/http"
	"strconv"
	"time"
)

const maxTransparencyVerifyLimit = 10000

// handleTransparencyLog returns transparency log entries.
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

	if serverName != "" {
		if err := ValidateServerName(serverName, s.serverNameValidationLimit()); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{
				"errcode": "M_INVALID_PARAM",
				"error":   "Invalid server parameter: " + err.Error(),
			})
			return
		}
	}

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

// handleTransparencyVerify verifies hash chain integrity.
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
	limit := maxTransparencyVerifyLimit
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > maxTransparencyVerifyLimit {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{
				"errcode": "M_INVALID_PARAM",
				"error":   "Invalid limit. Must be 1-10000",
			})
			return
		}
	}

	valid, checked, err := s.transparency.VerifyChain(r.Context(), limit)
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
		"entries_checked": checked,
		"verified_at":     time.Now().Format(time.RFC3339),
	})
}

// handleTransparencyStats returns transparency log statistics.
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

// handleTransparencyProof returns a Merkle proof for an entry.
// GET /_mxkeys/transparency/proof?index=123
func (s *Server) handleTransparencyProof(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.transparency == nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{
			"errcode": "M_NOT_FOUND",
			"error":   "Transparency log not enabled",
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

	proof, err := s.transparency.GetProof(index)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]string{
			"errcode": "M_NOT_FOUND",
			"error":   "Proof not found",
		})
		return
	}

	writeJSON(w, proof)
}
