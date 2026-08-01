# notifications-service

Bun + TypeScript service that consumes `order.created` events and delivers webhooks.

## Status

Day 1 skeleton only. Day 3 will add:

- Kafka consumer for `order.created`
- Simulated webhook delivery
- Retry with exponential backoff

## Run (after Day 3)

```bash
bun install
bun run start
```
