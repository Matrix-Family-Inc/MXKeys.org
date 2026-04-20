# Changelog

All notable changes to MXKeys are documented in this file.

## [Unreleased]

Ideal-pass release: architectural hardening, test-gate expansion,
and full landing reorganization. No breaking changes to the Matrix
federation API contract.

### Added

- Schema migrations runner (`internal/storage/migrations`) with embedded
  versioned SQL, per-migration transactions, and `schema_migrations`
  bookkeeping. Startup applies pending migrations before any dependent
  component touches the database. See ADR-0008.
- Signing-key provider abstraction (`internal/keys/keyprovider`) with
  `FileProvider`, `EnvProvider`, and `KMSStub` implementations. Backwards-
  compatible: existing file-backed deployments continue to work with no
  configuration change. See ADR-0007.
- Raft WAL (`internal/zero/raft/wal*.go`) with CRC32C per-record integrity,
  format magic `MXKS_WAL_v2`, group-commit batcher, bounded-queue back-
  pressure, and atomic truncate-rewrite. See ADR-0001.
- Raft snapshots (`internal/zero/raft/snapshot*.go`): CRC-protected
  `raft.snapshot` file, `InstallSnapshot` RPC for lagging-follower
  catch-up, and `Node.CompactLog` for periodic compaction. In-memory
  `logOffset` accessor layer so the slice can drop compacted prefixes
  without reindex-refactoring every call site.
- LRU per-IP rate limiter (`internal/server/ratelimit.go`) with O(1)
  eviction via `container/list` + `map[string]*list.Element`. Replaces
  the previous O(n) sort-based eviction path.
- CLI flags: `-config /path/to/config.yaml` and `-version`. Config.Load
  now accepts an explicit path and fails fast when that path is missing.
- Config guards: `cluster.shared_secret` rejects a known placeholder
  whitelist and enforces minimum 32 characters. Cluster-transport HMAC
  payload switched from ad-hoc string formatting to canonical JSON over
  the MACed fields, eliminating a class of structural-ambiguity
  collisions.
- Fuzz targets: `FuzzJSON`, `FuzzMarshalRoundTrip`
  (`internal/zero/canonical`), `FuzzValidateServerName`,
  `FuzzValidateKeyID`, `FuzzDecodeStrictJSON` (`internal/server`),
  `FuzzParseServerName` (`internal/keys`). Ran 30s per target in CI via
  `scripts/fuzz-quick.sh`.
- Golden vectors for canonical JSON
  (`internal/zero/canonical/testdata/golden_vectors.json`) with 16
  fixtures covering boundary integers, escapes, Unicode, deep nesting,
  and realistic federation shapes.
- CI gates: `coverage` (per-package floors + total floor via
  `scripts/coverage-gate.sh`), `staticcheck`, `errcheck` (`-ignoretests`
  + curated excludes), `fuzz-quick`.
- Landing E2E smoke: `@playwright/test` with three specs (home render,
  RTL direction toggle on `?lang=ar`, mobile menu open/close).
- New ADRs: 0006 (file header standard), 0007 (signing-key provider),
  0008 (schema migrations), 0009 (landing stack).
- Runbooks: `docs/runbook/key-rotation.md`,
  `docs/runbook/cluster-disaster-recovery.md`,
  `docs/runbook/schema-migration.md`.

### Changed

- Go toolchain bumped from 1.22 to 1.26; `GOTOOLCHAIN` workaround for
  govulncheck removed from CI and preflight.
- Dockerfile bumped to `golang:1.26-alpine` + `alpine:3.21`.
- README Go badge bumped to 1.26+.
- Unified file headers across all tracked source, config, docs, shell,
  and workflow files. Fields reduced to Project / Company / Maintainer /
  Contact / Date / Status (Owner, Role, Support, Matrix fields removed).
  See ADR-0006.
- Landing: complete Feature-Sliced Design reorganization with
  Zustand (mobile nav), Zod (env validation), CVA + clsx +
  tailwind-merge (shared UI), TanStack Router and Query providers,
  Sentry + ErrorBoundary, lazy i18n via dynamic imports. Main bundle
  dropped from ~552 KB to ~301 KB; 22 locales ship as separate chunks
  of 7-12 KB each. See ADR-0009.
- Landing: `VITE_SITE_URL` Zod-validated env-driven site URL replaces
  hardcoded `https://mxkeys.org` in `index.html`, `robots.txt`, and
  `sitemap.xml`. A Vite plugin substitutes `__MXKEYS_SITE_URL__` and
  `__MXKEYS_ENVIRONMENT__` placeholders at build/serve time.
- Landing: `jsdom` replaced by `happy-dom` for vitest (resolves the
  ESM-in-CJS error in `html-encoding-sniffer` on Node 20).
- `raft.wal` format: CRC polynomial IEEE -> Castagnoli, 12-byte magic
  prefix `MXKS_WAL_v2`. v1 never shipped outside the Phase 4 feature
  branch; operators with existing WAL data on a pre-release snapshot
  must start from an empty state dir.
- File size cap: every production Go file now <= 250 lines. Seven
  previously-oversized files split into focused modules.

### Fixed

- Critical: `cmd/mxkeys/main.go` was present on disk but never tracked
  in git. Clones could not build without it. Now tracked.
- staticcheck findings: SA9004 (incomplete const-group typing in
  `middleware.go`), SA1012 (nil context in deliberate nil-safety test),
  U1000 (unused `raft.persistEntries` helper, removed).
- errcheck findings: `cmd/mxkeys-verify` JSON encoder, `cmd/mxkeys`
  `srv.Close`, `internal/cluster/state.go` `fmt.Sscanf`,
  `internal/zero/metrics/handler.go` `registry.WriteTo`.

### Removed

- `internal/zero/router` package: dead code, referenced only in
  ADR-0002 prose. Routing has always been `http.ServeMux` in
  production.
- Pre-built binaries from the repository: `release/mxkeys-linux-amd64`,
  `release/mxkeys-linux-arm64`, `release/checksums.sha256` (15 MiB
  total). Artifacts now belong in GitHub Releases.
- Hardcoded `https://mxkeys.org` fallback in live-interop tests and CI
  scripts. `MXKEYS_LIVE_BASE_URL` is now required when
  `MXKEYS_LIVE_TEST=1`; absent env var skips cleanly.

### Infrastructure

- Go module stays at the short `mxkeys` identifier by operator choice
  (service, not a consumer-facing library). No module-path migration.
- `scripts/coverage-gate.sh`: per-package floors documented in the
  script so raises/drops are visible in diff review.
- `scripts/fuzz-quick.sh`: table-driven target list, `FUZZTIME`
  tunable for local deep passes.
- `scripts/errcheck-excludes.txt`: curated list of always-safe-to-ignore
  sinks (`*.Close`, `fmt.Fprint*`).
- Coverage gate: total >= 50%, per-package floors at
  internal/config 80, zero/config 85, zero/log 80, zero/merkle 65,
  zero/raft 60, keyprovider 65, cluster 60, server 55.

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
