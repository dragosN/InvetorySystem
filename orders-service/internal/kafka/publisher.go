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
		},
	}
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}

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

	msg := kafkago.Message{
		Key:   []byte(o.ID),
		Value: payload,
		Time:  time.Now().UTC(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}
