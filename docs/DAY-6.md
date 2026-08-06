# Day 6 — Grafana dashboard

## Goal

One Grafana dashboard that tells the story of the system under load: orders flowing in, consumer lag, webhook health, and API latency.

## Setup

Prometheus is already provisioned as the default Grafana datasource (`uid: prometheus`).

Dashboard is file-provisioned:

```
infra/grafana/
├── dashboards/ecommerce-observability.json
└── provisioning/
    ├── datasources/prometheus.yml
    └── dashboards/provider.yml
```

Compose mounts both folders into the Grafana container. After `docker compose up -d` (or recreate Grafana), open:

**http://localhost:3000/d/ecommerce-observability**  
Login: `admin` / `admin`

If login fails (password changed in the persistent Grafana volume):

```bash
cd infra
docker compose exec grafana grafana cli admin reset-admin-password admin
```

## Panels

| Panel | Query idea |
|-------|------------|
| Order throughput (stat) | `rate(orders_created_total[1m])` |
| Consumer lag (stat) | `sum(notifications_consumer_lag)` |
| Webhook success/failure (stats) | `rate(..._success_total[1m])` / `rate(..._failure_total[1m])` |
| Event throughput (timeseries) | orders created/sec vs events consumed/sec |
| Consumer lag by partition | `notifications_consumer_lag` |
| Webhook success / failure / retries | rates over 1m |
| API latency p50 / p95 | `histogram_quantile` on `orders_http_request_duration_seconds_bucket` for `POST /orders` |

Refresh: **5s**. Time range default: last 15 minutes.

## Verify

```bash
cd infra
docker compose up -d grafana
# apps must be running so Prometheus has scrapes (Day 5)
open http://localhost:3000/d/ecommerce-observability
```

Generate traffic (with orders + notifications up):

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -H 'X-Client-Id: grafana-demo' \
  -d '{"items":[{"sku":"G","quantity":1,"unit_price":100}]}'
```

Panels should move within ~15–30s (Prometheus scrape interval).

## Portfolio tip

Screenshot this dashboard after a short load burst (Day 7) — use it as the README hero image.
