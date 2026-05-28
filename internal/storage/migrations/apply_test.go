/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

package migrations

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"testing"
)

// fakeDriver is the minimum database/sql driver surface the migrations
// runner needs. It records executed SQL so tests can assert the DDL
// sequence without a real PostgreSQL.
type fakeDriver struct{ conn *fakeConn }

func (f *fakeDriver) Open(string) (driver.Conn, error) {
	if f.conn == nil {
		f.conn = newFakeConn()
	}
	return f.conn, nil
}

type fakeConn struct {
	execLog []string
	// failOn is the substring of a statement that, when matched, makes
	// Exec return an error. Used to exercise the transaction rollback
	// path.
	failOn string
	// appliedVersions is the set of rows the bookkeeping table would
	// return on SELECT.
	appliedVersions []int
}

func newFakeConn() *fakeConn { return &fakeConn{} }

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeStmt{conn: c, query: query}, nil
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { return &fakeTx{conn: c}, nil }

type fakeTx struct{ conn *fakeConn }

func (t *fakeTx) Commit() error   { return nil }
func (t *fakeTx) Rollback() error { return nil }

type fakeStmt struct {
	conn  *fakeConn
	query string
}

func (s *fakeStmt) Close() error  { return nil }
func (s *fakeStmt) NumInput() int { return -1 }

func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.conn.failOn != "" && containsSubstr(s.query, s.conn.failOn) {
		return nil, errors.New("synthetic exec failure")
	}
	s.conn.execLog = append(s.conn.execLog, s.query)
	return driver.RowsAffected(0), nil
}

func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	// Only the SELECT on schema_migrations is issued. Return the
	// applied version list.
	return &fakeRows{versions: s.conn.appliedVersions}, nil
}

type fakeRows struct {
	versions []int
	i        int
}

func (r *fakeRows) Columns() []string { return []string{"version"} }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.i >= len(r.versions) {
		return io.EOF
	}
	dest[0] = int64(r.versions[r.i])
	r.i++
	return nil
}

func containsSubstr(haystack, needle string) bool {
	return needle != "" && len(needle) <= len(haystack) &&
		(indexString(haystack, needle) >= 0)
}
func indexString(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// sharedDriver is the single fakeDriver instance registered with
// database/sql. Its backing connection is swapped per test through
// the conn pointer; sql.Register refuses duplicate names so we register
// exactly once.
var sharedDriver = &fakeDriver{}

func init() {
	sql.Register("fakepg", sharedDriver)
}

func openFakeDB(t *testing.T, conn *fakeConn) *sql.DB {
	t.Helper()
	sharedDriver.conn = conn
	db, err := sql.Open("fakepg", "fake")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}

func TestApplyAppliesEveryPendingMigration(t *testing.T) {
	conn := newFakeConn()
	db := openFakeDB(t, conn)
	defer db.Close()

	count, err := Apply(db)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least one migration applied, got %d", count)
	}
	// bookkeeping table creation + each migration body + insert row
	if len(conn.execLog) < count*2+1 {
		t.Fatalf("exec log too short: %d statements", len(conn.execLog))
	}
}

func TestApplyRejectsNilDB(t *testing.T) {
	if _, err := Apply(nil); err == nil {
		t.Fatal("expected error on nil db")
	}
}

func TestApplySkipsAlreadyAppliedVersions(t *testing.T) {
	conn := newFakeConn()
	// Pretend every embedded migration is already applied.
	ms, err := load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, m := range ms {
		conn.appliedVersions = append(conn.appliedVersions, m.version)
	}
	db := openFakeDB(t, conn)
	defer db.Close()

	count, err := Apply(db)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 new migrations, got %d", count)
	}
}

func TestApplyPropagatesBodyFailure(t *testing.T) {
	conn := newFakeConn()
	conn.failOn = "CREATE TABLE IF NOT EXISTS server_keys"
	db := openFakeDB(t, conn)
	defer db.Close()

	_, err := Apply(db)
	if err == nil {
		t.Fatal("expected migration body failure to propagate")
	}
	// Error message should mention the offending version.
	if !containsSubstr(fmt.Sprintf("%v", err), "0001") {
		t.Fatalf("error should mention version 0001, got %v", err)
	}
}
