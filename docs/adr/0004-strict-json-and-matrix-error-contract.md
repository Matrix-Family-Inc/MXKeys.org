Project: MXKeys
Company: Matrix.Family Inc. - Delaware C-Corp
Dev: Brabus
Date: Mon Mar 16 2026 UTC
Status: Created
Contact: @support:matrix.family

# ADR-0004: Strict JSON Validation and Matrix Error Contract

## Status

Accepted

## Context

Federation trust APIs require deterministic request validation and interoperable error reporting.
Lenient JSON parsing and inconsistent error shapes increase ambiguity and interoperability risk.

## Decision

Enforce strict JSON decoding semantics and matrix-compatible error response shape for key-query and key endpoints.

Decision details:

- reject malformed JSON and trailing payload,
- enforce size limits and parameter validation,
- return matrix-compatible error envelope with stable `errcode` semantics.

## Consequences

Positive:

- deterministic and testable request validation behavior,
- clearer client-side handling of failures,
- reduced parser-confusion and malformed payload risk.

Trade-offs:

- stricter behavior may reject previously tolerated malformed client requests.

## Alternatives Considered

- permissive JSON decoding,
- generic HTTP error strings without matrix-compatible `errcode` mapping.

## References

- `internal/server/handlers.go`
- `internal/server/handlers_matrix_errors_test.go`
- `docs/federation-behavior.md`
