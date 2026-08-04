# Inventory System — distributed orders + notifications

Portfolio project: an orders service (Go) publishes events to Kafka (Redpanda), a notifications service (Bun/TypeScript) consumes them and sends webhooks, Redis handles caching/rate limiting, and Prometheus + Grafana provide observability.

## Current status (Day 4)

```
POST /orders (Redis rate limit)
  → SQLite + Kafka order.created
  → notifications-service (Redis idempotency by event_id)
  → webhook POST (once per event)
```

## Prerequisites

- Docker / Colima with Compose v2
- Go 1.22+
- Bun

## Quick start

```bash
cd infra && docker compose up -d

cd ../orders-service && go run ./cmd/server

cd ../notifications-service
bun run mock-webhook   # terminal A
bun run start          # terminal B
```

Create an order:

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -H 'X-Client-Id: demo' \
  -d '{"items":[{"sku":"WIDGET","quantity":2,"unit_price":1500}]}'

curl -sS http://localhost:8090/deliveries
```

Rate limit demo (default 5/min per client): sixth request should be `429`.

## Docs

- [Day 2 — Orders API](docs/DAY-2.md)
- [Day 3 — Notifications](docs/DAY-3.md)
- [Day 4 — Redis](docs/DAY-4.md)

## Roadmap

| Day | Focus |
|-----|--------|
| 1–3 | Infra, orders API, notifications (done) |
| 4 | Redis rate limiting + idempotency (done) |
| 5 | Prometheus metrics on both services |
| 6 | Grafana dashboard |
| 7 | Load test, polish, README hero shot |
