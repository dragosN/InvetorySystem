# notifications-service

Bun + TypeScript consumer for `order.created`. Delivers a webhook for each event, with exponential backoff retries.

## Run

Requires Redpanda (`localhost:19092`) and a webhook target.

```bash
# Terminal A — mock receiver (records POSTs)
bun run mock-webhook

# Terminal B — consumer
bun install
bun run start
```

Then create an order via orders-service; the mock receiver should log the delivery. Inspect:

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

## Layout

```
src/
  index.ts           # start consumer + graceful shutdown
  consumer.ts        # kafkajs consumer loop
  webhook.ts         # POST + exponential backoff
  mock-receiver.ts   # local webhook sink for demos
  types.ts / config.ts
```
