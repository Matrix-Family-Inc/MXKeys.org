Project: MXKeys (mxkeys.org)
Company: Matrix.Family Inc. (https://matrix.family)
Owner: Matrix.Family Inc.
Contact: dev@matrix.family
Support: support@matrix.family
Matrix: @support:matrix.family
Date: Sat 21 Nov 2026 01:41:00 UTC
Status: Updated

# Matrix v1.16 Clause Map (MXKeys Scope)

## Purpose

This document maps Matrix v1.16 requirements to:
- MXKeys implementation,
- test evidence,
- requirement level (`MUST/SHOULD`) within the applicable notary/key-server scope.

Read it with `docs/matrix-v1.16-conformance-matrix.md`, which records
the status of each row.

## Scope

Only federation key-notary requirements are covered:
- version endpoint,
- server keys endpoints,
- key query endpoint,
- server discovery,
- canonical JSON/signature verification.

## Clause Mapping

| Clause ID | Matrix v1.16 Requirement (scope) | Level | MXKeys Implementation | Tests/Evidence |
|---|---|---|---|---|
| C-001 | Server-Server API: `GET /_matrix/federation/v1/version` | MUST | `internal/server/handlers_probes.go` `handleVersion()` | `internal/server/handlers_test.go` |
| C-002 | Server Keys API: `GET /_matrix/key/v2/server` | MUST | `internal/server/handlers_keys.go` `handleServerKeys()`, `internal/keys/notary.go` `GetOwnKeys()` | `internal/server/handlers_test.go` + live |
| C-003 | Server Keys API: `GET /_matrix/key/v2/server/{key_id}` | SHOULD | `handleServerKeys()` path value validation | `internal/server/handlers_test.go` |
| C-004 | Server Keys API: `POST /_matrix/key/v2/query` request/response envelope | MUST | `handleKeyQuery()` + `Notary.QueryKeys()` | `internal/server/handlers_test.go`, live tests |
| C-005 | Query request: reject malformed JSON | MUST | `decodeStrictJSON()`, `writeMatrixError()` | `TestDecodeStrictJSONTrailingData`, live strictness |
| C-006 | Query request: enforce input constraints (`server_name`, `key_id`, criteria) | MUST | `validateKeyQueryServerKeys()` + validators | `TestValidateKeyQueryServerKeys*` |
| C-007 | Query response: partial failures via `failures` map | MUST | `Notary.QueryKeys()` failure population | `TestLiveNotaryFailurePath` |
| C-008 | Canonical JSON for signing/verification | MUST | `internal/zero/canonical/json.go` | `internal/zero/canonical/json_test.go` |
| C-009 | Signature verification: Ed25519 self-signature required | MUST | `internal/keys/server_keys_parse.go` `verifySelfSignedServerKeys()` | `internal/keys/fetcher_test.go` |
| C-010 | Perspective signature verification (pinned notary key) | SHOULD | `internal/keys/fetcher_signature.go` `verifyNotarySignature()` | `TestVerifyNotarySignatureAcceptsSpecSignature`, `TestVerifyNotarySignatureRejectsSignatureOverSignatures` |
| C-011 | Server discovery: `.well-known` delegation | SHOULD | `Resolver.resolveWellKnown()` | `internal/keys/resolver_test.go` |
| C-012 | Server discovery: SRV `_matrix-fed._tcp` / fallback behavior | SHOULD | `resolveSRV()`, `resolveSRVLegacy()`, fallback | `internal/keys/resolver_test.go` |
| C-013 | Robust handling for IPv6 literals/URLs | SHOULD | `ResolvedServer.URL()` via `net.JoinHostPort` | `TestResolvedServerURL` |
| C-014 | Validity checks on key responses (`valid_until_ts`, server match) | MUST | `internal/keys/fetcher_direct.go` `fetchDirect()`, then `verifySelfSignedServerKeys()` | `internal/keys/fetcher_test.go` |
| C-015 | Request size/resource safety | SHOULD | `http.MaxBytesReader`, `readLimitedBody()` | handlers/fetcher tests |

## Coverage

These rows are the notary scope of Matrix v1.16. The map does not cite
paragraph numbers inside the Matrix specification. How to run the tests
is in `docs/build.md`.
