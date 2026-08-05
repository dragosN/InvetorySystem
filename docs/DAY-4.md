# Day 4 — Redis rate limiting + webhook idempotency

## Goal

Use Redis for two real jobs:

1. **Orders API** — fixed-window rate limit on `POST /orders`
2. **Notifications** — dedupe webhook delivery by `event_id` and cache delivery status

## How it fits together

```mermaid
flowchart LR
  Client[Client]
  Orders[orders-service]
  Redis[(Redis)]
  Kafka[Redpanda order.created]
  Notif[notifications-service]
  Hook[Webhook receiver]

  Client -->|"POST /orders"| Orders
  Orders -->|"1. INCR rate limit key"| Redis
  Orders -->|"2. if allowed: save + publish"| Kafka
  Kafka -->|consume| Notif
  Notif -->|"3. SET NX claim event_id"| Redis
  Notif -->|"4. if new: POST webhook"| Hook
  Notif -->|"5. cache delivery status"| Redis
```

Redis sits on both sides of the Kafka hop: it **throttles creates** before publish, and **dedupes deliveries** after consume.

### Rate limit path

```mermaid
flowchart TD
  Req["POST /orders + X-Client-Id"]
  Key["Redis key: ratelimit:orders:client:window"]
  Incr["INCR + EXPIRE on first hit"]
  Check{count greater than LIMIT?}
  Reject["429 Retry-After"]
  Allow["Create order in SQLite"]
  Pub["Publish order.created"]

  Req --> Key --> Incr --> Check
  Check -->|yes| Reject
  Check -->|no| Allow --> Pub
```

### Idempotency path

```mermaid
flowchart TD
  Evt["Consume order.created"]
  Claim["SET notify:delivery:event_id NX"]
  Exists{key already exists?}
  Skip["Skip webhook - log duplicate"]
  Post["POST webhook with retries"]
  Save["SET final status success or failed"]

  Evt --> Claim --> Exists
  Exists -->|yes| Skip
  Exists -->|no we own claim| Post --> Save
```

## Orders: fixed-window rate limit

```
POST /orders
  → middleware reads X-Client-Id (or IP)
  → Redis INCR ratelimit:orders:{client}:{window}
  → if count > LIMIT → 429 + Retry-After
  → else handler continues
```

Defaults: **5 requests / 60 seconds** per client (`RATE_LIMIT`, `RATE_WINDOW_SEC`).

Response headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`.

Note: the window is aligned to the UTC clock (`unix / 60`). A burst that crosses a minute boundary can partially reset — expected for a simple fixed window.

## Notifications: idempotency

```
consume order.created
  → SET notify:delivery:{event_id} NX  (claim)
  → if key already exists → skip webhook (log duplicate)
  → else deliver webhook, then SET final status (success|failed)
```

TTL default: 7 days (`IDEMPOTENCY_TTL_SECONDS`).

## Verify rate limit

```bash
# restart orders-service after pull
for i in 1 2 3 4 5 6; do
  curl -sS -o /tmp/o$i.json -w "%{http_code}\n" -X POST http://localhost:8080/orders \
    -H 'Content-Type: application/json' \
    -H 'X-Client-Id: demo' \
    -d '{"items":[{"sku":"RL","quantity":1,"unit_price":100}]}'
done
# expect five 201s then a 429
```

## Verify webhook dedupe

```bash
# after an order creates a delivery, re-produce the same event JSON:
cd infra
echo '<same event payload>' | docker compose exec -T redpanda rpk topic produce order.created
# consumer logs "skip duplicate event"; GET :8090/deliveries count unchanged
```
