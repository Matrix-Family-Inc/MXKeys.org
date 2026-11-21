/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Fri 20 Nov 2026 22:39:55 UTC
 * Status: Updated
 */

package keys

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// StoreServerResponse stores a verified key document. raw_response holds the
// delivered bytes, which are the only form ever served back; the response
// JSONB column keeps a parsed copy for operator queries.
func (s *Storage) StoreServerResponse(serverName string, response *ServerKeysResponse, validUntil time.Time) error {
	if len(response.Raw) == 0 {
		return fmt.Errorf("storage: key document for %s has no raw bytes", serverName)
	}
	responseJSON, err := json.Marshal(serverKeysResponseAlias{
		ServerName:    response.ServerName,
		VerifyKeys:    response.VerifyKeys,
		OldVerifyKeys: response.OldVerifyKeys,
		ValidUntilTS:  response.ValidUntilTS,
		Signatures:    response.Signatures,
	})
	if err != nil {
		return err
	}

	return s.execWrite(`
		INSERT INTO server_key_responses (server_name, response, raw_response, valid_until, fetched_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (server_name) DO UPDATE SET
			response = $2,
			raw_response = $3,
			valid_until = $4,
			fetched_at = NOW()
	`, serverName, responseJSON, response.Raw, validUntil)
}

// GetServerResponse returns the stored key document for serverName and when
// it was fetched. Expired documents are returned too; the caller decides
// whether one is acceptable. Rows written before raw bytes were stored are
// ignored, since their signatures cannot be served verifiably.
func (s *Storage) GetServerResponse(ctx context.Context, serverName string) (*ServerKeysResponse, time.Time, error) {
	var raw []byte
	var fetchedAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT raw_response, fetched_at
		FROM server_key_responses
		WHERE server_name = $1 AND raw_response IS NOT NULL
	`, serverName).Scan(&raw, &fetchedAt)
	if err == sql.ErrNoRows {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}

	response, _, err := parseServerKeys(raw)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("storage: stored document for %s: %w", serverName, err)
	}
	return response, fetchedAt, nil
}
