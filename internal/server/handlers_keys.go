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
	"net/http"

	"mxkeys/internal/version"
	"mxkeys/internal/zero/log"
)

// handleServerKeys handles both GET /_matrix/key/v2/server
// and GET /_matrix/key/v2/server/{keyID}. When a specific keyID is requested
// and does not match the notary's own key, M_NOT_FOUND is returned.
func (s *Server) handleServerKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Server", version.Full())

	requestedKeyID := r.PathValue("keyID")
	if requestedKeyID != "" {
		if err := ValidateKeyID(requestedKeyID); err != nil {
			RecordRequestRejection(RejectReasonInvalidKeyID)
			writeMatrixError(w, http.StatusBadRequest, "M_INVALID_PARAM", "Invalid key ID: "+err.Error())
			return
		}

		if requestedKeyID != s.notary.GetServerKeyID() {
			writeMatrixError(w, http.StatusNotFound, "M_NOT_FOUND", "Key not found")
			return
		}
	}

	response, err := s.notary.GetOwnKeys()
	if err != nil {
		log.Error("Failed to get own keys", "error", err)
		writeMatrixError(w, http.StatusInternalServerError, "M_UNKNOWN", "Internal server error")
		return
	}

	log.Debug("Serving own server keys",
		"key_id", s.notary.GetServerKeyID(),
		"keys", len(response.VerifyKeys),
		"requested", requestedKeyID,
	)

	writeJSON(w, response)
}
