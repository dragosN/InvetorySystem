# Inventory System — distributed orders + notifications

Portfolio project: an orders service (Go) publishes events to Kafka (Redpanda), a notifications service (Bun/TypeScript) consumes them and sends webhooks, Redis handles caching/rate limiting, and Prometheus + Grafana provide observability.

## Current status (Day 5)

```
POST /orders (rate limited)
  → SQLite + Kafka
  → notifications (idempotent webhooks)
  → /metrics on :8080 and :8081 scraped by Prometheus
```

## Quick start

```bash
cd infra && docker compose up -d

cd ../orders-service && go run ./cmd/server

cd ../notifications-service
bun run mock-webhook   # :8090
bun run start          # consumer + :8081/metrics
```

Create an order, then open Prometheus → **Status → Targets** (both app jobs should be UP):

- http://localhost:9090
- http://localhost:8080/metrics
- http://localhost:8081/metrics

## Docs

- [Day 1 — Local infra](docs/DAY-1.md)
- [Day 2 — Orders API](docs/DAY-2.md)
- [Day 3 — Notifications](docs/DAY-3.md)
- [Day 4 — Redis](docs/DAY-4.md)
- [Day 5 — Prometheus metrics](docs/DAY-5.md)

## Roadmap

| Day | Focus |
|-----|--------|
| 1–4 | Infra, orders, notifications, Redis (done) |
| 5 | Prometheus metrics (done) |
| 6 | Grafana dashboard |
| 7 | Load test, polish, README hero shot |
