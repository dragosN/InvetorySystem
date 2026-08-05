import type { OrderCreatedEvent } from "./types.ts";
import { config } from "./config";
import { webhookRetries } from "./metrics";

export type WebhookResult =
  | { ok: true; status: number; attempts: number }
  | { ok: false; status?: number; attempts: number; error: string };

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function isRetryableStatus(status: number): boolean {
  return status === 429 || status >= 500;
}

async function postOnce(
  event: OrderCreatedEvent,
  signal: AbortSignal,
): Promise<{ status: number; body: string }> {
  const res = await fetch(config.webhookUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Event-Id": event.event_id,
      "X-Event-Type": event.event_type,
    },
    body: JSON.stringify({
      event_id: event.event_id,
      event_type: event.event_type,
      order_id: event.order_id,
      total: event.total,
      status: event.status,
      timestamp: event.timestamp,
      items: event.items,
    }),
    signal,
  });

  const body = await res.text();
  return { status: res.status, body };
}

/**
 * Deliver webhook with exponential backoff.
 * Retries on network errors, 429, and 5xx. Does not retry other 4xx.
 */
export async function deliverWebhook(
  event: OrderCreatedEvent,
): Promise<WebhookResult> {
  let lastError = "unknown error";
  let lastStatus: number | undefined;

  for (let attempt = 1; attempt <= config.webhookMaxAttempts; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(
      () => controller.abort(),
      config.webhookTimeoutMs,
    );

    try {
      const { status } = await postOnce(event, controller.signal);
      lastStatus = status;

      if (status >= 200 && status < 300) {
        return { ok: true, status, attempts: attempt };
      }

      lastError = `webhook returned HTTP ${status}`;
      if (!isRetryableStatus(status) || attempt === config.webhookMaxAttempts) {
        return {
          ok: false,
          status,
          attempts: attempt,
          error: lastError,
        };
      }
    } catch (err) {
      lastError =
        err instanceof Error ? err.message : `webhook request failed: ${err}`;
      if (attempt === config.webhookMaxAttempts) {
        return { ok: false, status: lastStatus, attempts: attempt, error: lastError };
      }
    } finally {
      clearTimeout(timer);
    }

    const delay =
      config.webhookBaseDelayMs * 2 ** (attempt - 1) +
      Math.floor(Math.random() * 50);
    console.warn(
      JSON.stringify({
        msg: "webhook delivery retry scheduled",
        event_id: event.event_id,
        order_id: event.order_id,
        attempt,
        next_delay_ms: delay,
        error: lastError,
      }),
    );
    webhookRetries.inc();
    await sleep(delay);
  }

  return {
    ok: false,
    status: lastStatus,
    attempts: config.webhookMaxAttempts,
    error: lastError,
  };
}
