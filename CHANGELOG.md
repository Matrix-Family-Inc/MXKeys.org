# Changelog

All notable changes to MXKeys are documented in this file.

## [Unreleased]

No breaking changes to the Matrix federation API contract.

### Added

- `raft.Node.snapMu` (`internal/zero/raft/raft.go`): serialises
  every writer of raft.snapshot on disk and the matching in-memory
  snapshotIndex/snapshotTerm/logOffset bookkeeping.
  `CompactLog` and every `handleInstallSnapshot` invocation
  acquire it. Fixes the CompactLog vs InstallSnapshot and
  InstallSnapshot vs InstallSnapshot races: the persisted file
  and the pending-transfer spill state can no longer interleave.
  Lock order: snapMu → n.mu; never held while taking an
  application-level lock.
- `raft.SnapshotInstaller` is now streaming:
  `func(r io.Reader, size int64, lastIncludedIndex,
  lastIncludedTerm uint64) error`. The installer consumes the
  snapshot payload directly from a reader backed by the spill
  file (or a bytes.Reader in the in-memory test path). No full
  `[]byte` of the payload is ever materialised on the Go heap,
  so peak transient RAM at install time is O(streaming decoder
  buffer) + O(decoded state), not O(payload size) as before.
  Breaking API change for pre-1.0.0; every call site in the
  repository is updated.
- Spill file is now the final snapshot format from first byte.
  `beginPendingSnapshot` writes the SaveSnapshot header
  (magic + last_index + last_term + placeholder length + placeholder
  CRC) up front; `appendPendingSnapshot` feeds data bytes plus an
  incremental Castagnoli CRC; `finalizePendingSnapshot` patches the
  real length and CRC into the header and fsyncs. On installer
  success the spill file is atomically renamed to raft.snapshot,
  replacing the previous SaveSnapshot pass that copied bytes
  through RAM.
- `raft.LoadSnapshotReader` (`internal/zero/raft/snapshot_load.go`):
  opens raft.snapshot and returns a reader positioned at the
  start of the data portion plus the metadata, without
  buffering the payload. `LoadFromDisk` now uses it so the
  startup restore path also stays at O(chunk) peak RAM.
- `raft.readSnapshotHeader` centralises the magic + header + CRC
  field decoding shared by `LoadSnapshot`, `LoadSnapshotReader`,
  and `SaveSnapshot` (via `snapshotHeaderSize`,
  `snapshotHeaderLenOffset`, `snapshotHeaderCRCOffset`
  constants).
- Monotonicity guard in `handleInstallSnapshot`: an
  `InstallSnapshot` whose `LastIncludedIndex` is at or below the
  current `snapshotIndex` is ACKed idempotently with
  `Success=true` without invoking the installer or touching
  the spill file. Prevents a stale leader or a reordered
  concurrent install from rolling the persisted snapshot
  backwards.
- `cluster.installKeySnapshot` consumes the snapshot from an
  `io.Reader` via `json.NewDecoder` so the cluster-side installer
  never allocates a parallel full-size byte buffer.
- Concurrency tests: `TestCompactLogSerializesWithInstallSnapshot`
  blocks the provider inside `CompactLog` to verify
  `handleInstallSnapshot` cannot complete its persist while we
  hold snapMu, and asserts the final on-disk
  `LastIncludedIndex` reflects the strictly higher of the two
  parallel writers (not the older one).
  `TestConcurrentInstallSnapshotHandlersDoNotTrashSpillFile`
  drives two InstallSnapshot RPCs in parallel and asserts the
  monotonicity guard plus serialisation preserve the final
  file at the higher index.
- Disk-backed spill for incoming InstallSnapshot transfers.
  `internal/zero/raft/pending_snapshot.go` streams each chunk into
  `stateDir/raft.snapshot.recv` so follower RAM stays
  O(`snapshotChunkSize`) regardless of total snapshot size, instead
  of holding the whole in-flight payload (up to 256 MiB) in a
  growing `[]byte`. The in-memory path is retained only for
  `stateDir == ""` in-memory raft mode used by tests. New error
  `ErrPendingSnapshotOverflow` makes the `maxSnapshotSize` cap
  explicit: a chunk append that would cross the cap is rejected
  with `Success=false` before any write hits disk or memory.
