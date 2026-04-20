Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# ADR-0008: Schema Migrations

## Status

Accepted.

## Context

The earlier storage layer ran `CREATE TABLE IF NOT EXISTS` on
every startup from `internal/keys/storage.go`. That works only
while the schema is frozen. Three failure modes:

1. Silent skip on an incompatible existing table. `IF NOT EXISTS`
   matches by name, not by shape. Adding a column leaves an old
   operator database with the prior shape; new queries then fail
   at request time instead of at startup.
2. No rollback boundary. A half-applied schema cannot be
   distinguished from a fresh install.
3. No audit trail. Operators cannot tell which binary version
   last touched their database.

## Decision

`internal/storage/migrations` owns all DDL.

Architecture:

- Embedded `sql/NNNN_name.sql` files. Version NNNN is a positive
  integer; ordering is strictly ascending; duplicates are a hard
  error.
- `schema_migrations` bookkeeping table with
  `(version integer primary key, name text, applied_at timestamptz)`.
- `Apply(db *sql.DB) (applied int, err error)` creates the
  bookkeeping table when missing, reads the applied set, and runs
  each pending migration inside its own transaction. On error
  the migration rolls back and `Apply` returns; the database
  stays in the prior state.
- `server.New` calls `migrations.Apply` before any subsystem
  touches the database. A failure is fatal.

Migration authoring rules:

- A shipped migration is immutable. Corrections ship as a new
  numbered migration.
- `IF NOT EXISTS` is preferred so a re-applied migration is safe.
- `DROP` statements are not used in early migrations. Data loss
  requires an explicit operator runbook step.

Shipped migrations:

- `0001_initial.sql`: the historically-hard-coded `server_keys`
  and `server_key_responses` tables, verbatim, so existing
  operator databases converge to `version=1` on first upgrade.
- `0002_transparency_log.sql`: the transparency log table for
  the default configuration (`transparency.table_name =
  "key_transparency_log"`).

Operators who set `transparency.table_name` to a non-default
value continue to get lazy DDL via
`internal/keys/transparency.go`. That path logs a deprecation
warning and is documented as the single exception to the
"migrations own the schema" rule.

## Consequences

- Every operator database carries a stamped version for support
  and audit.
- Schema changes are reviewable as code, not ad-hoc `psql`
  patches.
- `internal/keys/storage.go` is a pure data-access layer.
- A failed migration halts startup with a precise error instead
  of letting the service run on drifted schema.

## Alternatives Considered

- Keep `CREATE IF NOT EXISTS`. Rejected for the reasons above.
- Adopt `golang-migrate` or `goose`. Rejected to preserve the
  zero-dependency policy for core packages (ADR-0002). The
  runner here is under 200 lines and has no transitive surface.
- Embed SQL in Go string literals. Rejected: `embed.FS` plus
  standalone `.sql` files is easier to diff in review.

## References

- `internal/storage/migrations/migrations.go`
- `internal/storage/migrations/sql/0001_initial.sql`
- `internal/storage/migrations/sql/0002_transparency_log.sql`
- `docs/runbook/schema-migration.md`
