Project: MXKeys
Company: Matrix Family Inc. (https://matrix.family)
Maintainer: Brabus
Contact: dev@matrix.family
Date: Mon Apr 20 2026 UTC
Status: Updated

# Monitoring Guide

## Prometheus Metrics

MXKeys exposes metrics at `GET /_mxkeys/metrics` (gated by the admin access token when `security.admin_access_token` is configured).

### Key Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `mxkeys_http_requests_total` | Counter | HTTP requests by method, route, status |
| `mxkeys_http_request_duration_seconds` | Histogram | Request latency by method, route |
| `mxkeys_in_flight_requests` | Gauge | Currently processing requests |
| `mxkeys_key_queries_total` | Counter | Key query operations by status |
| `mxkeys_rate_limited_requests_total` | Counter | Rate limited requests by limiter (global/query) |
| `mxkeys_keys_cache_hits_total` | Counter | Cache hits by type (memory/database) |
| `mxkeys_keys_cache_misses_total` | Counter | Cache misses by type |
| `mxkeys_keys_fetch_attempts_total` | Counter | Upstream fetch attempts by status, source |
| `mxkeys_keys_fetch_duration_seconds` | Histogram | Upstream fetch latency |
| `mxkeys_upstream_failures_total` | Counter | Upstream failures by reason |
| `mxkeys_circuit_breaker_servers` | Gauge | Tracked servers by state (closed/open/half_open) |
| `mxkeys_circuit_breaker_trips_total` | Counter | Circuit breaker activations |
| `mxkeys_circuit_breaker_recoveries_total` | Counter | Circuit breaker recoveries |
| `mxkeys_transparency_entries_total` | Counter | Transparency log entries written |
| `mxkeys_transparency_anomalies_total` | Counter | Anomalies detected |
| `go_goroutines` | Gauge | Active goroutines |
| `go_memstats_heap_alloc_bytes` | Gauge | Heap memory in use |
| `go_threads` | Gauge | OS threads |

### Scrape Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: mxkeys
    scheme: https
    authorization:
      credentials: <admin-access-token>
    static_configs:
      - targets: ['mxkeys.example.org:8448']
```

## Alerting Rules

See `docs/prometheus-alerts.yaml` for ready-to-use alert rules covering:

- High error rate (5xx > 5%)
- High latency (p95 > 5s)
- Upstream failures
- Circuit breaker trips and open breakers
- Rate limiting spikes
- Key fetch failure rate
- Goroutine leaks
- Memory pressure
- Service down
- Low cache hit rate
- Request rejections

## External Transparency Monitoring

### Overview

The `mxkeys-verify` CLI tool enables external verification of the transparency log independent of the server operator.

### Setup with node_exporter

Use `scripts/mxkeys-verify-exporter.sh` to export verification results as Prometheus metrics via node_exporter textfile collector:

```bash
# Install
cp scripts/mxkeys-verify-exporter.sh /usr/local/bin/
cp cmd/mxkeys-verify/mxkeys-verify /usr/local/bin/

# Configure
export MXKEYS_URL=https://mxkeys.example.org
export MXKEYS_EXPECTED_FINGERPRINT=<pinned-fingerprint>
export TEXTFILE_DIR=/var/lib/node_exporter/textfile

# Run via cron (every 5 minutes)
*/5 * * * * /usr/local/bin/mxkeys-verify-exporter.sh
```

Exported metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `mxkeys_sth_verify_success` | Gauge | 1 if verification passed, 0 otherwise |
| `mxkeys_sth_tree_size` | Gauge | Current log tree size |
| `mxkeys_sth_trust_level` | Gauge | Achieved trust level (1-3) |
| `mxkeys_sth_verify_duration_ms` | Gauge | Verification duration |

### Alert Rules for STH Monitoring

```yaml
- alert: MXKeysSTHVerificationFailed
  expr: mxkeys_sth_verify_success == 0
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "MXKeys transparency verification failing"

- alert: MXKeysSTHTreeSizeDecreased
  expr: delta(mxkeys_sth_tree_size[1h]) < 0
  labels:
    severity: critical
  annotations:
    summary: "MXKeys transparency log tree size decreased"

- alert: MXKeysSTHTrustLevelDegraded
  expr: mxkeys_sth_trust_level < 3
  for: 30m
  labels:
    severity: warning
  annotations:
    summary: "MXKeys transparency verification trust level below origin trust"
```

## Operational Endpoints

| Endpoint | Auth | Description |
|----------|------|-------------|
| `GET /_mxkeys/health` | None | Liveness probe |
| `GET /_mxkeys/live` | None | Process alive |
| `GET /_mxkeys/ready` | None | Database + key readiness |
| `GET /_mxkeys/status` | Token | Detailed status with cache, DB, cluster stats |
| `GET /_mxkeys/metrics` | Token | Prometheus metrics |
| `GET /_mxkeys/circuits` | Token | Circuit breaker states |
| `GET /_mxkeys/notary/key` | None | Public key for STH verification |
| `GET /_mxkeys/transparency/signed-head` | None | Signed Merkle tree head |
