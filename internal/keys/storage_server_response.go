/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Apr 21 2026 UTC
 * Status: Created
 */

// Raw-preserving server_key_responses persistence. Splitting this out
// of storage.go keeps the per-key CRUD surface separate from the
// whole-response cache surface and keeps file size under the 300-line
// house rule.

package keys

import (
	"database/sql"
	"encoding/json"
	"time"
)

// StoreServerResponse stores full server key response.
//
// Both the parsed struct (in the legacy `response` JSONB column) and
// the raw origin bytes (when known, in `raw_response` BYTEA) are
// persisted. The raw column is what later lets notary_query preserve
// origin self-signatures byte-for-byte; the struct column keeps the
// fast struct-based read path available for callers that only need
// parsed fields.
func (s *Storage) StoreServerResponse(serverName string, response *ServerKeysResponse, validUntil time.Time) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return err
	}
	var rawBytes []byte
	if len(response.Raw) > 0 {
		rawBytes = append([]byte(nil), response.Raw...)
	}

	return s.execWrite(`
		INSERT INTO server_key_responses (server_name, response, raw_response, valid_until, fetched_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (server_name) DO UPDATE SET
			response = $2,
			raw_response = $3,
			valid_until = $4,
			fetched_at = NOW()
	`, serverName, responseJSON, rawBytes, validUntil)
}

// GetServerResponse retrieves the cached server key response.
//
// When raw_response is present (written by post-0003 code or
// backfilled on a recent fetch) the returned struct carries Raw so
// downstream callers can preserve origin self-signature bytes. For
// legacy rows with a NULL raw_response the returned struct has
// Raw == nil, and callers fall back to the struct-based pipeline.
func (s *Storage) GetServerResponse(serverName string) (*ServerKeysResponse, error) {
	var responseJSON []byte
	var rawResponse sql.RawBytes
	var validUntil time.Time

	err := s.db.QueryRow(`
		SELECT response, raw_response, valid_until
		FROM server_key_responses
		WHERE server_name = $1 AND valid_until > NOW()
	`, serverName).Scan(&responseJSON, &rawResponse, &validUntil)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var response ServerKeysResponse
	if err := json.Unmarshal(responseJSON, &response); err != nil {
		return nil, err
	}
	if len(rawResponse) > 0 {
		response.Raw = append([]byte(nil), rawResponse...)
	}

	return &response, nil
}
