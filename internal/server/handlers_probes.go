/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

package server

import (
	"context"
	"net/http"
	"time"

	"mxkeys/internal/version"
)

// handleHealth is the legacy health endpoint, kept for backwards compatibility.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"status":  "healthy",
		"server":  s.config.Server.Name,
		"version": version.Version,
	})
}

// handleLiveness is the liveness probe: does the process respond at all?
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]string{"status": "alive"})
}

// handleReadiness is the readiness probe: can the service accept traffic?
//
// Checks, in order:
//  1. shuttingDown flag, set at the start of graceful shutdown so that
//     orchestrators (Kubernetes, haproxy, an external LB) can drop the
//     instance from rotation BEFORE in-flight requests are drained. This
//     avoids the "50x burst during rolling restart" class of incidents.
//  2. database ping with 2 s timeout.
//  3. signing key loaded.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.shuttingDown.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, map[string]interface{}{
			"status": "draining",
			"error":  "shutdown in progress",
		})
		return
	}

	pingCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(pingCtx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, map[string]interface{}{
			"status": "not_ready",
			"error":  "database unavailable",
		})
		return
	}

	if s.notary.GetServerKeyID() == "" {
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, map[string]interface{}{
			"status": "not_ready",
			"error":  "signing key not loaded",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]interface{}{
		"status":  "ready",
		"server":  s.config.Server.Name,
		"version": version.Version,
	})
}

// handleStatus returns a detailed status snapshot for operator diagnostics.
// Aggregates database stats, cache sizes, cluster state, transparency stats,
// and trust policy stats when the corresponding subsystems are enabled.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dbStats := s.db.Stats()
	memoryCacheSize := s.notary.GetCacheSize()

	var dbCacheCount int
	row := s.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM server_keys")
	dbCacheErr := row.Scan(&dbCacheCount)

	status := map[string]interface{}{
		"status":  "ok",
		"version": version.Version,
		"server":  s.config.Server.Name,
		"uptime":  time.Since(s.startTime).String(),
		"cache": map[string]interface{}{
			"memory_entries":   memoryCacheSize,
			"database_entries": dbCacheCount,
		},
		"database": map[string]interface{}{
			"open_connections": dbStats.OpenConnections,
			"in_use":           dbStats.InUse,
			"idle":             dbStats.Idle,
			"max_open":         dbStats.MaxOpenConnections,
		},
	}
	if dbCacheErr != nil {
		status["database_entries_error"] = "query failed"
	}
	if s.cluster != nil {
		status["cluster"] = s.cluster.Stats()
	}
	if s.transparency != nil {
		if stats, err := s.transparency.Stats(r.Context()); err == nil {
			status["transparency"] = stats
		}
	}
	if s.trustPolicy != nil {
		status["trust_policy"] = s.trustPolicy.Stats()
	}

	writeJSON(w, status)
}

// handleVersion handles GET /_matrix/federation/v1/version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", version.Full())

	writeJSON(w, map[string]interface{}{
		"server": map[string]interface{}{
			"name":    version.Name,
			"version": version.Version,
		},
	})
}

// handleNotaryPublicKey returns the notary's public key for external STH verification.
// GET /_mxkeys/notary/key
func (s *Server) handleNotaryPublicKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	writeJSON(w, s.notary.GetPublicKeyInfo())
}
