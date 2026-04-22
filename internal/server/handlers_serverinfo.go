/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

// HTTP handler for /_mxkeys/server-info. Rate-limited under the
// same bucket as /_matrix/key/v2/query so an anonymous visitor
// cannot weaponise the notary as a scanner proxy.

package server

import (
	"net/http"
	"strings"

	"mxkeys/internal/version"
	"mxkeys/internal/zero/log"
)

// handleServerInfo serves GET /_mxkeys/server-info?name=<host>.
// Returns ServerInfoResponse as JSON; the shape always populates
// ServerName and FetchedAt and includes whichever sub-sections
// succeeded within the request budget.
func (s *Server) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", version.Full())

	if s.serverInfo == nil {
		writeMatrixError(w, http.StatusServiceUnavailable, "M_NOT_FOUND",
			"server-info enrichment is not enabled on this notary")
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("name"))
	if raw == "" {
		writeMatrixError(w, http.StatusBadRequest, "M_MISSING_PARAM",
			"query parameter 'name' is required")
		return
	}

	maxNameLen := s.config.Security.MaxServerNameLength
	if maxNameLen <= 0 {
		maxNameLen = 255
	}
	if err := ValidateServerName(raw, maxNameLen); err != nil {
		writeMatrixError(w, http.StatusBadRequest, "M_INVALID_PARAM", err.Error())
		return
	}

	resp, err := s.serverInfo.Enrich(r.Context(), raw)
	if err != nil {
		log.Warn("server-info enrichment failed",
			"server", raw,
			"error", err,
		)
		writeMatrixError(w, http.StatusInternalServerError, "M_UNKNOWN",
			"server-info lookup failed")
		return
	}

	writeJSON(w, resp)
}
