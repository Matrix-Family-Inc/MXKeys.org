Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# ADR-0007: Signing Key Provider Abstraction

## Status

Accepted

## Context

The notary's ed25519 signing key is the root of trust exposed by
MXKeys: if it leaks, the notary's entire perspective-signature record
must be invalidated ecosystem-wide. The original implementation read
and wrote the key as raw bytes on local disk under
`keys.storage_path`. That works for single-node deployments but has
three structural problems:

1. **Operator ergonomics**: orchestrator-mounted secrets (Kubernetes
   Secrets via env, systemd-credentials) require a round-trip through
   a file path that may or may not be persistent.
2. **External KMS**: operators with compliance requirements need to
   keep the private material inside a dedicated key-management
   system (HSM, cloud KMS) rather than on the application host.
3. **Testability**: Notary tests that exercise permission hardening
   had to reach into private Notary state; the logic was coupled to
   the service type instead of living with the key.

## Decision

Introduce `internal/keys/keyprovider` with a `Provider` interface and
three implementations:

```go
type Provider interface {
    LoadOrGenerate(ctx context.Context) (ed25519.PrivateKey, string, error)
    PublicKey() ed25519.PublicKey
    Sign(ctx context.Context, data []byte) ([]byte, error)
    Kind() Kind
}
```

- `FileProvider`: backward-compatible disk storage. Generates on first
  call, enforces 0700 directory and 0600 file permissions on every
  open so out-of-band chmod cannot weaken posture silently.
- `EnvProvider`: reads a base64-encoded seed or full key from an env
  variable. No generation; operator is responsible for key provisioning.
- `KMSStub`: placeholder that documents the interface shape for future
  external-KMS integration. `LoadOrGenerate` and `Sign` return
  `ErrNotImplemented`.

The Notary retains a legacy `NewNotary` constructor that wraps the
file backend, preserving API compatibility with embedders that have
not yet migrated. New call sites use `NewNotaryWithConfig` with an
explicit provider.

Config knobs added to `keys` section (follow-up; currently file-only
is fully wired; env and kms paths require operator to construct the
provider directly and call `NewNotaryWithConfig`).

## Consequences

- Signing-key hygiene tests live in `internal/keys/keyprovider` and
  exercise the provider directly rather than reaching into Notary
  internals.
- A future external-KMS implementation slots in by implementing the
  interface and editing one switch in `keyprovider.New`.
- Operators running with `file` continue to see the same on-disk
  layout, permissions, and generation behavior.
- Backup/rotation procedures live in `docs/runbook/key-rotation.md`
  and operate at the provider boundary, not on file paths directly.

## Alternatives Considered

- Keep a hard-coded file path and let operators symlink from
  Kubernetes Secret mounts: rejected; symlink + readonly fs quirks
  are hard to debug and surface as cryptic startup errors.
- Embed full external-KMS client (AWS KMS, GCP KMS, HashiCorp Vault)
  immediately: rejected until a concrete operator requirement exists
  for a specific KMS; the `KMSStub` keeps the contract frozen so the
  integration is a drop-in later.

## References

- `internal/keys/keyprovider/`
- `docs/runbook/key-rotation.md`
