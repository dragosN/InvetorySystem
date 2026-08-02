# Inventory System — distributed orders + notifications

Portfolio project: an orders service (Go) publishes events to Kafka (Redpanda), a notifications service (Bun/TypeScript) consumes them and sends webhooks, Redis handles caching/rate limiting, and Prometheus + Grafana provide observability.

## Current status (Day 2)

Infra is up, and **orders-service** exposes a REST API that persists orders to SQLite and publishes `order.created` to Redpanda.

```
curl POST /orders  →  SQLite  +  Kafka topic order.created
curl GET  /orders/{id}
```

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2 (or Colima)
- Go 1.22+
- [Bun](https://bun.sh) (Day 3+)

## Quick start

### 1. Infra

```bash
cd infra
docker compose up -d
docker compose ps
```

| Service     | URL                         | Creds        |
|-------------|-----------------------------|--------------|
| Prometheus  | http://localhost:9090       | —            |
| Grafana     | http://localhost:3000       | admin/admin  |
| Redpanda    | Kafka on `localhost:19092`  | —            |
| Redis       | `localhost:6379`            | —            |

### 2. Orders API

```bash
cd orders-service
go run ./cmd/server
```

Create an order:

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"sku":"WIDGET","quantity":2,"unit_price":1500}]}'
```

Fetch it back (use the `id` from the response):

```bash
curl -sS http://localhost:8080/orders/<order-id>
```

Confirm the Kafka event:

```bash
cd infra
docker compose exec redpanda rpk topic consume order.created --offset start --num 1
```

### Kafka smoke (infra only)

```bash
./scripts/smoke-kafka.sh
```

## Repo layout

```
orders-service/          # Go API + Kafka producer (Day 2 done)
notifications-service/   # Bun consumer + webhooks (Day 3+)
infra/                   # docker-compose, Prometheus, Grafana
scripts/                 # smoke tests / load tests
```

## Roadmap

| Day | Focus |
|-----|--------|
| 1 | Infra + Kafka smoke test (done) |
| 2 | Orders REST API + `order.created` publish (done) |
| 3 | Notifications consumer + webhook delivery |
| 4 | Redis rate limiting + idempotency |
| 5 | Prometheus metrics on both services |
| 6 | Grafana dashboard |
| 7 | Load test, polish, README hero shot |
