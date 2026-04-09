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

# MXKeys Matrix v1.16 Conformance Matrix

## Scope

This matrix tracks the Matrix v1.16 areas intentionally implemented by MXKeys:

- `GET /_matrix/federation/v1/version`
- `GET /_matrix/key/v2/server`
- `GET /_matrix/key/v2/server/{keyID}`
- `POST /_matrix/key/v2/query`
- server discovery (`.well-known`, SRV, fallback)
- canonical JSON and Ed25519 verification

Out of scope:

- room/event federation flows,
- client-server API,
- identity, push, and application service APIs.

## Interpretation

- `PASS`: implemented and covered by code plus evidence.
- `PARTIAL`: implemented with caveats or incomplete evidence.
- `FAIL`: intentionally absent or not yet implemented.

This file is the only maintained v1.16 conformance artifact in `docs/`.

## Matrix

| ID | Requirement | Status | Implementation | Evidence |
|---|---|---|---|---|
| M-001 | `/_matrix/federation/v1/version` is available | PASS | `internal/server/handlers.go` `handleVersion()` | `internal/server/handlers_test.go` |
| M-002 | `GET /_matrix/key/v2/server` returns own server keys | PASS | `handleServerKeys()`, `Notary.GetOwnKeys()` | `internal/server/handlers_test.go` |
| M-003 | `GET /_matrix/key/v2/server/{keyID}` validates and selects key IDs | PASS | `handleServerKeys()`, `ValidateKeyID()` | `internal/server/handlers_test.go` |
| M-004 | Invalid `keyID` returns `M_INVALID_PARAM` | PASS | `handleServerKeys()` | `internal/server/handlers_test.go` |
| M-005 | Unknown `keyID` returns `M_NOT_FOUND` | PASS | `handleServerKeys()` | `internal/server/handlers_test.go` |
| M-006 | `POST /_matrix/key/v2/query` supports multiple servers | PASS | `handleKeyQuery()`, `Notary.QueryKeys()` | `internal/server/handlers_test.go`, live tests |
| M-007 | Partial upstream failures are reported via `failures` | PASS | `Notary.QueryKeys()` | `internal/server/handlers_live_test.go` |
| M-008 | Strict JSON rejects trailing payload | PASS | `decodeStrictJSON()` | `internal/server/handlers_test.go` |
| M-009 | Request body size is bounded | PASS | `http.MaxBytesReader` | `internal/server/handlers_limits_test.go` |
| M-010 | Upstream response body size is bounded | PASS | `readLimitedBody()` | `internal/keys/fetcher_test.go` |
| M-011 | `server_name` validation is enforced | PASS | `ValidateServerName()` | `internal/server/handlers_test.go` |
| M-012 | `key_id` validation is enforced | PASS | `ValidateKeyID()` | `internal/server/handlers_test.go` |
| M-013 | `minimum_valid_until_ts` must be non-negative | PASS | `validateKeyQueryServerKeys()` | `internal/server/handlers_test.go` |
| M-014 | Resolver supports `.well-known` delegation | PASS | `internal/keys/resolver_wellknown.go` | `internal/keys/resolver_test.go` |
| M-015 | Resolver supports SRV plus fallback | PASS | `internal/keys/resolver_srv.go` | `internal/keys/resolver_test.go` |
| M-016 | IPv6 URLs are formatted correctly | PASS | `ResolvedServer.URL()` | `internal/keys/resolver_test.go` |
| M-017 | Self-signature verification is mandatory | PASS | `verifySelfSignature()` | `internal/keys/fetcher_test.go` |
| M-018 | Pinned notary signatures are verified cryptographically | PASS | `verifyNotarySignature()` | `internal/keys/fetcher_test.go` |
| M-019 | Canonical JSON is deterministic | PASS | `internal/zero/canonical/json.go` | `internal/zero/canonical/json_test.go` |
| M-020 | Canonical JSON rejects unsupported numeric shapes | PASS | `internal/zero/canonical/json.go` | `internal/zero/canonical/json_test.go` |
| M-021 | Query responses avoid map-iteration nondeterminism | PASS | `sortedServerNames()` | `internal/keys/notary_query.go` |
| M-022 | Error-code branches are covered for core API paths | PASS | `writeMatrixError()`, handlers | `internal/server/handlers_matrix_errors_test.go` |
| M-023 | Server-name edge cases use ASCII/punycode policy | PASS | `internal/server/validation.go` | `internal/server/validation_test.go` |
| M-024 | Live interoperability checks exist | PASS | `internal/server/handlers_live_test.go` | optional live gate |

## Verification Commands

```bash
go test ./...
go test -tags=integration ./tests/integration/...
go test -race ./...
go vet ./...
```
