# Changelog

All notable changes to MXKeys are documented in this file.

## [0.2.0] - 2026-04-13

Production hardening release. Four-pass security audit with 47 verified fixes.

### Added

- Signed Tree Head endpoint (`GET /_mxkeys/transparency/signed-head`) with ed25519 signature for external verification
- Public key discovery endpoint (`GET /_mxkeys/notary/key`) with fingerprint for external STH verification
- CLI verifier tool (`cmd/mxkeys-verify`) for STH signature and consistency monitoring
- Consistency proof generation in Merkle tree (`GetConsistencyProof` + `VerifyConsistencyProof`)
- Circuit breaker stats endpoint (`GET /_mxkeys/circuits`) for upstream health visibility
- Circuit breaker Prometheus metrics (`mxkeys_circuit_breaker_servers`, `trips_total`, `recoveries_total`)
- Request ID (`request_id`) included in all Matrix error JSON response bodies
- Cluster SLA documentation in ADR-0001 with per-mode property table
- Route normalization for operational endpoints in Prometheus metrics
- Transparency verification guide (`docs/transparency-verification.md`)

### Security

- Block HTTP redirects on federation key fetch client (SSRF mitigation)
- Validate redirect targets in well-known resolver (scheme + private IP check)
- Sanitize database error messages in status endpoint responses
- Sanitize JSON parse error details in key query error responses
- Set Content-Type header in all Matrix error responses
- Remove raw error strings from analytics fetch failure records
- Remove dangerous default server.name ("mxkeys.org") — explicit configuration required
- Add SQL identifier validation in NewTransparencyLog constructor
- Add ErrNotaryKeyMismatch to IsPermanentError classification (prevents futile retries)

### Fixed

- Fix Prometheus histogram double cumulation in bucket counts
- Fix go_threads metric reporting CPU count instead of OS thread count
- Fix handleReadiness blocking indefinitely on database — 2s context timeout added
- Fix handleStatus SQL query missing request context
- Fix nil map panic in analytics when response has no signatures field
- Fix slice bounds panic in GetTopRotators with negative limit
- Fix cache GetServerKeys returning shared pointer — clone on cache hit
- Fix GetCachedKey in cluster returning pointer under mutex — copy on return
- Fix mergeState panic on nil KeyEntry from JSON deserialization
- Fix transparency Cleanup not synchronizing in-memory merkle tree and key history
- Fix checkAnomalies silently dropping appendEntry errors
- Fix non-deterministic map iteration in analytics and transparency logging
- Fix DeleteExpiredKeys returning count from first table only
- Fix VerifyChain entries_checked reporting requested limit instead of actual count
- Fix parseServerName accepting out-of-range ports for IPv6 literals
- Fix environment variable parse errors silently ignored — strict error propagation
- Fix cluster.Start failure not blocking server startup
- Fix rate limited requests counter lacking limiter type distinction

### Changed

- Split metrics.go (607 lines) into metrics.go, metrics_write.go, metrics_runtime.go
- Replace bubble sort with sort.Slice in GetTopRotators
- Replace string-based error matching in isRetryableError with net.Error type checks (string fallback retained)
- Rewrite applyEnvOverrides with envInt/envFloat/envBool helpers returning errors
- Add logging.level and logging.format validation (debug/info/warn/error, text/json)
- Mark Raft consensus mode as experimental with runtime warning
- Upgrade Docker HEALTHCHECK from /_mxkeys/health to /_mxkeys/ready with 10s start-period
- Add rate limit label ("global"/"query") to mxkeys_rate_limited_requests_total metric
- Transparency and cluster initialization failures are now fatal when enabled in config

### Removed

- Remove unused serverNameRegex and keyIDRegex from validation.go
- Remove dead connectPeers function and peers field from Raft node
- Remove unused reason parameter from RecordFetchFailure

### Infrastructure

- Add RPC error logging in Raft handleConnection (read/verify failures)
- Update config.example.yaml with Raft experimental note

## [0.1.0] - 2026-03-17

Initial public release. Matrix Federation Key Notary with:

- Full Matrix v1.16 key server API (GET /v2/server, POST /v2/query)
- Ed25519 signing with canonical JSON
- Multi-layer caching (memory + PostgreSQL)
- SSRF protection with DNS pinning
- Rate limiting with per-IP token buckets
- Circuit breaker and semaphore for upstream fetches
- Trust policy engine (deny/allow lists, notary signatures, key age)
- Key transparency log with hash chaining and Merkle proofs
- Cluster support (CRDT with HMAC-authenticated transport)
- Raft consensus mode (experimental)
- Prometheus-compatible metrics
- Structured logging (text/JSON)
- Graceful shutdown
- CI pipeline: unit, integration, race detector, vet, gofmt, govulncheck, gosec
