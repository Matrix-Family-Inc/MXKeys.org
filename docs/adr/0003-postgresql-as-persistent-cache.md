Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Maintainer: Brabus
Role: Lead Architect
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon Mar 16 2026 UTC
Status: Created

# ADR-0003: PostgreSQL as Persistent Key Cache

## Status

Accepted

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

- `internal/keys/storage.go`
- `internal/keys/notary_query.go`
- `internal/server/handlers.go`
- `docs/deployment.md`