- `LoadFromDisk` cleans any stale `raft.snapshot.recv` left by a
  crashed mid-transfer so a fresh transfer on the next startup
  never inherits partial bytes.
- `raft.Node.sendRPCCtx`
  (`internal/zero/raft/network.go`): a context-aware variant of
  `sendRPC` that closes the underlying TCP connection when the
  caller's context fires so an outstanding write/read wakes up
  immediately with `ctx.Err()`. `sendRPC` is now a thin wrapper
  around `sendRPCCtx(context.Background(), ...)` for call paths
  (election, replication) that do not yet carry a context; every
  new call site with a real context (Propose-forward) uses
  `sendRPCCtx` directly.
- `cluster.Cluster.InstalledSnapshotIndex()` returns the highest
  `LastIncludedIndex` the snapshot installer has processed on
  this instance. Observability hook used by startup diagnostics,
  operator tooling, and the raft end-to-end test to prove the
  restore path went through the snapshot installer rather than a
  WAL-only replay fallback. Zero until the first install.
- `cluster.Cluster.installedSnapshotIndex` (unexported
  `atomic.Uint64` backing the accessor above).
- Unit tests:
  - `TestProposeRespectsCancelledContextImmediately`,
    `TestProposeReturnsErrNoLeaderWhenAddrMissing`
    (`internal/zero/raft/propose_test.go`) lock in the terminal
    context contract of Propose.
  - `TestHandleAppendEntriesDoesNotClobberLeaderAddrWithEmpty`,
    `TestHandleAppendEntriesAcceptsUpdatedLeaderAddr`,
    `TestHandleInstallSnapshotDoesNotClobberLeaderAddrWithEmpty`
    pin the leaderAddr preservation rule across both RPCs.
- Atomic snapshot capture contract for Raft state machines. The
  `raft.SnapshotProvider` signature now returns
  `(data []byte, lastAppliedIndex uint64, err error)`. The
  application MUST capture both fields under the same lock that
  serialises `onApply` writes. `CompactLog` persists the
  provider-reported index verbatim (not the raft layer's current
  `lastApplied`) after validating it is strictly above the current
  snapshot boundary and at or below the commit index. Two replicas
  that have applied the same log prefix now produce byte-identical
  snapshot files at the same `LastIncludedIndex`.
- `cluster.CRDTState.raftLastApplied` (`internal/cluster/cluster.go`):
  the apply callback in `startRaft` updates both the LWW cache and
  this counter under a single `c.state.mu.Lock`, so the
  provider observes a coherent (payload, index) pair.
- `cluster.storeEntryLocked` / `cluster.applyEntryLocked`
  (`internal/cluster/state.go`): caller-holds-lock variants of the
  LWW merge, used by the apply callback to avoid a second lock
  acquisition. Public `storeEntry` is now a thin wrapper.
- Shutdown-aware contexts for raft-mode application writes.
  `raft.Node.ctxWithStop` (`internal/zero/raft/context.go`) derives
  a timeout-bounded context that also cancels on stopCh close,
  used by `driveInstallSnapshot` and `handleForwardProposal`.
  `cluster.Cluster.proposeCtx` (`internal/cluster/propose_ctx.go`)
  does the same for `BroadcastKeyUpdate` against the cluster's
  own stopCh. Shutdown no longer waits on the full
  `CommitTimeout` for an in-flight snapshot stream or forwarded
  proposal to drain.
- `internal/cluster/cluster_raft_e2e_test.go`:
  `TestRaftClusterEndToEndWriteCompactRestart` exercises the full
  production path (`BroadcastKeyUpdate` on a follower → Propose
  forward → leader Submit → commit → onApply on every replica →
  CompactLog → restart one node from its state directory →
  `GetCachedKey` returns the original entry). Locks in the
  atomicity contract and restart durability end to end.
