Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Created

# Runbook: Notary Signing Key Rotation

## When to Run This

- Planned rotation on a fixed schedule (recommended quarterly for
  production notaries).
- Suspected compromise of the signing key (file permission leak,
  host intrusion, backup tape outside the documented chain of
  custody, etc.).
- Migration between key providers (file -> env, file -> KMS).

## Invariants

- The notary key ID (`ed25519:mxkeys`) stays stable. Clients expect
  the same identifier; the key bytes change while the label does not.
- Old keys must continue to serve verification for long enough that
  any in-flight federation traffic with the old key completes. The
  Matrix spec recommends keeping old keys available via
  `old_verify_keys` for at least the original `valid_until_ts` window.
- The transparency log MUST record the rotation so external
  verifiers can reconstruct the signer identity at any past
  `valid_until_ts`.

## Prerequisites

- Shell access to the notary host (for `file` provider) or operator
  credentials for the target KMS (for `env` / `kms` provider).
- A recent `pg_dump` of the MXKeys database.
- A backup of the current signing key file (see
  `docs/deployment.md` "Backup and Recovery").

## Step-by-Step (file provider)

1. Stop the notary:

   ```bash
   systemctl stop mxkeys
   ```

2. Back up the current key:

   ```bash
   install -d -m 700 /var/lib/mxkeys/keys/archive
   cp -a /var/lib/mxkeys/keys/mxkeys_ed25519.key \
      /var/lib/mxkeys/keys/archive/mxkeys_ed25519.$(date -u +%Y%m%dT%H%M%SZ).key
   ```

3. Remove the current key file. On next start the FileProvider
   generates a new 32-byte seed, writes 0600 under a 0700 directory.

   ```bash
   rm /var/lib/mxkeys/keys/mxkeys_ed25519.key
   ```

4. Start the notary:

   ```bash
   systemctl start mxkeys
   journalctl -u mxkeys -f
   ```

   Look for `Notary signing key loaded key_id=ed25519:mxkeys provider=file`
   in the logs.

5. Verify the new key:

   ```bash
   curl -fsS https://notary.example.org/_matrix/key/v2/server | jq .
   curl -fsS https://notary.example.org/_mxkeys/notary/key | jq .
   ```

   The `fingerprint` field of the second response is the SHA-256 of
   the new public key; record it in your release notes so operators
   downstream can pin it.

## Step-by-Step (env provider)

1. Generate a fresh seed offline:

   ```bash
   head -c 32 /dev/urandom | base64 -w0
   ```

2. Update the orchestrator secret (Kubernetes `Secret`,
   systemd-credentials, HashiCorp Vault, etc.) with the new value.

3. Restart the notary so `EnvProvider` picks up the new value.

4. Run the same verification steps as the file flow.

## Incident Compromise Path

If the old key is believed to be in adversarial hands:

1. Execute the rotation as above within 1 hour of detection.
2. Publish a signed advisory listing the compromised `key_id` and
   the rotation time. Operators pinning MXKeys via
   `trusted_notaries` must redeploy with the new pinned public key.
3. Invalidate affected PostgreSQL cache rows so downstream
   consumers re-fetch any keys whose perspective signatures were
   issued by the compromised key:

   ```sql
   DELETE FROM server_key_responses;
   DELETE FROM server_keys;
   ```

   (Cache is rehydrated transparently as federation traffic resumes.)

4. File a postmortem in `docs/` with the detection path, exposure
   window, and follow-up hardening. Update `docs/threat-model.md`
   if a new class of risk surfaced.

## Follow-ups (not yet runbooked)

- Automatic rotation via scheduled job that uses the KMS provider.
- Multi-key verification (ed25519:mxkeys_202604 alongside
  ed25519:mxkeys) for zero-downtime rotation. Matrix `old_verify_keys`
  semantics already model this; full automation is pending operator
  demand.
