Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Fri Apr 24 2026 UTC
Status: Updated

# ADR-0007: Signing Key Provider Abstraction

## Status

Accepted.

## Visibility

Public.

## Ecosystem Scope

This ADR is the MXKeys implementation of
`../../../ecosystem-docs/adr/ECO-0008-signing-key-lifecycle.md`. The ecosystem
ADR owns signing-key lifecycle invariants; this file owns MXKeys provider
interfaces, at-rest protection, and operator-facing backend choices.

## Context

The notary's ed25519 signing key is the root of trust. Leakage
invalidates every perspective signature this notary ever issued.
The original implementation read and wrote the key as raw bytes
under `keys.storage_path`. Three structural gaps with that model:

1. Operator ergonomics: orchestrator-mounted secrets (Kubernetes
   Secrets via env, systemd credentials) require a round-trip
   through a file path that may or may not be persistent.
2. External KMS: operators with compliance requirements keep the
   private material inside a dedicated key-management system
   rather than on the application host.
3. Testability: tests for permission hardening had to reach into
   private Notary state because the key lifecycle was coupled to
   the service type.

## Decision

`internal/keys/keyprovider` defines a `Provider` interface:

```go
type Provider interface {
    LoadOrGenerate(ctx context.Context) (ed25519.PrivateKey, string, error)
    PublicKey() ed25519.PublicKey
    Sign(ctx context.Context, data []byte) ([]byte, error)
    Kind() Kind
}
```

Implementations:

- `FileProvider`. Disk storage. Generates on first call. Enforces
  0700 on the directory and 0600 on the key file at every open so
  an out-of-band `chmod` cannot weaken posture silently. When a
  passphrase is configured the key is stored as an `MXKENC01`
  envelope (AES-256-GCM, PBKDF2-HMAC-SHA256 at 600 000
  iterations). A legacy plaintext key is upgraded in place on the
  next load; the plaintext file is then removed. Plaintext mode
  refuses to read an existing `.enc` file rather than silently
  ignoring it.
- `EnvProvider`. Reads a base64-encoded seed or full key from an
  environment variable. No generation; the operator provisions
  the key.
- `KMSStub`. Placeholder that documents the interface for a
  future external-KMS implementation. `LoadOrGenerate` and `Sign`
  return `ErrNotImplemented`.

Server initialization builds the provider from `keys` config:

- `keys.storage_path` selects `FileProvider`.
- `keys.encryption.passphrase_env` names an environment variable
  that holds the passphrase for at-rest encryption. An empty
  value is a hard error, not a fallback to plaintext.

The Notary retains a legacy `NewNotary` constructor that wraps
the file backend without encryption, preserving API compatibility
with embedders that have not migrated. Server code uses
`NewNotaryWithConfig` with an explicit provider.

## Consequences

- Signing-key hygiene tests live in `internal/keys/keyprovider`
  and exercise the provider directly.
- An external-KMS implementation slots in by implementing the
  interface and adding a branch in `keyprovider.New`.
- Operators running with `file` without a passphrase see the same
  on-disk layout as prior versions.
- Operators who set a passphrase get AES-256-GCM at rest with
  PBKDF2-HMAC-SHA256 at the OWASP-recommended iteration count.
- Backup and rotation procedures operate at the provider
  boundary. See `docs/runbook/key-rotation.md` and
  `docs/runbook/backup-restore.md`.

## Alternatives Considered

- Hard-coded file path with symlink into Kubernetes Secret
  mounts. Rejected: symlink plus read-only filesystem surfaces
  as startup errors that are hard to diagnose.
- Embed a specific KMS client (AWS, GCP, Vault) immediately.
  Rejected until there is a concrete operator requirement for
  one. The stub keeps the interface fixed.

## References

- ECO-0008 Signing Key Lifecycle Policy - shared lifecycle invariants for
  active, superseded, and rotated signing keys.
- `internal/keys/keyprovider/` - provider interface and concrete signing-key
  backends.
- `internal/keys/keyprovider/file_crypto.go` - encrypted file-provider envelope
  implementation.
- `docs/runbook/key-rotation.md` - operator procedure for replacing signing
  keys.
- `docs/runbook/backup-restore.md` - backup and restore procedure for key
  material and service state.

## Alternatives

None recorded at authoring time. Any future revision that modifies this decision must list the rejected options explicitly.