- Automatic InstallSnapshot catch-up in production (see Fixed).
  `sendAppendEntries` now detects peers whose `nextIndex` sits at or
  below the leader's snapshot boundary and switches to
  `SendInstallSnapshot` in place of `AppendEntries`. Eliminates the
  deadlock where a lagging follower would loop forever decrementing
  through entries that only exist in the leader's on-disk snapshot.
- Follower-forward writes via `raft.Node.Propose`
  (`internal/zero/raft/propose.go`) and the new
  `MsgForwardProposal` RPC. A follower that receives a write forwards
  it to the current leader over a SharedSecret-signed RPC; the
  leader submits locally and responds with the Submit outcome.
  `cluster.BroadcastKeyUpdate` now routes every raft-mode write
  through `Propose`, so follower-originated cache updates are
  actually replicated cluster-wide instead of being silently
  dropped as `ErrNotLeader`.
- `raft.Config.AdvertiseAddr` and `AppendEntriesRequest.LeaderAddress`
  / `InstallSnapshotRequest.LeaderAddress`. The leader embeds its
  dialable "host:port" in every AE and InstallSnapshot RPC so
  followers learn a concrete forwarding endpoint used by Propose.
  The cluster runtime passes `advertiseAddress:advertisePort` into
  the raft config automatically.
- `raft.ErrNoLeader`: returned by Propose when the caller is not the
  leader and no leader is currently known (e.g. mid-election).
- Cluster Raft state-machine snapshotting
  (`internal/cluster/snapshot.go`). `snapshotKeyState` and
  `installKeySnapshot` are registered via
  `node.SetSnapshotProvider` / `node.SetSnapshotInstaller` before
  `node.Start()` so `LoadFromDisk` can restore the LWW key cache
  from a persisted snapshot and so `CompactLog` has a provider to
  call. Wire format is versioned (`keySnapshotVersion`);
  unknown versions are refused via `ErrUnsupportedSnapshotVersion`.
  Serialization is deterministic (JSON with sorted keys) so
  replicas at the same commit index produce byte-identical
  snapshots.
- Raft background log compaction loop (`raftCompactionLoop` in
  `internal/cluster/snapshot.go`). Ticks every
  `compactionCheckInterval` (30 s) and triggers `CompactLog` when
  the in-memory log exceeds `compactionLogThreshold` (1024
  entries). Bounds recovery time to snapshot size plus the most
  recent window.
- `InstallSnapshotResponse.Success` and
  `InstallSnapshotResponse.BytesStored`
  (`internal/zero/raft/snapshot.go`). The follower sets
  `Success=true` only when a non-Done chunk was buffered cleanly
  or a Done chunk was installed and persisted; it sets
  `Success=false` on stale term, offset gap, installer error, or
  snapshot save error. `ErrSnapshotRejected` (new public error in
  `internal/zero/raft/snapshot_send.go`) surfaces a follower
  rejection to the replication loop.
- Schema migrations runner (`internal/storage/migrations`) with
  embedded versioned SQL, per-migration transactions, and a
  `schema_migrations` bookkeeping table. Startup applies pending
  migrations before any dependent subsystem touches the database.
  Shipped migrations: `0001_initial.sql`, `0002_transparency_log.sql`.
  See ADR-0008.
- Signing-key provider abstraction (`internal/keys/keyprovider`)
  with `FileProvider`, `EnvProvider`, and `KMSStub`. Backward-
  compatible default. See ADR-0007.
- Signing-key at-rest encryption (`internal/keys/keyprovider/
  file_crypto.go`): AES-256-GCM envelope (`MXKENC01`) with KEK
  derived via PBKDF2-HMAC-SHA256 at 600 000 iterations. Opt-in
  via `keys.encryption.passphrase_env`. Legacy plaintext key is
  upgraded on first load when a passphrase is configured.
- Cluster transport TLS (`internal/zero/nettls`): server and
  client config loaders, mutual TLS, TLS 1.3 by default (1.2
  opt-in). Wired into CRDT listener, CRDT dials, and Raft
  network. Configured under `cluster.tls.*`.
