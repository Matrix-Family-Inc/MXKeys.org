Project: MXKeys (mxkeys.org)
Company: Matrix Family Inc. (https://matrix.family)
Owner: Matrix Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Mon 22 Jun 2026 00:51:51 UTC
Status: Updated

# ADR-0004: Strict JSON Validation and Matrix Error Contract

## Status

Accepted

## Visibility

Public.

## Context

Federation trust APIs require deterministic request validation and interoperable error reporting.
Lenient JSON parsing and inconsistent error shapes increase ambiguity and interoperability risk.

## Decision

Enforce strict JSON decoding semantics and matrix-compatible error response shape for key-query and key endpoints.

- reject malformed JSON and trailing payload,
- enforce size limits and parameter validation,
- return matrix-compatible error envelopes with stable `errcode` semantics.

## Consequences

- deterministic and testable request validation behavior,
- clearer client-side handling of failures,
- reduced parser-confusion and malformed payload risk.
- stricter behavior may reject previously tolerated malformed client requests.

## Alternatives Considered

- permissive JSON decoding,
- generic HTTP error strings without matrix-compatible `errcode` mapping.

## References

- `internal/server/json_decode.go` - strict JSON decoder and payload checks.
- `internal/server/handlers.go` - Matrix-facing HTTP error mapping.
- `internal/server/handlers_matrix_errors_test.go` - regression coverage for
  Matrix-compatible error envelopes.
- `docs/federation-behavior.md` - external behavior contract for federation
  clients.
