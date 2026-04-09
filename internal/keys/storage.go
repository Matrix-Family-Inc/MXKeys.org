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

package keys

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Storage handles persistence for server keys
type Storage struct {
	db *sql.DB
}

const (
	storageWriteAttempts = 3
	storageRetryBackoff  = 100 * time.Millisecond
)

// StoredKey represents a stored server key
type StoredKey struct {
	ServerName  string
	KeyID       string
	PublicKey   []byte
	ValidUntil  time.Time
	FetchedAt   time.Time
	RawResponse []byte
}

// NewStorage creates new key storage
func NewStorage(db *sql.DB) (*Storage, error) {
	s := &Storage{db: db}
	if err := s.createTables(); err != nil {
		return nil, err
	}
	return s, nil
}

// createTables creates required tables
func (s *Storage) createTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS server_keys (
			server_name TEXT NOT NULL,
			key_id TEXT NOT NULL,
			public_key BYTEA NOT NULL,
			valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
			fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			raw_response JSONB,
			PRIMARY KEY (server_name, key_id)
		)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_server_keys_server ON server_keys(server_name)
	`)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_server_keys_valid ON server_keys(valid_until)
	`)
	if err != nil {
		return err
	}

	// Table for caching full server key responses
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS server_key_responses (
			server_name TEXT PRIMARY KEY,
			response JSONB NOT NULL,
			valid_until TIMESTAMP WITH TIME ZONE NOT NULL,
			fetched_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	return err
}

func isRetryableStorageError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "timeout") ||
		strings.Contains(errText, "temporarily unavailable") ||
		strings.Contains(errText, "connection reset") ||
		strings.Contains(errText, "connection refused") ||
		strings.Contains(errText, "broken pipe") ||
		strings.Contains(errText, "driver: bad connection")
}

func (s *Storage) execWrite(query string, args ...interface{}) error {
	var lastErr error
	for attempt := 0; attempt < storageWriteAttempts; attempt++ {
		if _, err := s.db.Exec(query, args...); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if !isRetryableStorageError(lastErr) || attempt == storageWriteAttempts-1 {
			break
		}
		time.Sleep(storageRetryBackoff * time.Duration(1<<attempt))
	}
	return lastErr
}

// StoreKey stores a server key
func (s *Storage) StoreKey(serverName, keyID string, publicKey []byte, validUntil time.Time) error {
	return s.execWrite(`
		INSERT INTO server_keys (server_name, key_id, public_key, valid_until, fetched_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (server_name, key_id) DO UPDATE SET
			public_key = $3,
			valid_until = $4,
			fetched_at = NOW()
	`, serverName, keyID, publicKey, validUntil)
}

// StoreServerResponse stores full server key response
func (s *Storage) StoreServerResponse(serverName string, response *ServerKeysResponse, validUntil time.Time) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return err
	}

	return s.execWrite(`
		INSERT INTO server_key_responses (server_name, response, valid_until, fetched_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (server_name) DO UPDATE SET
			response = $2,
			valid_until = $3,
			fetched_at = NOW()
	`, serverName, responseJSON, validUntil)
}

// GetServerResponse retrieves cached server key response
func (s *Storage) GetServerResponse(serverName string) (*ServerKeysResponse, error) {
	var responseJSON []byte
	var validUntil time.Time

	err := s.db.QueryRow(`
		SELECT response, valid_until
		FROM server_key_responses
		WHERE server_name = $1 AND valid_until > NOW()
	`, serverName).Scan(&responseJSON, &validUntil)

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

	return &response, nil
}

// GetKey retrieves a server key
func (s *Storage) GetKey(serverName, keyID string) (*StoredKey, error) {
	key := &StoredKey{
		ServerName: serverName,
		KeyID:      keyID,
	}

	err := s.db.QueryRow(`
		SELECT public_key, valid_until, fetched_at
		FROM server_keys
		WHERE server_name = $1 AND key_id = $2
	`, serverName, keyID).Scan(&key.PublicKey, &key.ValidUntil, &key.FetchedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return key, nil
}

// GetValidKey retrieves a valid (non-expired) server key
func (s *Storage) GetValidKey(serverName, keyID string) (*StoredKey, error) {
	key := &StoredKey{
		ServerName: serverName,
		KeyID:      keyID,
	}

	err := s.db.QueryRow(`
		SELECT public_key, valid_until, fetched_at
		FROM server_keys
		WHERE server_name = $1 AND key_id = $2 AND valid_until > NOW()
	`, serverName, keyID).Scan(&key.PublicKey, &key.ValidUntil, &key.FetchedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return key, nil
}

// GetAllKeysForServer retrieves all keys for a server
func (s *Storage) GetAllKeysForServer(serverName string) ([]*StoredKey, error) {
	rows, err := s.db.Query(`
		SELECT key_id, public_key, valid_until, fetched_at
		FROM server_keys
		WHERE server_name = $1
	`, serverName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*StoredKey
	for rows.Next() {
		key := &StoredKey{ServerName: serverName}
		if err := rows.Scan(&key.KeyID, &key.PublicKey, &key.ValidUntil, &key.FetchedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	return keys, rows.Err()
}

// DeleteExpiredKeys removes expired keys
func (s *Storage) DeleteExpiredKeys() (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM server_keys WHERE valid_until < NOW()
	`)
	if err != nil {
		return 0, err
	}

	// Also clean up expired responses
	if _, err := s.db.Exec(`DELETE FROM server_key_responses WHERE valid_until < NOW()`); err != nil {
		return 0, fmt.Errorf("failed to delete expired responses: %w", err)
	}

	return result.RowsAffected()
}

// GetKnownServers returns list of all known servers
func (s *Storage) GetKnownServers() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT server_name FROM server_keys
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []string
	for rows.Next() {
		var serverName string
		if err := rows.Scan(&serverName); err != nil {
			return nil, err
		}
		servers = append(servers, serverName)
	}

	return servers, rows.Err()
}
