import {
  Kafka,
  logLevel,
  CompressionTypes,
  CompressionCodecs,
  type Admin,
  type Consumer,
  type EachMessagePayload,
} from "kafkajs";
import SnappyCodec from "kafkajs-snappy";
import { config } from "./config";
import { deliverWebhook } from "./webhook";
import { claimDelivery, saveDeliveryStatus } from "./idempotency";
import { isOrderCreatedEvent, type OrderCreatedEvent } from "./types";
import {
  consumerLag,
  eventsConsumed,
  eventsSkippedDuplicate,
  webhookFailure,
  webhookSuccess,
} from "./metrics";

CompressionCodecs[CompressionTypes.Snappy] = SnappyCodec;

function log(fields: Record<string, unknown>) {
  console.log(JSON.stringify({ time: new Date().toISOString(), ...fields }));
}

async function refreshConsumerLag(admin: Admin): Promise<void> {
  try {
    const topicOffsets = await admin.fetchTopicOffsets(config.kafkaTopic);
    const groupOffsets = await admin.fetchOffsets({
      groupId: config.kafkaGroupId,
      topics: [config.kafkaTopic],
    });

    const committed = new Map<number, number>();
    for (const topic of groupOffsets) {
      for (const p of topic.partitions) {
        const offset = Number.parseInt(p.offset, 10);
        if (Number.isFinite(offset) && offset >= 0) {
          committed.set(p.partition, offset);
        }
      }
    }

    for (const p of topicOffsets) {
      const high = Number.parseInt(p.high, 10);
      const current = committed.get(p.partition);
      const lag =
        Number.isFinite(high) && current !== undefined
          ? Math.max(0, high - current)
          : Number.isFinite(high)
            ? high
            : 0;
      consumerLag.set(
        { topic: config.kafkaTopic, partition: String(p.partition) },
        lag,
      );
    }
  } catch (err) {
    log({
      msg: "consumer lag refresh failed",
      error: err instanceof Error ? err.message : String(err),
    });
  }
}

export async function startConsumer(): Promise<() => Promise<void>> {
  const kafka = new Kafka({
    clientId: "notifications-service",
    brokers: config.kafkaBrokers,
    logLevel: logLevel.WARN,
  });

  const admin: Admin = kafka.admin();
  const consumer: Consumer = kafka.consumer({ groupId: config.kafkaGroupId });
  await admin.connect();
  await consumer.connect();
  await consumer.subscribe({
    topic: config.kafkaTopic,
    fromBeginning: false,
  });

  log({
    msg: "notifications-service consuming",
    brokers: config.kafkaBrokers,
    topic: config.kafkaTopic,
    group_id: config.kafkaGroupId,
    webhook_url: config.webhookUrl,
    redis_url: config.redisUrl,
  });

  const lagTimer = setInterval(() => {
    void refreshConsumerLag(admin);
  }, config.lagRefreshMs);
  void refreshConsumerLag(admin);

  await consumer.run({
    eachMessage: async ({ topic, partition, message }: EachMessagePayload) => {
      const raw = message.value?.toString("utf8");
      if (!raw) {
        log({ msg: "skip empty message", topic, partition, offset: message.offset });
        return;
      }

      let parsed: unknown;
      try {
        parsed = JSON.parse(raw);
      } catch {
        log({
          msg: "skip invalid JSON",
          topic,
          partition,
          offset: message.offset,
          raw,
        });
        return;
      }

      if (!isOrderCreatedEvent(parsed) || parsed.event_type !== "order.created") {
        log({
          msg: "skip unrecognized event",
          topic,
          partition,
          offset: message.offset,
          payload: parsed,
        });
        return;
      }

      const event: OrderCreatedEvent = parsed;
      eventsConsumed.inc();
      log({
        msg: "event consumed",
        event_id: event.event_id,
        order_id: event.order_id,
        topic,
        partition,
        offset: message.offset,
      });

      const existing = await claimDelivery(event.event_id, event.order_id);
      if (existing) {
        eventsSkippedDuplicate.inc();
        log({
          msg: "skip duplicate event (idempotent)",
          event_id: event.event_id,
          order_id: event.order_id,
          cached_status: existing.status,
          cached_at: existing.updated_at,
        });
        return;
      }

      const result = await deliverWebhook(event);
      if (result.ok) {
        webhookSuccess.inc();
        await saveDeliveryStatus({
          status: "success",
          event_id: event.event_id,
          order_id: event.order_id,
          attempts: result.attempts,
          http_status: result.status,
          updated_at: new Date().toISOString(),
        });
        log({
          msg: "webhook delivered",
          event_id: event.event_id,
          order_id: event.order_id,
          status: result.status,
          attempts: result.attempts,
        });
        return;
      }

      webhookFailure.inc();
      await saveDeliveryStatus({
        status: "failed",
        event_id: event.event_id,
        order_id: event.order_id,
        attempts: result.attempts,
        http_status: result.status,
        error: result.error,
        updated_at: new Date().toISOString(),
      });

      log({
        msg: "webhook delivery failed permanently",
        event_id: event.event_id,
        order_id: event.order_id,
        status: result.status,
        attempts: result.attempts,
        error: result.error,
      });
    },
  });

  return async () => {
    clearInterval(lagTimer);
    await consumer.disconnect();
    await admin.disconnect();
  };
}
