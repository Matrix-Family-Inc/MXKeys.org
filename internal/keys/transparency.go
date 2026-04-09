/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Mar 16 2026 UTC
 * Status: Created
 */

package keys

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"mxkeys/internal/zero/log"
	"mxkeys/internal/zero/merkle"
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
	merkleTree *merkle.Tree

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
		merkleTree:    merkle.New(),
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
		if err := tl.rebuildMerkleTree(context.Background()); err != nil {
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
