Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# Runbook: Schema Migration

## Scope

Operator procedure for PostgreSQL schema changes under MXKeys.
Covers the default startup path, a forced manual apply, and
rollback options. For the design see ADR-0008.

## When Migrations Run

- Every `mxkeys` startup invokes `migrations.Apply(db)` before any
  dependent subsystem touches the database.
- Unapplied migrations are executed in version order, each in its
  own transaction; `schema_migrations` is stamped on success.
- A failing migration rolls back its own transaction and aborts
  startup. The service does **not** come up with a partially-applied
  schema.

## Standard Upgrade Path

1. Take a logical backup:

   ```bash
   pg_dump mxkeys > /backup/mxkeys_$(date -u +%Y%m%dT%H%M%SZ).sql
   ```

2. Install the new MXKeys binary.

3. Restart the service:

   ```bash
   systemctl restart mxkeys
   journalctl -u mxkeys -n 50 -f
   ```

   Migrations that land produce log entries of the form:

   ```text
   Applied schema migration version=7 name=add_rotation_ts
   ```

4. Verify:

   ```bash
   psql -U mxkeys -d mxkeys -c 'SELECT version, name, applied_at FROM schema_migrations ORDER BY version;'
   curl -fsS https://notary.example.org/_mxkeys/ready
   ```

## Forced Manual Apply

When debugging or validating a migration offline (CI, staging):

```bash
# Temporarily point your local mxkeys at the target DB:
export MXKEYS_DATABASE_URL='postgres://...'
./mxkeys -config /tmp/migration-only.yaml &
PID=$!
sleep 2
kill "${PID}"
```

The short-lived start applies pending migrations and exits on
SIGTERM. Inspect `schema_migrations` afterwards to confirm the
version landed.

## Rollback Policy

- **There is no automated down-migration.** Forward-only migrations
  keep bookkeeping simple and eliminate a class of accidental data
  loss.
- To revert to an earlier schema:

  1. Take a fresh backup.
  2. Restore the last known-good dump:

     ```bash
     pg_restore -U mxkeys -d mxkeys /backup/mxkeys_YYYYMMDDTHHMMSSZ.sql
     ```

  3. Run the older MXKeys binary that targets that schema.

- **Never hand-edit `schema_migrations`.** Deleting a row does not
  reverse the DDL a migration performed; it will cause the next
  startup to attempt the same migration again and fail.

## Adding a New Migration (developer path)

1. Pick the next unused version number (existing: 0001; next would
   be 0002):

   ```bash
   ls internal/storage/migrations/sql
   ```

2. Create `internal/storage/migrations/sql/0002_add_rotation_ts.sql`.
   Use `IF NOT EXISTS` where possible:

   ```sql
   -- 0002_add_rotation_ts.sql
   ALTER TABLE server_keys ADD COLUMN IF NOT EXISTS rotation_ts timestamptz;
   CREATE INDEX IF NOT EXISTS idx_server_keys_rotation_ts ON server_keys(rotation_ts);
   ```

3. Add a test covering the migration in
   `internal/storage/migrations/migrations_test.go` (version
   ordering, filename parsing, and idempotence tests exist already;
   extend the load test if the new migration needs shape
   validation).

4. Bump the coverage-gate floor if the new code path brings the
   package under its threshold.

5. Ship. The first operator to upgrade applies the migration on
   their next restart.

## Recovery Scenarios

### Migration fails halfway

The migration is a transaction; PostgreSQL rolls it back. The
service refuses to start; `schema_migrations` does not contain the
failed version. Fix the migration, rebuild, restart.

### Migration file was deleted after application

If a migration previously applied is removed from the codebase,
`Apply` no longer sees it but `schema_migrations` still lists the
version. This is benign: `Apply` simply skips applied versions.
However, the team should restore the SQL file for audit purposes;
removing history makes future diffs misleading.

### Two migrations share a version number

`load()` detects duplicates at startup and fails fast with
`migrations: duplicate version N: <fileA> and <fileB>`. Pick a new
number for one of them and ship.
