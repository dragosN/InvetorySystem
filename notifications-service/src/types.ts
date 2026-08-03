export type OrderItem = {
  sku: string;
  quantity: number;
  unit_price: number;
};

export type OrderCreatedEvent = {
  event_id: string;
  event_type: string;
  order_id: string;
  items: OrderItem[];
  total: number;
  status: string;
  timestamp: string;
};

export function isOrderCreatedEvent(value: unknown): value is OrderCreatedEvent {
  if (!value || typeof value !== "object") return false;
  const v = value as Record<string, unknown>;
  return (
    typeof v.event_id === "string" &&
    typeof v.event_type === "string" &&
    typeof v.order_id === "string" &&
    Array.isArray(v.items) &&
    typeof v.total === "number" &&
    typeof v.status === "string"
  );
}
