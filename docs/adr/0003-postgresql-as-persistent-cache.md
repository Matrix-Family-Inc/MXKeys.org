Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Mar 16 2026 UTC
Status: Created

# ADR-0003: PostgreSQL as Persistent Key Cache

## Status

Accepted

## Visibility

Public.

## Context

MXKeys needs durable key-response storage beyond process lifetime, with deterministic query behavior and operational observability.
A pure in-memory approach would lose state on restart and increase upstream fetch pressure.

## Decision

Use PostgreSQL as persistent cache/storage for verified federation key responses, with in-memory cache as fast-path layer.

## Consequences

- persistence across restarts and deploys,
- lower repeated upstream fetch load,
- improved operational introspection through SQL-backed data.
- database availability becomes part of readiness semantics,
- requires backup/restore and schema lifecycle operations.

## Alternatives Considered

- memory-only cache,
- embedded local storage,
- external distributed cache without relational persistence.

## References

- `internal/keys/storage.go` - PostgreSQL persistence layer for key responses.
- `internal/keys/notary_query.go` - query path that reads and writes the
  persistent cache.
- `internal/server/handlers.go` - HTTP handlers that expose cached notary
  responses.
- `docs/deployment.md` - operator database configuration guidance.

## Alternatives

None recorded at authoring time. Any future revision that modifies this decision must list the rejected options explicitly.