- Raft WAL v3 (`internal/zero/raft/wal*.go`): per-record
  CRC32C (bit rot) and HMAC-SHA256 (tamper detection, keyed
  from `cluster.shared_secret`). Group-commit batcher, bounded-
  queue back-pressure, atomic truncate-rewrite. `ErrWALTampered`
  distinguishes intentional writes from bit rot. v2 files refused
  with `ErrWALLegacyFormat`.
- Raft snapshot chunking: `InstallSnapshot` streams in 512 KiB
  chunks with `(LastIncludedIndex, LastIncludedTerm)` on every
  chunk; the follower reassembles and applies on `Done=true`.
- Raft pre-vote extension: non-mutating `MsgPreVote` round before
  bumping `currentTerm`. Prevents a partitioned or flapping node
  from unseating a healthy leader.
- Graceful shutdown: SIGTERM flips `/_mxkeys/readyz` to 503
  (`draining`), waits `server.predrain_delay` (default 5 s) for
  load-balancer propagation, drains HTTP inside
  `server.shutdown_timeout` (default 30 s), stops the cluster,
  closes the DB handle. A second signal forces exit 130.
- LRU per-IP rate limiter (`internal/server/ratelimit.go`) with
  O(1) eviction via `container/list` + `map[string]*list.Element`.
- CLI flags: `-config /path/to/config.yaml` and `-version`.
- Config guards: `cluster.shared_secret` rejects placeholder
  strings and enforces 32+ characters. `trusted_notaries` entries
  reject placeholder public keys and require a valid base64
  ed25519 length. Cluster-transport HMAC payload uses canonical
  JSON.
- File-size lint (`scripts/file-size-lint.sh`): warn at 300 lines,
  fail at 500. New `file-size` CI job. See ADR-0010.
- Fuzz targets: `FuzzJSON`, `FuzzMarshalRoundTrip`
  (`internal/zero/canonical`), `FuzzValidateServerName`,
  `FuzzValidateKeyID`, `FuzzDecodeStrictJSON`
  (`internal/server`), `FuzzParseServerName` (`internal/keys`).
  Run 30 s per target in the `fuzz-quick` CI job.
- Golden vectors for canonical JSON
  (`internal/zero/canonical/testdata/golden_vectors.json`): 16
  fixtures covering boundary integers, escapes, Unicode, deep
  nesting, and realistic federation shapes.
- CI gates: `coverage`, `staticcheck`, `errcheck` (`-ignoretests`
  with a curated exclude list), `fuzz-quick`, `file-size`.
- Landing E2E smoke (`@playwright/test`): home render, RTL toggle
  on `?lang=ar`, mobile menu.
- Landing mandatory stack:
  - `msw` 2.x with `server.ts`, `browser.ts`, and `handlers.ts`;
    Vitest setup starts and stops the node server per suite.
  - `react-hook-form` + `@hookform/resolvers/zod` + a
    `features/notary-lookup` form.
  - Storybook 10 config with stories for `Logo` and
    `NotaryLookupForm`.
- Operator tooling: `scripts/mxkeys-backup.sh`,
  `scripts/mxkeys-restore.sh`, `scripts/build-release.sh`
  (reproducible: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"`,
  `SOURCE_DATE_EPOCH` from `git log`, optional CycloneDX SBOM).
- ADRs: 0006 (file header standard), 0007 (signing-key provider),
  0008 (schema migrations), 0009 (landing stack), 0010 (file-size
  policy).
- Runbooks: `docs/runbook/backup-restore.md`,
  `docs/runbook/cluster-disaster-recovery.md`,
  `docs/runbook/key-rotation.md`,
  `docs/runbook/release.md`,
  `docs/runbook/schema-migration.md`.
- `SECURITY.md` replaces the prior placeholder with scope,
  reporting, severity classification, and documented security
  properties plus limits.

### Changed

- Go toolchain bumped from 1.22 to 1.26. `GOTOOLCHAIN` workaround
  for govulncheck removed from CI and the local preflight.
