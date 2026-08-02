# orders-service

Go REST API for orders. Creates orders in SQLite and publishes `order.created` events to Kafka (Redpanda).

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/orders` | Create order + publish event |
| `GET` | `/orders/{id}` | Fetch order by id |
| `GET` | `/healthz` | Liveness |

### Create order body

```json
{
  "items": [
    { "sku": "SKU-1", "quantity": 2, "unit_price": 1500 }
  ]
}
```

`unit_price` and response `total` are in cents.

## Run

Requires Redpanda from `infra/` (`localhost:19092`).

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

## Example

```bash
curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"sku":"WIDGET","quantity":1,"unit_price":2500}]}' | jq

# Watch the event
cd ../infra
docker compose exec redpanda rpk topic consume order.created --offset end --num 1
```
