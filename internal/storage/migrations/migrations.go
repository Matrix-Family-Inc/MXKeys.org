/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

// Package migrations applies versioned SQL migrations to the PostgreSQL
// backing store. Migrations are embedded SQL files named NNNN_description.sql
// sorted by numeric prefix. Applied versions are tracked in the
// schema_migrations table so migrations are idempotent across restarts.
package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"mxkeys/internal/zero/log"
)

//go:embed sql/*.sql
var sqlFS embed.FS

// schemaVersionTable is the bookkeeping table that records which migrations
// have been applied.
const schemaVersionTable = "schema_migrations"

// migration is a parsed migration file.
type migration struct {
	version int
	name    string
	body    string
}

// Apply ensures schema_migrations exists and applies all pending migrations in
// version order. Each migration runs inside its own transaction: failure rolls
// the migration back without corrupting bookkeeping.
// Returns the number of migrations applied in this call.
func Apply(db *sql.DB) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("migrations: nil db")
	}
	if err := ensureBookkeeping(db); err != nil {
		return 0, err
	}

	migrations, err := load()
	if err != nil {
		return 0, err
	}

	applied, err := loadApplied(db)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range migrations {
		if _, ok := applied[m.version]; ok {
			continue
		}
		if err := apply(db, m); err != nil {
			return count, fmt.Errorf("migrations: apply %04d_%s: %w", m.version, m.name, err)
		}
		log.Info("Applied schema migration", "version", m.version, "name", m.name)
		count++
	}
	return count, nil
}

// ensureBookkeeping creates the schema_migrations table if missing.
// The table is owned by this package; do not rename.
func ensureBookkeeping(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + schemaVersionTable + ` (
			version     INTEGER PRIMARY KEY,
			name        TEXT NOT NULL,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("migrations: create %s: %w", schemaVersionTable, err)
	}
	return nil
}

// loadApplied returns the set of already-applied migration versions.
func loadApplied(db *sql.DB) (map[int]struct{}, error) {
	rows, err := db.Query(`SELECT version FROM ` + schemaVersionTable)
	if err != nil {
		return nil, fmt.Errorf("migrations: read %s: %w", schemaVersionTable, err)
	}
	defer rows.Close()

	applied := make(map[int]struct{})
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("migrations: scan %s: %w", schemaVersionTable, err)
		}
		applied[v] = struct{}{}
	}
	return applied, rows.Err()
}

// load reads all embedded migrations and returns them in version order.
// Filenames follow NNNN_name.sql where NNNN is a zero-padded positive integer.
// Duplicate or unparsable filenames cause load to fail.
func load() ([]migration, error) {
	entries, err := sqlFS.ReadDir("sql")
	if err != nil {
		return nil, fmt.Errorf("migrations: read embed: %w", err)
	}

	var out []migration
	seen := make(map[int]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		m, err := parseName(e.Name())
		if err != nil {
			return nil, err
		}
		if other, dup := seen[m.version]; dup {
			return nil, fmt.Errorf("migrations: duplicate version %d: %s and %s", m.version, other, e.Name())
		}
		body, err := sqlFS.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("migrations: read %s: %w", e.Name(), err)
		}
		m.body = string(body)
		seen[m.version] = e.Name()
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

// parseName extracts (version, name) from a filename like "0001_initial.sql".
func parseName(filename string) (migration, error) {
	stem := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(stem, "_", 2)
	if len(parts) != 2 {
		return migration{}, fmt.Errorf("migrations: invalid filename %q (expected NNNN_name.sql)", filename)
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil || version <= 0 {
		return migration{}, fmt.Errorf("migrations: invalid version in %q", filename)
	}
	return migration{version: version, name: parts[1]}, nil
}

// apply runs a single migration in a transaction and records it in
// schema_migrations on success.
func apply(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	if _, err := tx.Exec(m.body); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec body: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO `+schemaVersionTable+` (version, name) VALUES ($1, $2)`, m.version, m.name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
