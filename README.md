# Inventory System — distributed orders + notifications

Portfolio project: an orders service (Go) publishes events to Kafka (Redpanda), a notifications service (Bun/TypeScript) consumes them and sends webhooks, Redis handles caching/rate limiting, and Prometheus + Grafana provide observability.

## Current status (Day 3)

End-to-end event path works:

```
POST /orders  →  SQLite + Kafka order.created  →  notifications-service  →  webhook POST
```

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2 (or Colima)
- Go 1.22+
- [Bun](https://bun.sh)

## Quick start

### 1. Infra

```bash
cd infra
docker compose up -d
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

### 3. Notifications + mock webhook

```bash
cd notifications-service
bun install

# Terminal A
bun run mock-webhook

# Terminal B
bun run start
```

### 4. Create an order and confirm webhook

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"sku":"WIDGET","quantity":2,"unit_price":1500}]}'

curl -sS http://localhost:8090/deliveries
```

## Repo layout

```
orders-service/          # Go API + Kafka producer (Day 2)
notifications-service/   # Bun consumer + webhooks (Day 3)
infra/                   # docker-compose, Prometheus, Grafana
docs/                    # Day write-ups
scripts/                 # smoke tests / load tests
```

## Roadmap

| Day | Focus |
|-----|--------|
| 1 | Infra + Kafka smoke test (done) |
| 2 | Orders REST API + `order.created` publish (done) |
| 3 | Notifications consumer + webhook delivery (done) |
| 4 | Redis rate limiting + idempotency |
| 5 | Prometheus metrics on both services |
| 6 | Grafana dashboard |
| 7 | Load test, polish, README hero shot |
