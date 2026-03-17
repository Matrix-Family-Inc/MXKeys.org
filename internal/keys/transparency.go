/*
 * Project: MXKeys - Matrix Federation Trust Infrastructure
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 * Contact: @support:matrix.family
 *
 * Key Transparency Log
 * Append-only audit log for all observed server keys.
 * Tracks key appearances, rotations, and anomalies.
 */

package keys

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/metrics"
)

// LogEventType describes the type of transparency log event
type LogEventType string

const (
	EventKeyFirstSeen    LogEventType = "KEY_FIRST_SEEN"
	EventKeyVerified     LogEventType = "KEY_VERIFIED"
	EventKeyRotation     LogEventType = "KEY_ROTATION"
	EventKeyExpired      LogEventType = "KEY_EXPIRED"
	EventKeyRevoked      LogEventType = "KEY_REVOKED"
	EventAnomalyRapid    LogEventType = "ANOMALY_RAPID_ROTATION"
	EventAnomalyMulti    LogEventType = "ANOMALY_MULTIPLE_KEYS"
	EventAnomalyBackdate LogEventType = "ANOMALY_BACKDATED_KEY"
	EventFetchFailed     LogEventType = "FETCH_FAILED"
	EventPolicyViolation LogEventType = "POLICY_VIOLATION"
)

// TransparencyLogEntry represents a single log entry
type TransparencyLogEntry struct {
	ID           int64        `json:"id"`
	Timestamp    time.Time    `json:"timestamp"`
	ServerName   string       `json:"server_name"`
	KeyID        string       `json:"key_id,omitempty"`
	EventType    LogEventType `json:"event_type"`
	Details      string       `json:"details,omitempty"`
	KeyHash      string       `json:"key_hash,omitempty"`
	ValidUntilTS int64        `json:"valid_until_ts,omitempty"`
	PreviousHash string       `json:"previous_hash,omitempty"`
	EntryHash    string       `json:"entry_hash"`
}

// TransparencyLog manages the append-only key transparency log
type TransparencyLog struct {
	db            *sql.DB
	tableName     string
	enabled       bool
	logAllKeys    bool
	logKeyChanges bool
	logAnomalies  bool
	retentionDays int

	mu         sync.RWMutex
	lastHash   string
	keyHistory map[string]*keyHistoryEntry // serverName -> history

	// Metrics
	entriesTotal   *metrics.Counter
	anomaliesTotal *metrics.Counter
}

type keyHistoryEntry struct {
	lastKeyID     string
	lastKeyHash   string
	lastSeen      time.Time
	rotationCount int
	firstSeen     time.Time
}

// TransparencyConfig holds transparency log configuration
type TransparencyConfig struct {
	Enabled       bool
	LogAllKeys    bool
	LogKeyChanges bool
	LogAnomalies  bool
	RetentionDays int
	TableName     string
}

// NewTransparencyLog creates a new transparency log
func NewTransparencyLog(db *sql.DB, cfg TransparencyConfig) (*TransparencyLog, error) {
	tl := &TransparencyLog{
		db:            db,
		tableName:     cfg.TableName,
		enabled:       cfg.Enabled,
		logAllKeys:    cfg.LogAllKeys,
		logKeyChanges: cfg.LogKeyChanges,
		logAnomalies:  cfg.LogAnomalies,
		retentionDays: cfg.RetentionDays,
		keyHistory:    make(map[string]*keyHistoryEntry),
		entriesTotal: metrics.NewCounter(metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "transparency",
			Name:      "entries_total",
			Help:      "Total transparency log entries",
		}),
		anomaliesTotal: metrics.NewCounter(metrics.CounterOpts{
			Namespace: "mxkeys",
			Subsystem: "transparency",
			Name:      "anomalies_total",
			Help:      "Total anomalies detected",
		}),
	}

	if cfg.TableName == "" {
		tl.tableName = "key_transparency_log"
	}

	if cfg.Enabled {
		if err := tl.initTable(); err != nil {
			return nil, err
		}
		if err := tl.loadLastHash(); err != nil {
			return nil, err
		}
		log.Info("Transparency log initialized",
			"table", tl.tableName,
			"retention_days", cfg.RetentionDays,
		)
	}

	return tl, nil
}

