# notifications-service

Bun + TypeScript consumer for `order.created`. Delivers webhooks with exponential backoff, and dedupes by `event_id` in Redis.

## Run

Requires Redpanda + Redis (`localhost:19092`, `localhost:6379`) and a webhook target.

```bash
# Terminal A — mock receiver (records POSTs)
bun run mock-webhook

# Terminal B — consumer
bun install
bun run start
```

Inspect deliveries:

```bash
curl -sS http://localhost:8090/deliveries | jq
```

## Env

| Variable | Default |
|----------|---------|
| `KAFKA_BROKERS` | `localhost:19092` |
| `KAFKA_TOPIC` | `order.created` |
| `KAFKA_GROUP_ID` | `notifications-service` |
| `WEBHOOK_URL` | `http://localhost:8090/webhook` |
| `WEBHOOK_MAX_ATTEMPTS` | `5` |
| `WEBHOOK_BASE_DELAY_MS` | `200` |
| `WEBHOOK_TIMEOUT_MS` | `5000` |
| `MOCK_WEBHOOK_PORT` | `8090` |
| `REDIS_URL` | `redis://localhost:6379` |
| `IDEMPOTENCY_KEY_PREFIX` | `notify:delivery` |
| `IDEMPOTENCY_TTL_SECONDS` | `604800` (7d) |

## Layout

```
src/
  index.ts           # start consumer + graceful shutdown
  consumer.ts        # kafkajs consumer loop + idempotency
  idempotency.ts     # Redis claim + delivery status cache
  redis.ts           # ioredis client
  webhook.ts         # POST + exponential backoff
  mock-receiver.ts   # local webhook sink for demos
  types.ts / config.ts
```
