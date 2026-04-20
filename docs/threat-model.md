Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# MXKeys Threat Model

## Scope

This document defines the threat model for MXKeys as a Matrix federation key-notary service.
It covers key fetch/verify/sign/query behavior, cache/storage paths, and operational endpoints.

## Security Objectives

- Preserve integrity of key material returned by the notary.
- Prevent acceptance of forged or substituted server keys.
- Ensure deterministic validation and rejection behavior.
- Maintain service availability under hostile traffic.
- Limit blast radius if a notary node is compromised.

## Primary Assets

- MXKeys signing private key (`ed25519`).
- Verified server key responses and signatures.
- Trust policy configuration and fallback notary pins.
- PostgreSQL key cache and transparency/analytics data.
- API availability for federation key queries.

## Trust Boundaries

- Public federation network -> MXKeys HTTP API.
- MXKeys -> remote homeservers / fallback notaries.
- MXKeys process -> local key storage and database.
- Cluster node <-> cluster node (if enabled).

## Threat Actors

- Malicious homeserver operator.
- External attacker on public network path.
- Resource-exhaustion attacker (DoS).
- Compromised infrastructure/operator account.

## Threats and Controls

| Threat | Attack Pattern | Current Controls | Detection | Residual Risk |
|---|---|---|---|---|
| Malicious homeserver | Publishes invalid or misleading key response | Canonical JSON checks (continuously fuzz-tested), signature verification, server name validation, key/signature length validation | Request rejection metrics, fetch failure metrics, logs | Medium (depends on upstream correctness and operator policy) |
| Key substitution | MITM/upstream substitution of `server_keys` | Required signature validation, optional pinned notary verification, trust policy checks | Query failures with reason codes, policy violation counters | Medium (reduced with strict trust policy) |
| DNS/SRV poisoning (service-scope) | Malicious DNS/SRV answer redirects fetch path to attacker-controlled endpoint | TLS validation, signature/self-signature verification, resolved-IP SSRF guard (IPs pinned for the dial), optional strict trust policy (`requireWellKnown`, `requireValidTLS`), fallback pinning | Upstream failure reasons, policy violations, resolver/fetch logs | Medium (network naming layer remains external trust input) |
| Replay | Reuse of stale but previously valid responses | `valid_until_ts` checks, cache expiry logic, stale memory-cache restrictions | Expired key cleanup metrics, key-age analytics | Low-Medium (window limited by validity and policy) |
| Cache poisoning attempts | Attempt to inject invalid key material into memory/DB cache via malformed or forged responses | Verify-before-store flow, strict payload/signature checks, server-name consistency validation | Rejection counters, anomaly metrics, query/fetch error logs | Low-Medium (depends on correctness of verification path) |
| DoS vectors | Oversized body, high-rate requests, upstream fanout abuse, unique-IP floods | Body size limits, per-IP rate limiting with O(1) LRU eviction, max servers per query, concurrency limits, singleflight dedup | Rate-limit and rejection metrics, latency/error alerts | Medium (availability still bounded by host/network capacity) |
| Notary compromise | Theft/misuse of MXKeys signing key or host access | Key-provider abstraction (file 0600/0700, env, KMS stub), backup/rotation runbook, isolated storage path | Operational logs, `docs/runbook/key-rotation.md` | High (requires incident response and trust reset) |
| Cluster MAC forgery or replay | Attacker with network access crafts or replays cluster messages | HMAC-SHA256 over canonical JSON (no structural-ambiguity collisions), 5-minute skew window, signature-replay cache, minimum-length shared-secret guard, placeholder-secret guard | Cluster message rejection counters | Medium (mitigated further by network-layer overlay) |
| Cluster restart with stale state | Raft cluster loses committed entries on restart | Write-ahead log with CRC32C per record (Castagnoli, fuzz-tested), group-commit + bounded queue backpressure, atomic snapshot persistence, InstallSnapshot RPC for lagging followers | WAL replay logs, snapshot-age metrics | Low (covered by ADR-0001 production SLA) |
| Schema drift across versions | `CREATE IF NOT EXISTS` loses track of evolving schema | Versioned migrations runner with per-migration transaction + `schema_migrations` bookkeeping | Startup log entries per applied migration | Low |
| WAL on-disk tampering | Attacker with filesystem access modifies `raft.wal` | CRC32C detects random/single-bit corruption and short tails; format magic `MXKS_WAL_v2` detects cross-version attempts | ErrWALCorrupt on replay, per-file attestation recommended | Medium; CRC is not a MAC. Targeted-tampering resistance requires a signed-envelope layer (follow-up) |

## Additional Abuse Cases

### Malformed JSON / Protocol Abuse

- Threat: parser confusion, trailing payload injection.
- Control: strict JSON decode with trailing-token rejection and matrix-compatible error codes.

### Discovery Manipulation

- Threat: abuse of `.well-known` / SRV resolution paths.
- Control: deterministic resolver behavior, hostname/IP validation, optional strict policy requirements.

### DNS/SRV Poisoning (Service-Scope Assumptions)

- Threat: resolver receives poisoned DNS/SRV data and reaches malicious endpoints.
- Control: cryptographic key/signature verification remains mandatory; optional strict trust policies can tighten resolution acceptance.
- Assumption: DNS integrity is not guaranteed by service alone; service relies on verification controls to detect malicious key material.

### Cache Poisoning Attempts

- Threat: attacker attempts to persist forged key material in cache/storage.
- Control: MXKeys stores remote key material only after validation succeeds; invalid responses are rejected before persistence.
- Detection: request rejection metrics, fetch failure metrics, and anomaly counters.

### Fallback Notary Trust Drift

- Threat: weak fallback source introduces false trust.
- Control: fallback pinning and signature verification; policy can require stricter trust conditions.

## Operational Security Requirements

- Keep signing key directory and file permissions hardened.
- Enforce secure backups and key restore drills.
- Use TLS with valid certificates on public endpoints.
- Monitor rejection, failure, latency, and anomaly metrics continuously.
- Treat key compromise as a security incident requiring immediate rotation and trust cache invalidation.

## Out of Scope

- Endpoint authentication/authorization for private admin planes (not part of current public API model).
- Full BGP/DNSSEC-level routing threats outside host/service controls.

## Review Triggers

Update this threat model when:

- key verification logic changes,
- trust policy semantics change,
- cluster consensus behavior changes,
- new public endpoints are introduced,
- incident postmortems identify new classes of risk.
