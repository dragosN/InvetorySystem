function envOr(key: string, fallback: string): string {
  return process.env[key]?.trim() || fallback;
}

function envInt(key: string, fallback: number): number {
  const raw = process.env[key];
  if (!raw) return fallback;
  const n = Number.parseInt(raw, 10);
  return Number.isFinite(n) ? n : fallback;
}

export const config = {
  kafkaBrokers: envOr("KAFKA_BROKERS", "localhost:19092").split(","),
  kafkaTopic: envOr("KAFKA_TOPIC", "order.created"),
  kafkaGroupId: envOr("KAFKA_GROUP_ID", "notifications-service"),
  webhookUrl: envOr("WEBHOOK_URL", "http://localhost:8090/webhook"),
  webhookMaxAttempts: envInt("WEBHOOK_MAX_ATTEMPTS", 5),
  webhookBaseDelayMs: envInt("WEBHOOK_BASE_DELAY_MS", 200),
  webhookTimeoutMs: envInt("WEBHOOK_TIMEOUT_MS", 5000),
  mockReceiverPort: envInt("MOCK_WEBHOOK_PORT", 8090),
} as const;