- Dockerfile base images bumped to `golang:1.26-alpine` and
  `alpine:3.21`.
- README Go badge bumped to 1.26+.
- Unified file headers across tracked source, config, docs,
  shell, and workflow files. Fields reduced to Project / Company /
  Maintainer / Contact / Date / Status (Owner, Role, Support,
  Matrix fields removed). See ADR-0006.
- Landing: complete Feature-Sliced Design reorganization with
  Zustand (mobile nav), Zod (env and form validation), TanStack
  Router and Query providers, Sentry + `AppErrorBoundary`, lazy
  i18n via dynamic imports. Main bundle ~273 KB (gzip ~82 KB);
  22 locales ship as separate chunks of 7 - 12 KB each. See
  ADR-0009.
- Landing: `VITE_SITE_URL` (Zod-validated) replaces hard-coded
  `https://mxkeys.org` in `index.html`, `robots.txt`,
  `sitemap.xml`. The `htmlEnvReplace` Vite plugin substitutes
  `__MXKEYS_SITE_URL__` and `__MXKEYS_ENVIRONMENT__` at build
  and serve time.
- Landing: `jsdom` replaced by `happy-dom` for vitest.
- `raft.wal` format: CRC polynomial IEEE to Castagnoli. Magic
  bumped to `MXKS_WAL_v3` to carry the HMAC tag.
- `InstallSnapshotRequest.Data` type changed to `[]byte`
  (binary-safe via base64 over JSON).
- `cluster.raft_state_dir` is now mandatory when
  `cluster.consensus_mode=raft`. `config.Validate()` rejects the
  empty value (`TestValidateRaftRequiresStateDir`); the runtime
  no longer degrades silently to in-memory Raft. Matches the
  durability promise stated in `docs/architecture.md` and
  ADR-0001.
- `internal/cluster/state.go` `BroadcastKeyUpdate` no longer
  writes the entry into the local LWW cache before `Submit` in
  `raft` mode. Only the apply callback (after commit) populates
  `c.state.keys`, so a non-leader call or a `Submit` failure can
  no longer leave unreplicated state visible via
  `GetCachedKey`. CRDT-mode behaviour is unchanged.
- `SendInstallSnapshot`
  (`internal/zero/raft/snapshot_send.go`, split out of
  `snapshot_rpc.go`) now advances `nextIndex` / `matchIndex` for
  a peer only when the follower ACKed the Done chunk with
  `Success=true`. Transport errors, decode errors, and
  `Success=false` ACKs leave peer bookkeeping untouched and
  return `ErrSnapshotRejected` so the next replication pass
  restarts from offset 0. Earlier behaviour advanced the indices
  unconditionally after the last chunk.
- `handleInstallSnapshot` no longer swallows `SaveSnapshot`
  errors; a failed persist rejects the install with
  `Success=false` so the leader retries rather than treating a
  best-effort install as a successful snapshot install.
- File-size policy: target 250 - 300 lines, hard ceiling 500
  lines. See ADR-0010. Earlier production files already split
  below 300 stay as they are.
- Coverage gate floors (see `scripts/coverage-gate.sh`): total
  >= 50%; per-package floors for `internal/config`,
  `zero/config`, `zero/log`, `zero/merkle`, `zero/raft`,
  `keyprovider`, `cluster`, `server`, `keys`, `zero/canonical`,
  `zero/metrics`, `storage/migrations`.

### Fixed

- `cmd/mxkeys/main.go` was present on disk but not tracked in
  git. Clones could not build without it. Now tracked.
- staticcheck findings: SA9004 in `middleware.go`, SA1012 in a
  deliberate nil-context test, U1000 for the unused
  `raft.persistEntries` helper (removed).
- errcheck findings: `cmd/mxkeys-verify` JSON encoder,
  `cmd/mxkeys` `srv.Close`, `internal/cluster/state.go`
  `fmt.Sscanf`, `internal/zero/metrics/handler.go`
  `registry.WriteTo`.
