import { Kafka, logLevel, type EachMessagePayload } from "kafkajs";
import { config } from "./config";
import { deliverWebhook } from "./webhook";
import { isOrderCreatedEvent, type OrderCreatedEvent } from "./types";

function log(fields: Record<string, unknown>) {
  console.log(JSON.stringify({ time: new Date().toISOString(), ...fields }));
}

export async function startConsumer(): Promise<() => Promise<void>> {
  const kafka = new Kafka({
    clientId: "notifications-service",
    brokers: config.kafkaBrokers,
    logLevel: logLevel.WARN,
  });

  const consumer = kafka.consumer({ groupId: config.kafkaGroupId });
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
  });

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

      // Ignore Day-1 smoke payloads that lack the full event envelope
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
      log({
        msg: "event consumed",
        event_id: event.event_id,
        order_id: event.order_id,
        topic,
        partition,
        offset: message.offset,
      });

      const result = await deliverWebhook(event);
      if (result.ok) {
        log({
          msg: "webhook delivered",
          event_id: event.event_id,
          order_id: event.order_id,
          status: result.status,
          attempts: result.attempts,
        });
        return;
      }

      // Commit progresses anyway so a poison event does not block the group.
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
    await consumer.disconnect();
  };
}