// initTable creates the transparency log table
func (tl *TransparencyLog) initTable() error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			server_name TEXT NOT NULL,
			key_id TEXT,
			event_type TEXT NOT NULL,
			details TEXT,
			key_hash TEXT,
			valid_until_ts BIGINT,
			previous_hash TEXT,
			entry_hash TEXT NOT NULL,
			
			CONSTRAINT %s_entry_hash_unique UNIQUE (entry_hash)
		);
		
		CREATE INDEX IF NOT EXISTS %s_server_name_idx ON %s (server_name);
		CREATE INDEX IF NOT EXISTS %s_timestamp_idx ON %s (timestamp);
		CREATE INDEX IF NOT EXISTS %s_event_type_idx ON %s (event_type);
	`, tl.tableName, tl.tableName, tl.tableName, tl.tableName,
		tl.tableName, tl.tableName, tl.tableName, tl.tableName)

	_, err := tl.db.Exec(query)
	return err
}

// loadLastHash loads the most recent entry hash for chaining
func (tl *TransparencyLog) loadLastHash() error {
	query := fmt.Sprintf(`SELECT entry_hash FROM %s ORDER BY id DESC LIMIT 1`, tl.tableName)
	row := tl.db.QueryRow(query)

	var hash string
	err := row.Scan(&hash)
	if err == sql.ErrNoRows {
		tl.lastHash = "genesis"
		return nil
	}
	if err != nil {
		return err
	}

	tl.lastHash = hash
	return nil
}

// LogKey records a key observation
func (tl *TransparencyLog) LogKey(ctx context.Context, serverName string, resp *ServerKeysResponse) error {
	if !tl.enabled || !tl.logAllKeys {
		return nil
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	for keyID, verifyKey := range resp.VerifyKeys {
		keyHash := hashKey(verifyKey.Key)

		history, exists := tl.keyHistory[serverName]
		if !exists {
			// First time seeing this server
			history = &keyHistoryEntry{
				firstSeen: time.Now(),
			}
			tl.keyHistory[serverName] = history

			if err := tl.appendEntry(ctx, &TransparencyLogEntry{
				Timestamp:    time.Now(),
				ServerName:   serverName,
				KeyID:        keyID,
				EventType:    EventKeyFirstSeen,
				KeyHash:      keyHash,
				ValidUntilTS: resp.ValidUntilTS,
			}); err != nil {
				return err
			}
		} else if history.lastKeyHash != keyHash {
			// Key rotation detected
			if tl.logKeyChanges {
				if err := tl.appendEntry(ctx, &TransparencyLogEntry{
					Timestamp:    time.Now(),
					ServerName:   serverName,
					KeyID:        keyID,
					EventType:    EventKeyRotation,
					Details:      fmt.Sprintf("previous_key_id=%s", history.lastKeyID),
					KeyHash:      keyHash,
					ValidUntilTS: resp.ValidUntilTS,
				}); err != nil {
					return err
				}
			}

			// Check for anomalies
			if tl.logAnomalies {
				tl.checkAnomalies(ctx, serverName, keyID, history)
			}

			history.rotationCount++
		}

		// Update history
		history.lastKeyID = keyID
		history.lastKeyHash = keyHash
		history.lastSeen = time.Now()
	}

	return nil
}

// LogVerification records a successful key verification
func (tl *TransparencyLog) LogVerification(ctx context.Context, serverName, keyID string) error {
	if !tl.enabled || !tl.logAllKeys {
		return nil
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	return tl.appendEntry(ctx, &TransparencyLogEntry{
		Timestamp:  time.Now(),
		ServerName: serverName,
		KeyID:      keyID,
		EventType:  EventKeyVerified,
	})
}

// LogFailure records a fetch failure
func (tl *TransparencyLog) LogFailure(ctx context.Context, serverName string, reason string) error {
	if !tl.enabled {
		return nil
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	return tl.appendEntry(ctx, &TransparencyLogEntry{
		Timestamp:  time.Now(),
		ServerName: serverName,
		EventType:  EventFetchFailed,
		Details:    reason,
	})
}

// LogPolicyViolation records a trust policy violation
func (tl *TransparencyLog) LogPolicyViolation(ctx context.Context, violation *PolicyViolation) error {
	if !tl.enabled {
		return nil
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	return tl.appendEntry(ctx, &TransparencyLogEntry{
		Timestamp:  time.Now(),
		ServerName: violation.ServerName,
		EventType:  EventPolicyViolation,
		Details:    fmt.Sprintf("%s: %s", violation.Rule, violation.Details),
	})
}

// checkAnomalies detects suspicious key behavior
func (tl *TransparencyLog) checkAnomalies(ctx context.Context, serverName, keyID string, history *keyHistoryEntry) {
	now := time.Now()

	// Rapid rotation: more than 3 rotations in 24 hours
	if history.rotationCount > 3 && now.Sub(history.firstSeen) < 24*time.Hour {
		tl.anomaliesTotal.Inc()
		tl.appendEntry(ctx, &TransparencyLogEntry{
			Timestamp:  now,
			ServerName: serverName,
			KeyID:      keyID,
			EventType:  EventAnomalyRapid,
			Details:    fmt.Sprintf("rotations=%d in %v", history.rotationCount, now.Sub(history.firstSeen)),
		})
	}
}

// appendEntry adds a new entry to the log with hash chaining
func (tl *TransparencyLog) appendEntry(ctx context.Context, entry *TransparencyLogEntry) error {
	entry.PreviousHash = tl.lastHash
	entry.EntryHash = tl.computeEntryHash(entry)

	query := fmt.Sprintf(`
		INSERT INTO %s (timestamp, server_name, key_id, event_type, details, key_hash, valid_until_ts, previous_hash, entry_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, tl.tableName)

	_, err := tl.db.ExecContext(ctx, query,
		entry.Timestamp,
		entry.ServerName,
		entry.KeyID,
		entry.EventType,
		entry.Details,
		entry.KeyHash,
		entry.ValidUntilTS,
		entry.PreviousHash,
		entry.EntryHash,
	)

	if err != nil {
		return err
	}

	tl.lastHash = entry.EntryHash
	tl.entriesTotal.Inc()

	log.Debug("Transparency log entry",
		"server", entry.ServerName,
		"event", entry.EventType,
		"hash", entry.EntryHash[:16],
	)

	return nil
}