- `internal/keys/fetcher_retry.go` and
  `internal/keys/storage.go`: error classification now uses
  typed `net.Error` / `*net.OpError` / `*net.DNSError` /
  `*os.SyscallError` / `syscall.Errno` / `driver.ErrBadConn` /
  `context.DeadlineExceeded` / `io.ErrUnexpectedEOF`. The
  string-match fallback was removed.
- Raft `Submit` moves WAL persistence out of `n.mu` so the
  group-commit batcher can amortise fsync across concurrent
  submissions. Index assignment stays serialised through the
  lock; persist failure truncates the in-memory tail.
- Raft cluster runtime now wires snapshot provider / installer
  before `node.Start()`. Before this fix the callbacks lived in
  `internal/zero/raft` but no production path invoked them;
  `LoadFromDisk` and `handleInstallSnapshot` advanced
  `snapshotIndex` / `commitIndex` / `lastApplied` and truncated
  the log without restoring the application-level key state.
- Raft `InstallSnapshot` protocol no longer has silent
  follower-failure semantics. See the new `Success` field under
  **Changed**; the combined effect closes the path where a
  follower installer error or save error left the leader
  convinced the peer had caught up.
- Raft production replication now drives `InstallSnapshot`
  automatically after compaction. `sendAppendEntries` previously
  only shipped `AppendEntries`, so a peer whose `nextIndex` sat at
  or below the leader's snapshot boundary would reject every AE
  (`termAt` cannot resolve an index inside the compacted prefix),
  the leader would decrement `nextIndex` on every response, and the
  peer would never catch up. `SendInstallSnapshot` now lives on the
  heartbeat path via `needsSnapshotCatchUp` and
  `driveInstallSnapshot`, bounded by `CommitTimeout`.
- Raft `becomeLeader` initialises per-peer `nextIndex` via
  `logLen()+1` (absolute index) instead of `len(n.log)+1`. After
  compaction `logOffset > 0` and the in-memory slice length no
  longer reflects the tail position, so the previous formula placed
  `nextIndex` below the real tail and forced the new leader into
  the same stuck-on-compacted-prefix loop described above.
- Raft snapshot payload and `LastIncludedIndex` are no longer
  captured out of step. `CompactLog` previously read
  `n.lastApplied` under an RLock, released the lock, and only
  then called the provider; any `onApply` that ran in the
  intervening window caused the payload to reflect indices past
  the recorded boundary. Two replicas racing compaction at the
  same nominal index could produce byte-differing snapshot
  files, silently breaking the per-index determinism the audit
  model depends on. The new provider contract (see
  **Added**) forces the application to capture data and index
  under a single lock; `CompactLog` trusts the result and
  validates it against its own log view before persisting.
- `cluster.BroadcastKeyUpdate` and `raft.driveInstallSnapshot` /
  `handleForwardProposal` no longer block shutdown for the full
  `CommitTimeout`. They now run under contexts that cancel on
  stopCh close, so `Cluster.Stop()` / `Node.Stop()` evict
  in-flight proposals and snapshot streams immediately.
- `raft.Node.Propose` now terminally respects its caller context.
  Previously the follower path ignored `ctx` once it decided to
  forward: the context was not checked up front and not threaded
  into the underlying `sendRPC`, so a cancel during the
  forwarded round trip still waited for `net.Conn`'s default
  deadline. Propose now returns `ctx.Err()` early on a cancelled
  context and uses `sendRPCCtx` for the forward RPC itself.
- `handleAppendEntries` and `handleInstallSnapshot` no longer
  overwrite a known `leaderAddr` with an empty value. A mixed-
  version or misconfigured leader that ships RPCs without
  `LeaderAddress` would previously strip the follower's
  forwarding endpoint on every heartbeat and cause Propose to
  return `ErrNoLeader` even though leadership was healthy.
  Populated addresses still overwrite stale ones.
