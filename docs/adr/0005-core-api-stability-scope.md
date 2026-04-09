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

# ADR-0005: Core API Stability Scope

## Status

Accepted

## Context

MXKeys exposes both federation-facing core API endpoints and internal/operational endpoints.
Without explicit scope, "API stability" can be interpreted too broadly and cause contract ambiguity.

## Decision

Define stability commitment scope for core Matrix key-notary API endpoints:

- `GET /_matrix/key/v2/server`
- `GET /_matrix/key/v2/server/{keyID}`
- `POST /_matrix/key/v2/query`

Related compatibility endpoint:

- `GET /_matrix/federation/v1/version`

Operational endpoints remain documented but are not part of the same strict compatibility promise.

## Consequences

- clear compatibility boundary for integrators,
- safer evolution of operational/admin surfaces,
- explicit change-discipline for contract-impacting modifications.
- requires changelog and conformance updates when core API semantics change.

## Alternatives Considered

- no explicit stability scope,
- full stability guarantee for all operational endpoints.

## References

- `docs/federation-behavior.md`
- `docs/matrix-v1.16-conformance-matrix.md`
