# orders-service

Go REST API for orders. Persists to SQLite with a **transactional outbox**, publishes `order.created` asynchronously to Kafka, and rate-limits creates via Redis.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/orders` | Create order + outbox row (rate limited); Kafka publish is async |
| `GET` | `/orders/{id}` | Fetch order by id |
| `GET` | `/healthz` | Liveness |
| `GET` | `/metrics` | Prometheus metrics |

### Create order body

```json
{
  "items": [
    { "sku": "SKU-1", "quantity": 2, "unit_price": 1500 }
  ]
}
```

`unit_price` and response `total` are in cents.

Rate limit identity: `X-Client-Id` header, else client IP. Over limit → `429` with `Retry-After`.

## Publish path

1. Validate request
2. Begin SQLite transaction: insert `orders` + `outbox` (payload ready for Kafka)
3. Commit → return `201`
4. Background `OutboxWorker` publishes unpublished rows and marks them done

Metrics: `orders_created_total` (accepted), `orders_published_total` (Kafka OK), `orders_outbox_pending`.

## Run

Requires Redpanda + Redis from `infra/`.

```bash
cd orders-service
go run ./cmd/server
```

Env defaults:

| Variable | Default |
|----------|---------|
| `HTTP_ADDR` | `:8080` |
| `KAFKA_BROKERS` | `localhost:19092` |
| `KAFKA_TOPIC` | `order.created` |
| `SQLITE_PATH` | `data/orders.db` |
| `REDIS_ADDR` | `localhost:6379` |
| `RATE_LIMIT` | `5` |
| `RATE_WINDOW_SEC` | `60` |

## Example

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -H 'X-Client-Id: demo' \
  -d '{"items":[{"sku":"WIDGET","quantity":1,"unit_price":2500}]}'
```