- Raft `CompactLog` and `handleInstallSnapshot` no longer race on
  disk state. Before this commit CompactLog released the lock
  between pre-validation and `SaveSnapshot`, letting an
  InstallSnapshot handler finish its own persist in the gap; the
  stale CompactLog then renamed its older-index file on top of the
  newer one. Likewise two concurrent InstallSnapshot handlers
  shared the `pendingSnapshot*` fields and the single
  `raft.snapshot.recv` file, so one could reset or remove the
  other's in-flight transfer during the installer/persist window.
  `snapMu` serialises both contours, the monotonicity guard
  idempotently ACKs older indices, and the spill is renamed
  atomically to become raft.snapshot. Both failure modes are
  pinned by `TestCompactLogSerializesWithInstallSnapshot` and
  `TestConcurrentInstallSnapshotHandlersDoNotTrashSpillFile`.
- Raft follower peak RAM during a large `InstallSnapshot` is now
  O(chunk). Previously `drainPendingSnapshot` read the entire
  spill file into a transient `[]byte` so the installer could
  consume it, and `SaveSnapshot` wrote it back to disk through
  RAM; together they allocated up to `maxSnapshotSize` (256 MiB)
  twice per transfer. The streaming installer + in-place
  spill-as-snapshot-format eliminate both allocations.
- WAL `TruncateBefore` failure after a successful
  `InstallSnapshot` no longer silently ignored. Previously the
  follower discarded the error (`_ = n.wal.TruncateBefore(...)`);
  now it logs a `Warn` with `snapshot_index` and the underlying
  error. The install itself remains committed (the snapshot
  supersedes every entry up to `LastIncludedIndex` and
  `LoadFromDisk` skips stale records on replay), but disk
  pressure and permission issues are now visible to operators
  rather than buried.
- `internal/cluster/cluster_raft_e2e_test.go`:
  `TestRaftClusterEndToEndWriteCompactRestart` now restarts the
  node that actually compacted (the leader at the time of
  `CompactLog`), asserts the snapshot file exists on that node's
  state directory before the restart, and asserts the reborn
  node's `InstalledSnapshotIndex() > 0` to prove the restore went
  through `installKeySnapshot` rather than a silent WAL-replay
  fallback. Previously `restartIdx` was hard-coded to zero, which
  silently exercised the wrong path whenever elections landed
  outside `nodes[0]`.
- Raft `BroadcastKeyUpdate` in cluster mode now transparently
  forwards follower-originated writes to the leader. Before this
  fix a fetch served by a follower was pinned to that node's
  local notary storage but never reached the replicated cluster
  cache, because `Submit` on a non-leader returns `ErrNotLeader`
  with no fallback. See the `Propose` / `MsgForwardProposal`
  entry under **Added**.
- Transparency log default table is created by the migrations
  runner (`sql/0002_transparency_log.sql`). The lazy-DDL path
  remains for operators using a custom `transparency.table_name`
  with a deprecation warning.

### Removed

- `internal/zero/router` package: dead code, referenced only in
  ADR-0002 prose. Routing is `http.ServeMux` in production.
- Pre-built binaries from the repository:
  `release/mxkeys-linux-amd64`, `release/mxkeys-linux-arm64`,
  `release/checksums.sha256` (15 MiB total). Release artifacts
  now ship via `scripts/build-release.sh` and GitHub Releases.
- Hard-coded `https://mxkeys.org` fallback in live-interop tests
  and CI scripts. `MXKEYS_LIVE_BASE_URL` is required when
  `MXKEYS_LIVE_TEST=1`; absent env var skips cleanly.
- Shared UI kit `Button`, `Container`, `ExternalLink`, and the
  `cn` helper in `landing/src/shared/`: no widget imported them,
  and the variants did not match widget needs. `Logo` and
  `TextField` remain.

### Infrastructure

- Go module path stays at `mxkeys`.
- `scripts/coverage-gate.sh`: per-package floors tracked inline
  so raises and drops are visible in diff review.
- `scripts/fuzz-quick.sh`: table-driven target list, `FUZZTIME`
  tunable for local deep passes.
- `scripts/errcheck-excludes.txt`: curated list of sinks safe to
  ignore (`*.Close`, `fmt.Fprint*`).
- `scripts/verify-github-branch-protection.sh`: required status
  checks list tracks every job in `pr-gate.yml`.

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
