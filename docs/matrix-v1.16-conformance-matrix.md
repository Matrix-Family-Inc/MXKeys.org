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

## Purpose

This document captures MXKeys conformance to Matrix Specification v1.16 within the federation key-notary scope.

This document is a release-gate artifact:
- source of truth for compatibility claims,
- regression control point,
- basis for release `GO/NO-GO`.

Base specification: [Matrix Specification v1.16](https://github.com/easypro-tech/matrix-specification-v1.16).

## Scope

This matrix evaluates only functionality implemented in MXKeys:
- `GET /_matrix/federation/v1/version`
- `GET /_matrix/key/v2/server`
- `GET /_matrix/key/v2/server/{keyID}`
- `POST /_matrix/key/v2/query`
- server discovery (`.well-known`, SRV, fallback)
- canonical JSON and Ed25519 signature verification

Out of scope:
- room/event federation flows,
- client-server API,
- identity/push/application service API.

## Evaluation Method

Statuses:
- `PASS` - requirement implemented with code and test/evidence.
- `PARTIAL` - partially implemented with limitations or incomplete evidence.
- `FAIL` - not implemented.

Evidence:
- file/symbol references,
- unit/integration/live tests,
- CI/lint/vet/race results.

## Current Verification Result (Mon Mar 16 2026 UTC)

Automated checks:
- `go test -count=1 ./...` — PASS
- `go test -race -count=1 ./...` — PASS
- `go vet ./...` — PASS

Live interop:
- `TestLiveFederationQueryStrictness` — PASS
- `TestLiveQueryCompatibility` — PASS
- `TestLiveNotaryFailurePath` — PASS

## Conformance Summary

- `PASS`: 28
- `PARTIAL`: 0
- `FAIL`: 0

## Conformance Matrix

| ID | v1.16 Requirement (key-notary scope) | Status | Implementation | Evidence |
|---|---|---|---|---|
| M-001 | `/_matrix/federation/v1/version` is available and returns server info | PASS | `internal/server/handlers.go` `handleVersion()` | unit: `internal/server/handlers_test.go`, live curl |
| M-002 | `GET /_matrix/key/v2/server` returns server keys | PASS | `handleServerKeys()`, `Notary.GetOwnKeys()` | unit + live |
| M-003 | `GET /_matrix/key/v2/server/{keyID}` is supported | PASS | `handleServerKeys()` path value + `ValidateKeyID()` | unit |
| M-004 | Invalid `keyID` is rejected with `M_INVALID_PARAM` | PASS | `handleServerKeys()` | unit |
| M-005 | Unknown `keyID` is rejected with `M_NOT_FOUND` | PASS | `handleServerKeys()` | unit |
| M-006 | `POST /_matrix/key/v2/query` supports multiple servers | PASS | `handleKeyQuery()` + `Notary.QueryKeys()` | unit + live |
| M-007 | Partial errors are returned in `failures` | PASS | `Notary.QueryKeys()` populates `Failures` | live: `TestLiveNotaryFailurePath` |
| M-008 | Strict JSON decoding rejects trailing payload | PASS | `decodeStrictJSON()` | unit + live strictness |
| M-009 | Request body size limit is enforced | PASS | `http.MaxBytesReader` + `M_TOO_LARGE` | unit |
| M-010 | Upstream response body size limit is enforced | PASS | `readLimitedBody()` + `maxFederationBody` | unit |
| M-011 | `server_name` validation in query | PASS | `ValidateServerName()` + `validateKeyQueryServerKeys()` | unit |
| M-012 | `key_id` validation in query criteria map | PASS | `validateKeyQueryServerKeys()` + `ValidateKeyID()` | unit |
| M-013 | `minimum_valid_until_ts` validation (non-negative) | PASS | `validateKeyQueryServerKeys()` | unit |
| M-014 | Matrix server discovery `.well-known` | PASS | `Resolver.resolveWellKnown()` | unit resolver |
| M-015 | Matrix server discovery SRV + fallback | PASS | `resolveSRV`, `resolveSRVLegacy`, fallback `:8448` | unit resolver |
| M-016 | Correct IPv6 URL formatting (`[host]:port`) | PASS | `ResolvedServer.URL()` via `net.JoinHostPort` | unit resolver |
| M-017 | Self-signature verification is mandatory | PASS | `Fetcher.verifySelfSignature()` | unit fetcher |
| M-018 | Pinned notary signature verification is cryptographic | PASS | `Fetcher.verifyNotarySignature()` + `ed25519.Verify` | unit fetcher |
| M-019 | Canonical JSON deterministic | PASS | `internal/zero/canonical/json.go` | unit canonical |
| M-020 | Canonical JSON rejects non-integer numbers | PASS | canonical number guards | unit canonical |
| M-021 | Canonical JSON enforces integer range | PASS | safe integer range checks | unit canonical |
| M-022 | Elimination of map-iteration non-determinism in query responses | PASS | `sortedServerNames()` in `Notary.QueryKeys()` | unit notary |
| M-023 | Runtime error paths do not contain silent decode failures in critical flows | PASS | raft/cluster hardening | unit + code audit |
| M-024 | Live interop with two Synapse servers | PASS | live tests + test_servers environment | live test suite |
| M-025 | Complete Matrix `errcode` semantics coverage for edge-case branches | PASS | unified `writeMatrixError()`, strict JSON/decode/validation branches in `handleKeyQuery()` and `handleServerKeys()` | `internal/server/handlers_matrix_errors_test.go`, `internal/server/handlers_test.go` |
| M-026 | Formal tracing `spec clause -> test case` for RFC2119 MUST/SHOULD items | PASS | `docs/matrix-v1.16-clause-map.md` | clause map + test links |
| M-027 | IDNA/Unicode corner cases for server names aligned with spec/implementations | PASS | ASCII+punycode policy, strict hostname validation, IPv6 literal/port edge checks | `internal/server/validation.go`, `internal/server/validation_test.go`, `internal/server/handlers_test.go` |
| M-028 | Fallback/notary interop scenario as mandatory release blocker | PASS | `release-live-interop-gate.yml` | mandatory live stage |

## Detailed Limits and Open Items

At the current snapshot, there are no critical `PARTIAL/FAIL` items within the key-notary scope.

## Release Rule

Release level "100/100" is allowed only when:
- all critical items in this matrix are `PASS`,
- all internal release-checklist `MUST` items are `PASS`,
- all mandatory CI gates are green.
