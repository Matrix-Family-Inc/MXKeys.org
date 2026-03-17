# MXKeys Grafana Dashboards

Grafana dashboards for MXKeys monitoring and operations.

## Available Dashboards

### Overview Dashboard (`mxkeys-overview.json`)

General operational overview:
- Success rate
- Requests per second
- P99 latency
- In-flight requests
- Request rate by status
- Latency percentiles (p50, p95, p99)
- Key fetches by source
- Cache hits vs misses
- Rate limited requests
- Upstream failures
- Goroutines and heap memory

### Federation Health Dashboard (`mxkeys-federation.json`)

Federation-specific metrics:
- Unique servers tracked
- Anomalous servers count
- Key rotations (1h)
- Policy violations (1h)
- Upstream fetch duration by server
- Anomalies by type
- Upstream failures by reason
- Circuit breaker state changes
- Transparency log events

## Installation

### Option 1: Import via Grafana UI

1. Open Grafana → Dashboards → Import
2. Upload the JSON file or paste its content
3. Select your Prometheus data source
4. Click Import

### Option 2: Provisioning

Add to `grafana/provisioning/dashboards/mxkeys.yaml`:

```yaml
apiVersion: 1
providers:
  - name: MXKeys
    folder: MXKeys
    type: file
    options:
      path: /var/lib/grafana/dashboards/mxkeys
```

Copy dashboard JSON files to `/var/lib/grafana/dashboards/mxkeys/`.

## Data Source

Both dashboards expect a Prometheus data source named `${DS_PROMETHEUS}`.

To use a different data source:
1. Import the dashboard
2. Go to Dashboard Settings → Variables
3. Update the `DS_PROMETHEUS` variable

## Metrics Requirements

Ensure MXKeys is configured to expose metrics:

```yaml
# config.yaml
metrics:
  enabled: true
  path: /_mxkeys/metrics
```

Configure Prometheus to scrape MXKeys:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: mxkeys
    static_configs:
      - targets: ['mxkeys:8448']
    metrics_path: /_mxkeys/metrics
```

## Customization

### Time Ranges

- Overview: Default 1h, recommended for real-time monitoring
- Federation: Default 6h, better for trend analysis

### Thresholds

Adjust panel thresholds based on your environment:
- Success rate: yellow < 99%, red < 90%
- Latency: yellow > 500ms, red > 1s
- Goroutines: yellow > 100, red > 500

## Alerting

See `docs/deployment.md` for recommended Prometheus alerting rules.

Key alerts to configure:
- `MXKeysHighLatency` — P99 latency > 1s
- `MXKeysHighErrorRate` — Error rate > 1%
- `MXKeysCircuitBreakerOpen` — Circuit breaker opened
- `MXKeysPolicyViolation` — Trust policy violation detected
