/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Wed Apr 08 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/merkle"
)

// Query returns log entries matching criteria
func (tl *TransparencyLog) Query(ctx context.Context, serverName string, since time.Time, limit int) ([]TransparencyLogEntry, error) {
	if !tl.enabled {
		return nil, nil
	}

	query := fmt.Sprintf(`
		SELECT id, timestamp, server_name, key_id, event_type, details, key_hash, valid_until_ts, previous_hash, entry_hash
		FROM %s
		WHERE ($1 = '' OR server_name = $1)
		AND timestamp >= $2
		ORDER BY id DESC
		LIMIT $3
	`, tl.tableName)

	rows, err := tl.db.QueryContext(ctx, query, serverName, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []TransparencyLogEntry
	for rows.Next() {
		var e TransparencyLogEntry
		var keyID, details, keyHash, prevHash sql.NullString
		var validUntil sql.NullInt64

		err := rows.Scan(
			&e.ID, &e.Timestamp, &e.ServerName, &keyID, &e.EventType,
			&details, &keyHash, &validUntil, &prevHash, &e.EntryHash,
		)
		if err != nil {
			return nil, err
		}

		e.KeyID = keyID.String
		e.Details = details.String
		e.KeyHash = keyHash.String
		e.ValidUntilTS = validUntil.Int64
		e.PreviousHash = prevHash.String

		entries = append(entries, e)
	}

	return entries, nil
}

// GetProof returns a Merkle inclusion proof for a log entry.
func (tl *TransparencyLog) GetProof(index int) (*merkle.Proof, error) {
	if !tl.enabled {
		return nil, fmt.Errorf("transparency log is disabled")
	}

	tl.mu.RLock()
	defer tl.mu.RUnlock()

	return tl.merkleTree.GetProof(index)
}

// VerifyChain verifies the hash chain integrity and returns (valid, checked_count, error)
func (tl *TransparencyLog) VerifyChain(ctx context.Context, limit int) (bool, int, error) {
	if !tl.enabled {
		return true, 0, nil
	}

	query := fmt.Sprintf(`
		SELECT timestamp, server_name, key_id, event_type, details, key_hash, valid_until_ts, previous_hash, entry_hash
		FROM %s
		ORDER BY id ASC
		LIMIT $1
	`, tl.tableName)

	rows, err := tl.db.QueryContext(ctx, query, limit)
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()

	checked := 0
	expectedPrevHash := "genesis"
	for rows.Next() {
		var e TransparencyLogEntry
		var keyID, details, keyHash, prevHash sql.NullString
		var validUntil sql.NullInt64

		err := rows.Scan(
			&e.Timestamp, &e.ServerName, &keyID, &e.EventType,
			&details, &keyHash, &validUntil, &prevHash, &e.EntryHash,
		)
		if err != nil {
			return false, checked, err
		}

		e.KeyID = keyID.String
		e.Details = details.String
		e.KeyHash = keyHash.String
		e.ValidUntilTS = validUntil.Int64
		e.PreviousHash = prevHash.String
		checked++

		if e.PreviousHash != expectedPrevHash {
			log.Error("Chain verification failed",
				"expected_prev", expectedPrevHash,
				"got_prev", e.PreviousHash,
				"entry_hash", e.EntryHash,
			)
			return false, checked, nil
		}

		computedHash := tl.computeEntryHash(&e)
		if computedHash != e.EntryHash {
			log.Error("Entry hash verification failed",
				"expected", computedHash,
				"got", e.EntryHash,
			)
			return false, checked, nil
		}

		expectedPrevHash = e.EntryHash
	}

	return true, checked, nil
}

// Cleanup removes entries older than retention period
func (tl *TransparencyLog) Cleanup(ctx context.Context) (int64, error) {
	if !tl.enabled || tl.retentionDays <= 0 {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -tl.retentionDays)

	query := fmt.Sprintf(`DELETE FROM %s WHERE timestamp < $1`, tl.tableName)
	result, err := tl.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	deleted, _ := result.RowsAffected()
	if deleted > 0 {
		log.Info("Transparency log cleanup",
			"deleted", deleted,
			"cutoff", cutoff,
		)
		tl.mu.Lock()
		tl.merkleTree = merkle.New()
		if err := tl.rebuildMerkleTree(ctx); err != nil {
			log.Warn("Failed to rebuild merkle tree after cleanup", "error", err)
		}
		if err := tl.loadLastHash(); err != nil {
			log.Warn("Failed to reload last hash after cleanup", "error", err)
		}
		for server, history := range tl.keyHistory {
			if history.lastSeen.Before(cutoff) {
				delete(tl.keyHistory, server)
			}
		}
		tl.mu.Unlock()
	}

	return deleted, nil
}

// Stats returns log statistics
func (tl *TransparencyLog) Stats(ctx context.Context) (map[string]interface{}, error) {
	if !tl.enabled {
		return map[string]interface{}{"enabled": false}, nil
	}

	var totalEntries, uniqueServers int64
	var oldestEntry, newestEntry time.Time

	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT server_name) as servers,
			MIN(timestamp) as oldest,
			MAX(timestamp) as newest
		FROM %s
	`, tl.tableName)

	row := tl.db.QueryRowContext(ctx, query)
	var oldest, newest sql.NullTime
	err := row.Scan(&totalEntries, &uniqueServers, &oldest, &newest)
	if err != nil {
		return nil, err
	}

	if oldest.Valid {
		oldestEntry = oldest.Time
	}
	if newest.Valid {
		newestEntry = newest.Time
	}

	tl.mu.RLock()
	lastHash := tl.lastHash
	historySize := len(tl.keyHistory)
	merkleRoot := tl.merkleTree.RootHex()
	merkleSize := tl.merkleTree.Size()
	tl.mu.RUnlock()

	return map[string]interface{}{
		"enabled":         true,
		"total_entries":   totalEntries,
		"unique_servers":  uniqueServers,
		"oldest_entry":    oldestEntry,
		"newest_entry":    newestEntry,
		"last_hash":       hashPreview(lastHash),
		"tracked_servers": historySize,
		"merkle_root":     merkleRoot,
		"merkle_size":     merkleSize,
	}, nil
}

// ExportJSON exports log entries as JSON
func (tl *TransparencyLog) ExportJSON(ctx context.Context, serverName string, since time.Time) ([]byte, error) {
	entries, err := tl.Query(ctx, serverName, since, 10000)
	if err != nil {
		return nil, err
	}
	return json.Marshal(entries)
}
