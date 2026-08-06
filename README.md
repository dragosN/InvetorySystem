# Inventory System — distributed orders + notifications

Portfolio project: an orders service (Go) publishes events to Kafka (Redpanda), a notifications service (Bun/TypeScript) consumes them and sends webhooks, Redis handles caching/rate limiting, and Prometheus + Grafana provide observability.

## Current status (Day 6)

End-to-end path with a live Grafana dashboard:

```
POST /orders → Kafka → webhooks
     ↓              ↓
  :8080/metrics  :8081/metrics  →  Prometheus  →  Grafana
```

Dashboard: http://localhost:3000/d/ecommerce-observability (`admin` / `admin`)

## Quick start

```bash
cd infra && docker compose up -d

cd ../orders-service && go run ./cmd/server

cd ../notifications-service
bun run mock-webhook   # :8090
bun run start          # consumer + :8081/metrics
```

## Docs

- [Day 1 — Local infra](docs/DAY-1.md)
- [Day 2 — Orders API](docs/DAY-2.md)
- [Day 3 — Notifications](docs/DAY-3.md)
- [Day 4 — Redis](docs/DAY-4.md)
- [Day 5 — Prometheus metrics](docs/DAY-5.md)
- [Day 6 — Grafana dashboard](docs/DAY-6.md)

## Roadmap

| Day | Focus |
|-----|--------|
| 1–5 | Infra through Prometheus (done) |
| 6 | Grafana dashboard (done) |
| 7 | Load test, polish, README hero shot |
