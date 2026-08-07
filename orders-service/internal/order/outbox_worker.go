package order

import (
	"context"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/nicadragos/InventorySystem/orders-service/internal/metrics"
)

// EventPublisher writes pre-built outbox payloads to Kafka.
type EventPublisher interface {
	PublishRaw(ctx context.Context, key string, payload []byte) error
	PublishRawBatch(ctx context.Context, msgs []kafkago.Message) error
}

// OutboxWorker drains unpublished outbox rows to Kafka.
// HTTP creates only write SQLite; this loop owns the network hop.
type OutboxWorker struct {
	store        *Store
	publisher    EventPublisher
	logger       *slog.Logger
	notify       chan struct{}
	pollInterval time.Duration
	batchSize    int
}

func NewOutboxWorker(store *Store, publisher EventPublisher, logger *slog.Logger) *OutboxWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &OutboxWorker{
		store:        store,
		publisher:    publisher,
		logger:       logger,
		notify:       make(chan struct{}, 1),
		pollInterval: 100 * time.Millisecond,
		batchSize:    100,
	}
}

// Notify wakes the worker after a new outbox row is committed (non-blocking).
func (w *OutboxWorker) Notify() {
	select {
	case w.notify <- struct{}{}:
	default:
	}
}

func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.refreshPending(ctx)

	for {
		w.drain(ctx)

		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			w.drain(flushCtx)
			cancel()
			return
		case <-w.notify:
		case <-ticker.C:
		}
	}
}

func (w *OutboxWorker) drain(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		rows, err := w.store.ListUnpublishedOutbox(ctx, w.batchSize)
		if err != nil {
			w.logger.Error("list outbox", "error", err)
			w.refreshPending(ctx)
			return
		}
		if len(rows) == 0 {
			w.refreshPending(ctx)
			return
		}

		msgs := make([]kafkago.Message, 0, len(rows))
		now := time.Now().UTC()
		for _, row := range rows {
			msgs = append(msgs, kafkago.Message{
				Key:   []byte(row.OrderID),
				Value: row.Payload,
				Time:  now,
			})
		}

		if err := w.publisher.PublishRawBatch(ctx, msgs); err != nil {
			metrics.KafkaPublishErrors.Inc()
			w.logger.Error("publish outbox batch",
				"batch_size", len(rows),
				"first_outbox_id", rows[0].ID,
				"error", err,
			)
			w.refreshPending(ctx)
			return
		}

		ids := make([]int64, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		if err := w.store.MarkOutboxPublishedBatch(ctx, ids, now); err != nil {
			w.logger.Error("mark outbox batch published",
				"batch_size", len(rows),
				"error", err,
			)
			w.refreshPending(ctx)
			return
		}
		metrics.OrdersPublished.Add(float64(len(rows)))

		w.logger.Info("outbox batch published",
			"batch_size", len(rows),
			"first_outbox_id", rows[0].ID,
			"last_outbox_id", rows[len(rows)-1].ID,
		)
		w.refreshPending(ctx)

		if len(rows) < w.batchSize {
			return
		}
	}
}

func (w *OutboxWorker) refreshPending(ctx context.Context) {
	n, err := w.store.CountUnpublishedOutbox(ctx)
	if err != nil {
		w.logger.Error("count outbox", "error", err)
		return
	}
	metrics.OutboxPending.Set(float64(n))
}
