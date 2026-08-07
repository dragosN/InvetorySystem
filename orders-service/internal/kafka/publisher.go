package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"github.com/nicadragos/InventorySystem/orders-service/internal/order"
)

type OrderCreatedEvent struct {
	EventID   string       `json:"event_id"`
	EventType string       `json:"event_type"`
	OrderID   string       `json:"order_id"`
	Items     []order.Item `json:"items"`
	Total     int64        `json:"total"`
	Status    string       `json:"status"`
	Timestamp time.Time    `json:"timestamp"`
}

type Publisher struct {
	writer *kafkago.Writer
	topic  string
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{
		topic: topic,
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireOne,
			Async:        false,
			BatchTimeout: 10 * time.Millisecond,
			BatchSize:    100,
		},
	}
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}

// PublishRaw writes a pre-serialized outbox payload (key = order id).
func (p *Publisher) PublishRaw(ctx context.Context, key string, payload []byte) error {
	return p.PublishRawBatch(ctx, []kafkago.Message{{
		Key:   []byte(key),
		Value: payload,
		Time:  time.Now().UTC(),
	}})
}

// PublishRawBatch writes multiple outbox payloads in one Kafka produce round-trip.
func (p *Publisher) PublishRawBatch(ctx context.Context, msgs []kafkago.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	if err := p.writer.WriteMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("write kafka messages: %w", err)
	}
	return nil
}

// PublishOrderCreated builds and publishes an order.created event (tests / ad-hoc use).
func (p *Publisher) PublishOrderCreated(ctx context.Context, o order.Order) error {
	event := OrderCreatedEvent{
		EventID:   uuid.NewString(),
		EventType: "order.created",
		OrderID:   o.ID,
		Items:     o.Items,
		Total:     o.Total,
		Status:    o.Status,
		Timestamp: o.Timestamp,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.PublishRaw(ctx, o.ID, payload)
}
