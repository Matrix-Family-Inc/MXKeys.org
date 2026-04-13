# Transparency Verification Guide

## Overview

MXKeys implements a key transparency log with hash-chained entries and a Merkle tree. External parties can independently verify that:

1. The log is append-only (no entries removed or modified)
2. A specific entry exists in the log (inclusion proof)
3. The notary's signed tree head matches the actual state

## Concepts

### Signed Tree Head (STH)

A snapshot of the Merkle tree root signed by the notary's ed25519 key. Contains:

- `tree_size` — number of entries in the log
- `root_hash` — Merkle tree root (SHA-256, hex)
- `timestamp` — when the snapshot was taken
- `signature` — ed25519 signature over `"{tree_size}|{root_hash}|{timestamp_ms}"`
- `signer` — notary server name
- `key_id` — signing key identifier

### Public Key Discovery

The notary's public key is available at `GET /_mxkeys/notary/key`:

```json
{
  "server_name": "mxkeys.example.org",
  "key_id": "ed25519:mxkeys",
  "algorithm": "ed25519",
  "public_key": "<base64-raw>",
  "fingerprint": "<sha256-hex-of-public-key>"
}
```

The same key is published via the Matrix Key Server API at `GET /_matrix/key/v2/server`.

### Key Rotation Policy

- Key ID is fixed as `ed25519:mxkeys` for the lifetime of the key file
- Key is stored at `{keys.storage_path}/mxkeys_ed25519.key`
- On first start, if no key exists, one is generated automatically
- To rotate: stop the server, back up the old key, delete the file, restart
- After rotation, old STH signatures can no longer be verified with the new key
- Operators should archive STH snapshots before rotation for audit continuity

## Verification Steps

### 1. Verify STH Signature

```bash
mxkeys-verify -url https://mxkeys.example.org
```

This fetches the public key, fetches the STH, and verifies the ed25519 signature.

### 2. Monitor for Append-Only Growth

```bash
# First run: save baseline
mxkeys-verify -url https://mxkeys.example.org -out sth-baseline.json

# Subsequent runs: compare with previous
mxkeys-verify -url https://mxkeys.example.org -prev sth-baseline.json -out sth-latest.json
```

The verifier checks:
- Tree size never decreases (no rollback)
- Same size implies same root hash (no silent modification)
- Timestamp never goes backwards

### 3. Verify Inclusion Proof (API)

```bash
curl https://mxkeys.example.org/_mxkeys/transparency/proof?index=42
```

The response contains a Merkle audit path. Reconstruct the root hash by iterating the path from the leaf, hashing each pair with the `0x01` internal node prefix. Compare with the STH root hash.

### 4. Verify Hash Chain Integrity (API)

```bash
curl -H "Authorization: Bearer <token>" \
  "https://mxkeys.example.org/_mxkeys/transparency/verify?limit=10000"
```

Returns `{"valid": true, "entries_checked": N}` if all entries have correct hash chains.

## Threat Scenarios Covered

| Threat | Detection |
|--------|-----------|
| Entry deleted from log | STH tree size decreases; consistency check fails |
| Entry modified in log | Hash chain verification fails; Merkle root changes |
| Log silently rolled back | Previous STH shows larger tree or different root |
| Key response tampered after logging | Inclusion proof against STH will not match |
| Notary claims false log state | STH signature verification fails with published public key |
| Operator replaces log database | All consistency checks fail; STH root changes |

## Limitations

- STH is point-in-time; continuous monitoring requires periodic polling
- Consistency proof verifies structure only — does not prove completeness of the log
- No gossip protocol between verifiers (single-notary trust model)
- Key rotation invalidates old STH signatures; archive before rotating

## Monitoring Recommendations

### Polling Interval

Recommended STH polling frequency depends on operational requirements:

| Environment | Interval | Rationale |
|------------|----------|-----------|
| Production | 5 minutes | Detect tampering within SLA window |
| Staging | 1 hour | Sufficient for development verification |
| Audit / compliance | 1 minute | Tighter detection window |

### Snapshot Retention

Keep STH snapshots for at least `transparency.retention_days` (default: 365 days) to maintain full audit trail. Recommended storage:

- Local: `/var/lib/mxkeys/sth-history/` with date-stamped filenames
- Remote: Object storage (S3/GCS) for durability

### External Monitor Job (cron)

```bash
#!/bin/bash
# /etc/cron.d/mxkeys-verify
# Run every 5 minutes

MXKEYS_URL="https://mxkeys.example.org"
STH_DIR="/var/lib/mxkeys/sth-history"
LATEST="$STH_DIR/sth-latest.json"
ARCHIVE="$STH_DIR/sth-$(date +%Y%m%d-%H%M%S).json"

mkdir -p "$STH_DIR"

PREV_FLAG=""
if [ -f "$LATEST" ]; then
    PREV_FLAG="-prev $LATEST"
fi

OUTPUT=$(mxkeys-verify -url "$MXKEYS_URL" $PREV_FLAG -out "$LATEST" -json 2>&1)
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    echo "$OUTPUT" | logger -t mxkeys-verify -p local0.err
    # Alert via PagerDuty / Slack / etc
    exit $EXIT_CODE
fi

# Archive periodic snapshots (hourly)
MINUTE=$(date +%M)
if [ "$MINUTE" = "00" ]; then
    cp "$LATEST" "$ARCHIVE"
fi
```

### Exit Codes Reference

| Code | Meaning | Alert Severity |
|------|---------|----------------|
| 0 | All checks passed | — |
| 1 | Usage error (bad arguments) | — |
| 2 | Fetch error (network / HTTP) | Warning |
| 3 | Signature invalid | Critical |
| 4 | Consistency check failed | Critical |
| 5 | I/O error (file read/write) | Warning |
