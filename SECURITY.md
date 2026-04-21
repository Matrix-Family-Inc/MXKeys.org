Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Security Policy

MXKeys is a Matrix federation-key notary. Signing-key compromise
compromises every homeserver that trusts the notary. Treat it as
identity-critical infrastructure.

## Scope

In scope for security reports:

- Go source under `cmd/` and `internal/`.
- Cryptographic code: ed25519 signing, canonical JSON, HMAC-SHA256
  over cluster RPCs and WAL records, AES-256-GCM + PBKDF2 for
  at-rest key encryption, TLS for cluster transport.
- On-disk formats: `MXKENC01` key envelope, WAL v3, Raft snapshot,
  transparency log schema.
- HTTP surface: `/_matrix/key/v2/*`, `/_mxkeys/*`, CRDT and Raft
  replication RPCs.

Out of scope:

- Vulnerabilities in upstream dependencies (report upstream; we can
  coordinate if needed).
- Denial-of-service that requires root on the host or physical
  access to disk.
- Third-party forks that diverge from this repository.

## Reporting

Send reports privately. Do not open a public GitHub issue or post
in a public Matrix room until a disclosure date is agreed.

- Email: `dev@matrix.family`
- Subject prefix: `[mxkeys-security]`
- Required contents: affected version or commit SHA, reproduction
  or proof-of-concept, environment details, impact assessment,
  preferred disclosure timeline.

Acknowledgement target: 72 hours. If that target is missed,
resend the report and copy the maintainer directly. If a further
72 hours pass without response, open a private GitHub security
advisory draft.

## Handling

1. Report received. Acknowledged within 72 hours.
2. Reproduction and severity classified on a private branch.
3. Fix developed and reviewed privately; reporter may review the
   draft.
4. Release prepared; CVE requested when applicable.
5. Disclosure date agreed with the reporter. Public advisory,
   CHANGELOG entry, and release notes land on that date.

No fixed turnaround is published. Fix-to-release cycles are
measured in days for critical bugs and weeks for lower severity.

## Severity

- **Critical**: attacker forges a notary signature, reads the
  on-disk signing key, bypasses TLS or mTLS on the cluster
  transport, or causes a node to accept forged Raft traffic.
- **High**: attacker crashes the service, corrupts the WAL or
  snapshot, exhausts resources through a single unauthenticated
  request, or poisons the transparency log.
- **Moderate/Low**: information disclosure without signing-key
  impact, rate-limit bypass requiring heavy traffic, bugs whose
  exploitability depends on operator misconfiguration.

## Security Properties

See `docs/threat-model.md` for full analysis. Summary of what
this release provides:

- **Signing-key confidentiality**: the ed25519 key may be stored
  encrypted at rest as an `MXKENC01` envelope (AES-256-GCM,
  PBKDF2-HMAC-SHA256 at 600 000 iterations) when
  `keys.encryption.passphrase_env` is set. Otherwise the key is
  stored at 0600 as plaintext. On Linux the loaded seed's pages
  are mlock'd via `syscall.Mlock` so the key does not land in
  swap. The mlock call is best-effort; failures surface as a
  startup WARN and do not block startup.
- **Cluster confidentiality**: TLS 1.3 with optional mutual auth
  covers both CRDT and Raft transports. TLS 1.2 and earlier are
  not accepted. When TLS is disabled, cluster traffic carries
  HMAC-SHA256 over canonical JSON but is not encrypted.
- **WAL integrity**: every record carries CRC32C (bit rot) and
  HMAC-SHA256 (intentional tampering). A CRC-valid but HMAC-
  invalid record surfaces as `ErrWALTampered`.
- **Log compaction / catch-up**: Raft snapshots stream in 512 KiB
  chunks. Each chunk carries the `(LastIncludedIndex,
  LastIncludedTerm)` tuple so a leader change mid-transfer is
  detected.
- **Election stability**: Raft pre-vote prevents a partitioned or
  flapping node from forcing a new election on a healthy leader.
- **Graceful shutdown**: SIGTERM flips `/_mxkeys/readyz` to 503,
  waits `predrain_delay` for load-balancer propagation, drains
  HTTP, stops the cluster, closes the DB handle. A second signal
  forces exit.

Explicit limits:

- No HSM path is shipped. The KMS provider is a stub; operators
  who need HSM integration extend `internal/keys/keyprovider`.
- Root on the host defeats the at-rest encryption: the passphrase
  env var is readable from `/proc/<pid>/environ`. Defense in depth
  is operator-controlled: systemd `LoadCredential`, Kubernetes
  `Secret` mounts, or an external vault.

## Supported Versions

- `1.x` is the first supported line. Security fixes backport to
  the most recent minor release.
- Pre-1.0 tags are not supported.

## Credit

Reporters of verified issues are credited by name in the release
notes and the GitHub security advisory unless they opt out.
