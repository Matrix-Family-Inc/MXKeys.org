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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mxkeys/internal/keys"
	"mxkeys/internal/zero/log"
)

// writeJSON encodes v as JSON without HTML escaping. Errors are logged at debug
// level since response writes cannot be meaningfully recovered here.
func writeJSON(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		log.Debug("Failed to write JSON response", "error", err)
	}
}

// writeMatrixError writes a Matrix-style error body with errcode and error.
// If a request_id header is set, it is included in the body for trace correlation.
func writeMatrixError(w http.ResponseWriter, status int, errCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]string{
		"errcode": errCode,
		"error":   message,
	}
	if reqID := w.Header().Get("X-Request-ID"); reqID != "" {
		resp["request_id"] = reqID
	}
	writeJSON(w, resp)
}

func (s *Server) serverNameValidationLimit() int {
	if s == nil || s.config == nil || s.config.Security.MaxServerNameLength <= 0 {
		return maxServerNameLength
	}
	return s.config.Security.MaxServerNameLength
}

// validateKeyQueryServerKeys checks the shape of the server_keys map of a
// POST /_matrix/key/v2/query body before it is handed to the notary.
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
