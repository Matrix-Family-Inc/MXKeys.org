Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Runbook: Notary Signing Key Rotation

## When to Run

- Planned rotation on the operator's internal schedule.
- Suspected compromise of the signing key (file permission leak,
  host intrusion, backup outside the documented chain of custody).
- Migration between key providers (`file` to `env`, `file` to KMS).
- KEK rotation: re-encrypt the existing key under a new passphrase
  without changing the server identity.

## Invariants

- The notary key ID (`ed25519:mxkeys`) stays stable. Clients
  expect the same identifier; the key bytes change while the
  label does not.
- Old keys continue to serve verification for long enough that
  in-flight federation traffic with the old key completes. The
  Matrix spec keeps old keys reachable via `old_verify_keys`
  until their original `valid_until_ts` expires.
- The transparency log records each rotation so external
  verifiers can reconstruct signer identity at any past
  `valid_until_ts`.

## Prerequisites

- Shell access to the notary host (for the `file` provider) or
  credentials for the target KMS (for `env` or future `kms`
  providers).
- A recent backup (see `docs/runbook/backup-restore.md`).
- A backup of the current signing-key file (or envelope).

## File Provider: Identity Rotation

Generates a fresh ed25519 key pair. Downstream operators who
pin the public key must be notified.

1. Stop the notary:

   ```bash
   systemctl stop mxkeys
   ```

2. Archive the current key:

   ```bash
   install -d -m 700 /var/lib/mxkeys/keys/archive
   cp -a /var/lib/mxkeys/keys/mxkeys_ed25519.key* \
      /var/lib/mxkeys/keys/archive/$(date -u +%Y%m%dT%H%M%SZ)/
   ```

3. Remove the current key material. On next start `FileProvider`
   generates a new 32-byte seed, writes it 0600 inside the 0700
   directory, and encrypts it under the configured passphrase
   when `keys.encryption.passphrase_env` is set.

   ```bash
   rm /var/lib/mxkeys/keys/mxkeys_ed25519.key*
   ```

4. Start the notary:

   ```bash
   systemctl start mxkeys
   journalctl -u mxkeys -f
   ```

   Look for `Notary signing key loaded key_id=ed25519:mxkeys
   provider=file` in the log.

5. Verify the new key:

   ```bash
   curl -fsS https://notary.example.org/_matrix/key/v2/server | jq .
   curl -fsS https://notary.example.org/_mxkeys/notary/key | jq .
   ```

   The `fingerprint` field from the second endpoint is the
   SHA-256 of the new public key. Record it in the release notes
   so downstream operators can pin it.

## File Provider: KEK Rotation (identity preserved)

Re-encrypts the existing key under a new passphrase. The public
key, and therefore the server identity, does not change. Clients
do not need to re-pin.

1. Stop the notary.

2. Stage the plaintext seed while holding the old passphrase:

   ```bash
   MXKEYS_KEY_PASSPHRASE='<old>' mxkeys-keyctl dump-seed \
     --keys-dir /var/lib/mxkeys/keys \
     > /root/seed.bin   # or any 0600 location
   ```

   Operators without `mxkeys-keyctl` can place a decrypted seed
   file named `mxkeys_ed25519.key` (0600) in an empty directory,
   then rewrite under the new passphrase via a temporary
   FileProvider.

3. Replace the envelope using the new passphrase:

   ```bash
   rm /var/lib/mxkeys/keys/mxkeys_ed25519.key.enc
   install -m 600 /root/seed.bin \
     /var/lib/mxkeys/keys/mxkeys_ed25519.key
   MXKEYS_KEY_PASSPHRASE='<new>' systemctl start mxkeys
   ```

   On first start after the swap `FileProvider` sees the
   plaintext file, re-encrypts it, and removes the plaintext.

4. Scrub `/root/seed.bin`.

5. Verify that the startup log shows `Signing key at-rest
   encryption enabled` and that the `/_mxkeys/notary/key`
   fingerprint equals the pre-rotation value.

## Env Provider

1. Generate a fresh seed offline:

   ```bash
   head -c 32 /dev/urandom | base64 -w0
   ```

2. Update the orchestrator secret (Kubernetes `Secret`, systemd
   `LoadCredential`, external vault).

3. Restart the notary so `EnvProvider` picks up the new value.

4. Run the same verification as the file flow.

## Compromise Path

If the key is believed to be in adversarial hands:

1. Execute the identity-rotation procedure immediately.
2. Publish a signed advisory listing the compromised `key_id`
   and the rotation time. Operators pinning this notary via
   `trusted_notaries` must redeploy with the new pinned public
   key.
3. Invalidate cache rows so downstream consumers re-fetch any
   keys whose perspective signatures were issued under the
   compromised key:

   ```sql
   DELETE FROM server_key_responses;
   DELETE FROM server_keys;
   ```

   Cache rehydrates as federation traffic resumes.
4. File a postmortem in `docs/` with the detection path,
   exposure window, and resulting hardening. Update
   `docs/threat-model.md` if a new class of risk surfaced.
