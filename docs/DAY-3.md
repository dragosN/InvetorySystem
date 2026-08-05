# Day 3 — Notifications service (Bun/TypeScript)

## Goal

Consume `order.created` from Kafka and POST a webhook for each event, with exponential backoff on failure.

End state: `POST /orders` on the Go service triggers a webhook hit from the Bun consumer.

## Flow

```
orders-service  --publish-->  order.created (Redpanda)
                                      │
                                      ▼
                         notifications-service (kafkajs)
                                      │
                                      ▼  POST + retries
                         mock webhook (:8090/webhook)
```

## Pieces

| File | Role |
|------|------|
| `src/consumer.ts` | Subscribe to topic, parse event, call webhook |
| `src/webhook.ts` | `fetch` POST with exponential backoff |
| `src/mock-receiver.ts` | Local sink; `GET /deliveries` lists received payloads |
| `src/types.ts` | Matches Go `OrderCreatedEvent` envelope |

Retries: network errors, `429`, and `5xx`. Other `4xx` fail immediately. After max attempts, the failure is logged and the offset still commits (avoids poison-pill blocking).

## Verify

```bash
# infra already up
bun run mock-webhook          # :8090
bun run start                 # consumer

curl -sS -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"items":[{"sku":"WIDGET","quantity":1,"unit_price":1000}]}'

curl -sS http://localhost:8090/deliveries
```
