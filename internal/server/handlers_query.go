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
	"errors"
	"fmt"
	"net/http"

	"mxkeys/internal/keys"
	"mxkeys/internal/version"
	"mxkeys/internal/zero/log"
)

const (
	maxRequestBodySize = 1 << 20 // 1 MiB
	maxServersPerQuery = 100
)

// handleKeyQuery handles POST /_matrix/key/v2/query.
// Enforces body size, JSON depth, max servers per request, and server name
// validation before delegating to notary.QueryKeys.
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
		writeMatrixError(w, http.StatusBadRequest, "M_BAD_JSON", "Invalid or malformed JSON in request body")
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
