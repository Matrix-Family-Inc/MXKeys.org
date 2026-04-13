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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"mxkeys/internal/zero/log"
)

// LogKey records a key observation
func (tl *TransparencyLog) LogKey(ctx context.Context, serverName string, resp *ServerKeysResponse) error {
	if !tl.enabled || !tl.logAllKeys {
		return nil
	}

	tl.mu.Lock()
	defer tl.mu.Unlock()

	sortedKeys := make([]string, 0, len(resp.VerifyKeys))
	for keyID := range resp.VerifyKeys {
		sortedKeys = append(sortedKeys, keyID)
	}
	sort.Strings(sortedKeys)

	for _, keyID := range sortedKeys {
		verifyKey := resp.VerifyKeys[keyID]
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
		if err := tl.appendEntry(ctx, &TransparencyLogEntry{
			Timestamp:  now,
			ServerName: serverName,
			KeyID:      keyID,
			EventType:  EventAnomalyRapid,
			Details:    fmt.Sprintf("rotations=%d in %v", history.rotationCount, now.Sub(history.firstSeen)),
		}); err != nil {
			log.Warn("Failed to append anomaly entry", "server", serverName, "error", err)
		}
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
	if err := tl.addMerkleHash(entry.EntryHash); err != nil {
		return err
	}
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
