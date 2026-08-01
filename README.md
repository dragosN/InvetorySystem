# Inventory System — distributed orders + notifications

Portfolio project: an orders service (Go) publishes events to Kafka (Redpanda), a notifications service (Bun/TypeScript) consumes them and sends webhooks, Redis handles caching/rate limiting, and Prometheus + Grafana provide observability.

## Day 1 status

Local infrastructure is up: Redpanda, Redis, Prometheus, Grafana. App services are stubs until Days 2–3.

```
Host CLI (rpk / smoke script)
        │
        ▼
┌─────────────────────────────────────────────┐
│  Docker Compose (infra/)                    │
│  Redpanda :19092  Redis :6379               │
│  Prometheus :9090 Grafana :3000             │
└─────────────────────────────────────────────┘
```

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2
- (Optional later) Go 1.22+, [Bun](https://bun.sh)

## Quick start

```bash
cd infra
docker compose up -d
docker compose ps
```

Verify Redis:

```bash
docker compose exec redis redis-cli ping
# PONG
```

UIs:

| Service     | URL                         | Creds        |
|-------------|-----------------------------|--------------|
| Prometheus  | http://localhost:9090       | —            |
| Grafana     | http://localhost:3000       | admin/admin  |
| Redpanda    | Kafka on `localhost:19092`  | —            |

## Kafka smoke test

From the repo root:

```bash
chmod +x scripts/smoke-kafka.sh
./scripts/smoke-kafka.sh
```

Or manually with `rpk` inside the Redpanda container:

```bash
cd infra

# Create topic
docker compose exec redpanda rpk topic create order.created --partitions 1 --replicas 1

# Produce
echo '{"event_id":"smoke-001","order_id":"ord-001"}' \
  | docker compose exec -T redpanda rpk topic produce order.created

# Consume
docker compose exec redpanda rpk topic consume order.created --offset start --num 1
```

From the host (if you have [`rpk`](https://docs.redpanda.com/current/get-started/rpk-install/) installed):

```bash
rpk topic list -X brokers=localhost:19092
rpk topic produce order.created -X brokers=localhost:19092
rpk topic consume order.created -X brokers=localhost:19092 --offset start
```

## Repo layout

```
orders-service/          # Go API + Kafka producer (Day 2+)
notifications-service/   # Bun consumer + webhooks (Day 3+)
infra/                   # docker-compose, Prometheus, Grafana
scripts/                 # smoke tests / load tests
```

## Roadmap

| Day | Focus |
|-----|--------|
| 1 | Infra + Kafka smoke test (done) |
| 2 | Orders REST API + `order.created` publish |
| 3 | Notifications consumer + webhook delivery |
| 4 | Redis rate limiting + idempotency |
| 5 | Prometheus metrics on both services |
| 6 | Grafana dashboard |
| 7 | Load test, polish, README hero shot |
