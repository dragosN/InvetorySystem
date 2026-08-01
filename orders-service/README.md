# orders-service

Go service that owns the orders API and publishes domain events to Kafka (Redpanda).

## Status

Day 1 skeleton only. Day 2 will add:

- `POST /orders` — create order
- `GET /orders/:id` — fetch order
- Publish `order.created` to Kafka on create

## Run (after Day 2)

```bash
go run ./cmd/server
```
