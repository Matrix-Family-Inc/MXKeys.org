/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"mxkeys/internal/keys"
	"mxkeys/internal/version"
	"mxkeys/internal/zero/log"
)

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func writeMatrixError(w http.ResponseWriter, status int, errCode, message string) {
	w.WriteHeader(status)
	_ = writeJSON(w, map[string]string{
		"errcode": errCode,
		"error":   message,
	})
}

func validateKeyQueryServerKeys(serverKeys map[string]map[string]keys.KeyCriteria, maxNameLen int) error {
	for serverName, keyMap := range serverKeys {
		if err := ValidateServerName(serverName, maxNameLen); err != nil {
			return fmt.Errorf("invalid server name '%s': %w", serverName, err)
		}
		for keyID, criteria := range keyMap {
			if keyID == "" {
				return fmt.Errorf("invalid key ID for '%s': key ID is empty", serverName)
			}
			if err := ValidateKeyID(keyID); err != nil {
				return fmt.Errorf("invalid key ID '%s' for '%s': %w", keyID, serverName, err)
			}
			if criteria.MinimumValidUntilTS < 0 {
				return fmt.Errorf("invalid minimum_valid_until_ts for '%s': must be non-negative", serverName)
			}
		}
	}
	return nil
}

// handleHealth handles health check requests (backwards compatible)
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, map[string]interface{}{
		"status":  "healthy",
		"server":  s.config.Server.Name,
		"version": version.Version,
	})
}

// handleLiveness handles liveness probe (process is alive)
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]string{"status": "alive"})
}

// handleReadiness handles readiness probe (service is ready to accept traffic)
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	if err := s.db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, map[string]interface{}{
			"status": "not_ready",
			"error":  "database unavailable",
		})
		return
	}

	// Check that notary key is loaded
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

// handleStatus handles detailed status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get database stats
	dbStats := s.db.Stats()

	// Get cache stats from notary
	memoryCacheSize := s.notary.GetCacheSize()

	// Get DB cache count
	var dbCacheCount int
	row := s.db.QueryRow("SELECT COUNT(*) FROM server_keys")
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
		status["database_entries_error"] = dbCacheErr.Error()
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

// handleVersion handles /_matrix/federation/v1/version
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

// handleServerKeys handles GET /_matrix/key/v2/server and GET /_matrix/key/v2/server/{keyID}
func (s *Server) handleServerKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", version.Full())

	// Check if specific key is requested (Go 1.22+ path value)
	requestedKeyID := r.PathValue("keyID")
	if requestedKeyID != "" {
		// Validate key ID format
		if err := ValidateKeyID(requestedKeyID); err != nil {
			RecordRequestRejection(RejectReasonInvalidKeyID)
			writeMatrixError(w, http.StatusBadRequest, "M_INVALID_PARAM", "Invalid key ID: "+err.Error())
			return
		}

		// Check if we have this key
		if requestedKeyID != s.notary.GetServerKeyID() {
			writeMatrixError(w, http.StatusNotFound, "M_NOT_FOUND", "Key not found")
			return
		}
	}

	response := s.notary.GetOwnKeys()

	log.Debug("Serving own server keys",
		"key_id", s.notary.GetServerKeyID(),
		"keys", len(response.VerifyKeys),
		"requested", requestedKeyID,
	)

	writeJSON(w, response)
}

const (
	maxRequestBodySize = 1 << 20 // 1MB
	maxServersPerQuery = 100
)

// handleKeyQuery handles POST /_matrix/key/v2/query
// This is the main notary functionality - returns keys for other servers
func (s *Server) handleKeyQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", version.Full())

	requestID := GetRequestID(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	var request keys.KeyQueryRequest
	if err := decodeStrictJSON(r.Body, &request, s.config.Security.MaxJSONDepth); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			RecordRequestRejection(RejectReasonBodyTooLarge)
			RecordKeyQuery(QueryStatusFailure, 0)
			writeMatrixError(w, http.StatusRequestEntityTooLarge, "M_TOO_LARGE", "Request body too large")
			return
		}
		RecordRequestRejection(RejectReasonInvalidJSON)
		RecordKeyQuery(QueryStatusFailure, 0)
		writeMatrixError(w, http.StatusBadRequest, "M_BAD_JSON", "Invalid JSON: "+err.Error())
		return
	}

	if len(request.ServerKeys) == 0 {
		RecordRequestRejection(RejectReasonEmptyRequest)
		RecordKeyQuery(QueryStatusFailure, 0)
		writeMatrixError(w, http.StatusBadRequest, "M_BAD_JSON", "No servers specified in server_keys")
		return
	}

	maxServers := s.config.Security.MaxServersPerQuery
	if maxServers <= 0 {
		maxServers = maxServersPerQuery
	}

	if len(request.ServerKeys) > maxServers {
		RecordRequestRejection(RejectReasonTooManyServers)
		RecordKeyQuery(QueryStatusFailure, len(request.ServerKeys))
		writeMatrixError(w, http.StatusBadRequest, "M_BAD_JSON", fmt.Sprintf("Too many servers in request (max %d)", maxServers))
		return
	}

	// Validate all server names
	maxNameLen := s.config.Security.MaxServerNameLength
	if maxNameLen <= 0 {
		maxNameLen = 255
	}

	if err := validateKeyQueryServerKeys(request.ServerKeys, maxNameLen); err != nil {
		RecordRequestRejection(RejectReasonInvalidServerName)
		RecordKeyQuery(QueryStatusFailure, len(request.ServerKeys))
		writeMatrixError(w, http.StatusBadRequest, "M_INVALID_PARAM", err.Error())
		return
	}

	log.Info("Processing key query request",
		"servers_requested", len(request.ServerKeys),
		"request_id", requestID,
		"remote_addr", r.RemoteAddr,
	)

	response := s.notary.QueryKeys(r.Context(), &request)

	status := QueryStatusSuccess
	if len(response.ServerKeys) == 0 && len(response.Failures) > 0 {
		status = QueryStatusFailure
	}
	RecordKeyQuery(status, len(request.ServerKeys))

	log.Info("Key query completed",
		"servers_returned", len(response.ServerKeys),
		"failures", len(response.Failures),
		"request_id", requestID,
	)

	writeJSON(w, response)
}
