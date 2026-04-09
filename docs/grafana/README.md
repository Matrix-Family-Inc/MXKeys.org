# MXKeys Grafana Dashboards

This directory contains Grafana dashboard assets for MXKeys.

## Files

- `mxkeys-overview.json`
- `mxkeys-federation.json`

## Requirements

- MXKeys metrics endpoint: `GET /_mxkeys/metrics`
- Prometheus scrape configuration pointing to that path

Example scrape job:

```yaml
scrape_configs:
  - job_name: mxkeys
    static_configs:
      - targets: ['mxkeys:8448']
    metrics_path: /_mxkeys/metrics
```

## Import

1. Open Grafana.
2. Import one of the JSON dashboards from this directory.
3. Bind the dashboard to your Prometheus data source.

Alerting rules are maintained separately in `docs/prometheus-alerts.yaml`.
