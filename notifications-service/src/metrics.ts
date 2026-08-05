import client from "prom-client";

export const registry = new client.Registry();
client.collectDefaultMetrics({ register: registry, prefix: "notifications_" });

export const eventsConsumed = new client.Counter({
  name: "notifications_events_consumed_total",
  help: "order.created events consumed from Kafka",
  registers: [registry],
});

export const eventsSkippedDuplicate = new client.Counter({
  name: "notifications_events_skipped_duplicate_total",
  help: "Events skipped due to Redis idempotency",
  registers: [registry],
});

export const webhookSuccess = new client.Counter({
  name: "notifications_webhook_success_total",
  help: "Successful webhook deliveries",
  registers: [registry],
});

export const webhookFailure = new client.Counter({
  name: "notifications_webhook_failure_total",
  help: "Webhook deliveries that failed after all retries",
  registers: [registry],
});

export const webhookRetries = new client.Counter({
  name: "notifications_webhook_retries_total",
  help: "Webhook retry attempts (not counting the first try)",
  registers: [registry],
});

export const consumerLag = new client.Gauge({
  name: "notifications_consumer_lag",
  help: "Consumer lag (high watermark - committed offset) per partition",
  labelNames: ["topic", "partition"],
  registers: [registry],
});
