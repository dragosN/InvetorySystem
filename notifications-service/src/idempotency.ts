import { getRedis } from "./redis";
import { config } from "./config";

export type DeliveryStatus = {
  status: "success" | "failed" | "pending";
  event_id: string;
  order_id: string;
  attempts?: number;
  http_status?: number;
  error?: string;
  updated_at: string;
};

function key(eventId: string): string {
  return `${config.idempotencyKeyPrefix}:${eventId}`;
}

/**
 * Atomically claim an event for delivery (SET NX).
 * Returns existing status if already claimed/processed, or null if we own it.
 */
export async function claimDelivery(
  eventId: string,
  orderId: string,
): Promise<DeliveryStatus | null> {
  const redis = getRedis();
  const pending: DeliveryStatus = {
    status: "pending",
    event_id: eventId,
    order_id: orderId,
    updated_at: new Date().toISOString(),
  };

  const claimed = await redis.set(
    key(eventId),
    JSON.stringify(pending),
    "EX",
    config.idempotencyTTLSeconds,
    "NX",
  );

  if (claimed === "OK") {
    return null; // we own the claim — proceed to deliver
  }

  const raw = await redis.get(key(eventId));
  if (!raw) {
    // Rare race: key expired between SET NX miss and GET — allow retry path
    return null;
  }
  return JSON.parse(raw) as DeliveryStatus;
}

export async function saveDeliveryStatus(status: DeliveryStatus): Promise<void> {
  const redis = getRedis();
  await redis.set(
    key(status.event_id),
    JSON.stringify(status),
    "EX",
    config.idempotencyTTLSeconds,
  );
}

export async function getDeliveryStatus(
  eventId: string,
): Promise<DeliveryStatus | null> {
  const raw = await getRedis().get(key(eventId));
  if (!raw) return null;
  return JSON.parse(raw) as DeliveryStatus;
}