// computeEntryHash creates a SHA-256 hash of the entry for chain integrity
func (tl *TransparencyLog) computeEntryHash(entry *TransparencyLogEntry) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s",
		entry.Timestamp.Format(time.RFC3339Nano),
		entry.ServerName,
		entry.KeyID,
		entry.EventType,
		entry.Details,
		entry.KeyHash,
		entry.ValidUntilTS,
		entry.PreviousHash,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

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

// VerifyChain verifies the hash chain integrity
func (tl *TransparencyLog) VerifyChain(ctx context.Context, limit int) (bool, error) {
	if !tl.enabled {
		return true, nil
	}

	query := fmt.Sprintf(`
		SELECT timestamp, server_name, key_id, event_type, details, key_hash, valid_until_ts, previous_hash, entry_hash
		FROM %s
		ORDER BY id ASC
		LIMIT $1
	`, tl.tableName)

	rows, err := tl.db.QueryContext(ctx, query, limit)
	if err != nil {
		return false, err
	}
	defer rows.Close()

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
			return false, err
		}

		e.KeyID = keyID.String
		e.Details = details.String
		e.KeyHash = keyHash.String
		e.ValidUntilTS = validUntil.Int64
		e.PreviousHash = prevHash.String

		// Verify previous hash
		if e.PreviousHash != expectedPrevHash {
			log.Error("Chain verification failed",
				"expected_prev", expectedPrevHash,
				"got_prev", e.PreviousHash,
				"entry_hash", e.EntryHash,
			)
			return false, nil
		}

		// Verify entry hash
		computedHash := tl.computeEntryHash(&e)
		if computedHash != e.EntryHash {
			log.Error("Entry hash verification failed",
				"expected", computedHash,
				"got", e.EntryHash,
			)
			return false, nil
		}

		expectedPrevHash = e.EntryHash
	}

	return true, nil
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
	tl.mu.RUnlock()

	return map[string]interface{}{
		"enabled":         true,
		"total_entries":   totalEntries,
		"unique_servers":  uniqueServers,
		"oldest_entry":    oldestEntry,
		"newest_entry":    newestEntry,
		"last_hash":       hashPreview(lastHash),
		"tracked_servers": historySize,
	}, nil
}

// hashKey creates a SHA-256 hash of a key
func hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

func hashPreview(hash string) string {
	if hash == "" {
		return ""
	}
	if len(hash) <= 16 {
		return hash
	}
	return hash[:16] + "..."
}

// ExportJSON exports log entries as JSON
func (tl *TransparencyLog) ExportJSON(ctx context.Context, serverName string, since time.Time) ([]byte, error) {
	entries, err := tl.Query(ctx, serverName, since, 10000)
	if err != nil {
		return nil, err
	}
	return json.Marshal(entries)
}
