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

# Federation Behavior Specification

## Purpose

This document specifies deterministic federation-key behavior for MXKeys.
It defines request/response semantics, validation rules, resolution order, and failure behavior.

## Supported Endpoints

### Matrix key API

- `GET /_matrix/key/v2/server`
- `GET /_matrix/key/v2/server/{keyID}`
- `POST /_matrix/key/v2/query`
- `GET /_matrix/federation/v1/version`

### Operational endpoints

- `GET /_mxkeys/health`
- `GET /_mxkeys/live`
- `GET /_mxkeys/ready`
- `GET /_mxkeys/status`
- `GET /_mxkeys/metrics`

## Endpoint Status and Error Codes

| Endpoint | Success Codes | Failure Codes / Behavior |
|---|---|---|
| `GET /_matrix/key/v2/server` | `200` | internal failures may produce `5xx` |
| `GET /_matrix/key/v2/server/{keyID}` | `200` | `400` (`M_INVALID_PARAM`) for invalid `keyID`, `404` (`M_NOT_FOUND`) for unknown key |
| `POST /_matrix/key/v2/query` | `200` | `400` (`M_BAD_JSON`, `M_INVALID_PARAM`), `413` (`M_TOO_LARGE`) |
| `GET /_matrix/federation/v1/version` | `200` | internal failures may produce `5xx` |
| `GET /_mxkeys/health` | `200` | internal failures may produce `5xx` |
| `GET /_mxkeys/live` | `200` | internal failures may produce `5xx` |
| `GET /_mxkeys/ready` | `200` | `503` when DB or signing-key readiness checks fail |
| `GET /_mxkeys/status` | `200` | internal failures may produce `5xx` |
| `GET /_mxkeys/metrics` | `200` | internal failures may produce `5xx` |

## Deterministic Request Handling

### `POST /_matrix/key/v2/query`

MXKeys MUST process requests in this order:

1. Enforce request body size limit.
2. Decode strict JSON and reject trailing payload.
3. Validate `server_keys` presence and server count limits.
4. Validate each `server_name`, `key_id`, and criteria fields.
5. Resolve/fetch/verify keys per server.
6. Return `server_keys` and `failures` envelope.

If validation fails, MXKeys MUST return matrix-compatible error shape:

```json
{"errcode":"<M_CODE>","error":"<message>"}
```

## Server Discovery Behavior

For remote server key fetches, resolver MUST follow this order:

1. `.well-known/matrix/server` delegation (when applicable),
2. SRV records,
3. explicit host/port rules,
4. direct connection fallback behavior defined by resolver policy.

Resolver behavior MUST remain deterministic for equivalent inputs.

## Signature and Payload Verification

For fetched server keys, MXKeys MUST:

- validate canonical JSON/signature-compatible payload structure,
- validate self-signatures cryptographically,
- validate server-name consistency,
- validate key and signature lengths,
- reject invalid material before cache/store/sign.

If configured, pinned fallback/notary signatures MUST be verified cryptographically.

## Caching Behavior

MXKeys uses memory and database caching.

Cache behavior requirements:

- only valid key material may be returned,
- expired cryptographic material MUST NOT be served as valid,
- stale fallbacks are allowed only under explicit logic constraints,
- cleanup of expired entries MUST run periodically.

## Failure Semantics

### Query-level behavior

- Partial upstream failures MUST be reported under `failures`.
- Successful entries MUST still be returned in `server_keys`.
- `server_keys` MUST contain only successfully resolved and verified entries.
- `failures` MUST be a map keyed by requested server name.
- A server MUST NOT appear in both `server_keys` and `failures` for the same request.
- If all requested servers fail, response remains `200` with `server_keys: []` and populated `failures`.

### Error codes

Common matrix-compatible codes:

- `M_BAD_JSON` for malformed/invalid JSON envelope,
- `M_INVALID_PARAM` for invalid parameter semantics,
- `M_NOT_FOUND` for missing key ID on own-key endpoint,
- `M_TOO_LARGE` for oversized request body.

## Operational Endpoint Semantics

- `/_mxkeys/health`: process health check.
- `/_mxkeys/live`: liveness (process alive).
- `/_mxkeys/ready`: readiness (DB + signing-key readiness).
- `/_mxkeys/status`: detailed runtime status (cache/db/uptime fields).
- `/_mxkeys/metrics`: Prometheus exposition format.

## Performance and Abuse Controls

MXKeys MUST enforce:

- rate limiting,
- request size limits,
- max server-count per query,
- concurrency limits for upstream fetch paths.

## Timeout and Retry Policy

For remote key fetch behavior:

- fetch timeout is configuration-driven (`keys.fetch_timeout_s`),
- direct fetch retries are attempted for transient/network errors,
- retry backoff is exponential (base 200ms in current implementation),
- non-retryable/permanent validation errors MUST fail fast,
- retry attempts are bounded and configuration-driven (`RetryAttempts`, default 3).

## Compatibility Contract

The core API behavior (`/_matrix/key/v2/server*`, `/_matrix/key/v2/query`) is treated as stable.
Changes that modify response shape, validation semantics, or error-code mapping require:

- explicit changelog entry,
- test updates,
- conformance matrix review.
