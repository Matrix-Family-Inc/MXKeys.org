Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# ADR-0008: Schema Migrations

## Status

Accepted

## Context

The pre-ADR-0008 storage layer ran `CREATE TABLE IF NOT EXISTS`
statements on every startup from `internal/keys/storage.go`. This
worked while the schema was frozen, but it has three failure modes
as MXKeys evolves:

1. Silent skip on existing incompatible tables. `IF NOT EXISTS`
   matches by name, not by shape. A version bump that adds a column
   leaves existing operator databases with the old shape; application
   queries against the new shape then error at request time rather
   than at startup.
2. No rollback guarantee. Without explicit bookkeeping, a half-
   applied schema cannot be distinguished from a fresh install.
3. No audit trail. Operators cannot tell which version of MXKeys
   last ran against their database, which is a common request during
   incident response.

## Decision

Introduce `internal/storage/migrations` as the sole owner of DDL.

Architecture:

- Embedded `sql/NNNN_name.sql` files. Version NNNN is a positive
  integer; ordering is strictly ascending; duplicates are a hard
  error.
- `schema_migrations` bookkeeping table: `(version integer primary
  key, name text, applied_at timestamptz)`.
- `Apply(db *sql.DB) (applied int, err error)` creates the
  bookkeeping table if missing, loads the set of applied versions,
  and runs each pending migration inside its own transaction. A
  failed migration rolls back and returns, leaving the database in
  its prior state.
- `server.New` invokes `migrations.Apply` before any dependent
  subsystem touches the database. A failure is fatal.

Migration authoring rules:

- A shipped migration is immutable. Corrections ship as a new
  numbered migration.
- `IF NOT EXISTS` is preferred for idempotence so re-applying a
  migration accidentally (e.g. concurrent deployments) is safe.
- No `DROP` statements in the first few migrations: data loss
  requires explicit operator runbook steps.

Initial migration (0001_initial.sql) contains the previously-hard-
coded `server_keys` and `server_key_responses` tables verbatim so
existing operator databases converge to "state = v0001" on first
upgrade.

## Consequences

- Every operator database carries a stamped version visible to
  support and audit.
- Schema changes are reviewable as code rather than ad-hoc
  `psql` patches.
- `internal/keys/storage.go` becomes a pure data-access layer with
  no DDL responsibility.
- A failed migration halts startup with a precise error instead of
  letting the service run on drifted schema.
- The transparency log still manages its own table creation because
  its table name is operator-configurable (`transparency.table_name`);
  that remains a small, documented exception until a generic
  parameterized-DDL mechanism is needed.

## Alternatives Considered

- Keep `CREATE IF NOT EXISTS`: rejected for the reasons above.
- Adopt `golang-migrate` or `goose`: rejected to preserve the
  zero-dependency policy for core packages (see ADR-0002). The
  migrations runner here is ~180 lines and carries no transitive
  surface area.
- Write SQL into Go string literals: rejected; `embed.FS` plus
  standalone `.sql` files keeps migrations diffable in reviews.

## References

- `internal/storage/migrations/migrations.go`
- `internal/storage/migrations/sql/0001_initial.sql`
- `docs/runbook/schema-migration.md`
