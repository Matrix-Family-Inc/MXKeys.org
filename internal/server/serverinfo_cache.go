/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

// PostgreSQL-backed cache for /_mxkeys/server-info responses.
// Backed by the server_info_cache table from migration 0004.
// The schema enforces one row per server_name; writes are
// upserts; reads filter out expired rows so the handler never
// has to interpret a stale row.

package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ServerInfoCache persists lookup results across restarts. The
// handler is read-through: it returns a cached hit when fresh,
// recomputes and upserts otherwise.
type ServerInfoCache struct {
	db *sql.DB
}

// NewServerInfoCache wraps a *sql.DB. Does not create the
// schema; migration 0004 owns DDL.
func NewServerInfoCache(db *sql.DB) (*ServerInfoCache, error) {
	if db == nil {
		return nil, fmt.Errorf("server_info cache: nil db")
	}
	return &ServerInfoCache{db: db}, nil
}

// Get returns the cached ServerInfoResponse for serverName when
// expires_at is still in the future. A cache miss returns
// (nil, nil) so the caller can distinguish it from a DB error.
func (c *ServerInfoCache) Get(ctx context.Context, serverName string) (*ServerInfoResponse, error) {
	const query = `
		SELECT info
		  FROM server_info_cache
		 WHERE server_name = $1
		   AND expires_at > NOW()
	`
	var payload []byte
	err := c.db.QueryRowContext(ctx, query, serverName).Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("server_info cache get: %w", err)
	}
	var out ServerInfoResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, fmt.Errorf("server_info cache decode: %w", err)
	}
	return &out, nil
}

// Put upserts the response for serverName with the provided TTL.
// Zero or negative TTL is a programmer error (the caller must
// resolve the effective TTL via the config defaults before
// calling).
func (c *ServerInfoCache) Put(ctx context.Context, serverName string, resp *ServerInfoResponse, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("server_info cache put: ttl must be positive (got %s)", ttl)
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("server_info cache encode: %w", err)
	}
	_, err = c.db.ExecContext(ctx, `
		INSERT INTO server_info_cache (server_name, info, fetched_at, expires_at)
		VALUES ($1, $2, NOW(), NOW() + $3::interval)
		ON CONFLICT (server_name) DO UPDATE SET
			info       = EXCLUDED.info,
			fetched_at = EXCLUDED.fetched_at,
			expires_at = EXCLUDED.expires_at
	`, serverName, payload, fmt.Sprintf("%d seconds", int64(ttl.Seconds())))
	if err != nil {
		return fmt.Errorf("server_info cache put: %w", err)
	}
	return nil
}

// DeleteExpired trims rows whose expires_at has passed. Intended
// for the periodic cleanup goroutine; safe to run concurrently
// with Get/Put.
func (c *ServerInfoCache) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := c.db.ExecContext(ctx, `DELETE FROM server_info_cache WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("server_info cache cleanup: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}
