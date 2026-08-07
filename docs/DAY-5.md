# Day 5 — Prometheus metrics

## Goal

Expose `/metrics` on both services and scrape them from Prometheus so counters/histograms update live as you hit the API.

## Metrics

### orders-service (`:8080/metrics`)

| Metric | Type | Meaning |
|--------|------|---------|
| `orders_http_requests_total` | counter | Requests by method/path/status |
| `orders_http_request_duration_seconds` | histogram | Latency |
| `orders_created_total` | counter | Successful create (SQLite + outbox commit) |
| `orders_published_total` | counter | Outbox rows published to Kafka |
| `orders_outbox_pending` | gauge | Unpublished outbox rows |
| `orders_kafka_publish_errors_total` | counter | Kafka publish failures |
| `orders_rate_limit_rejected_total` | counter | `429` from rate limiter |

### notifications-service (`:8081/metrics`)

| Metric | Type | Meaning |
|--------|------|---------|
| `notifications_events_consumed_total` | counter | Valid events processed |
| `notifications_events_skipped_duplicate_total` | counter | Idempotent skips |
| `notifications_webhook_success_total` | counter | Webhook OK |
| `notifications_webhook_failure_total` | counter | Webhook failed after retries |
| `notifications_webhook_retries_total` | counter | Retry attempts |
| `notifications_consumer_lag` | gauge | High watermark − committed offset (per partition) |

## Scrape config

Prometheus scrapes host apps via `host.docker.internal` (Colima/Docker Desktop). Compose sets `extra_hosts: host.docker.internal:host-gateway` on the Prometheus service.

```yaml
- job_name: orders-service
  static_configs: [{ targets: ["host.docker.internal:8080"] }]
- job_name: notifications-service
  static_configs: [{ targets: ["host.docker.internal:8081"] }]
```

## Verify

```bash
# after restarting services + prometheus
curl -sS http://localhost:8080/metrics | grep orders_created
curl -sS http://localhost:8081/metrics | grep notifications_events

# Prometheus UI → Graph
#   orders_created_total
#   rate(orders_http_requests_total[1m])
#   notifications_webhook_success_total
#   notifications_consumer_lag
# Status → Targets should show both jobs UP
```
