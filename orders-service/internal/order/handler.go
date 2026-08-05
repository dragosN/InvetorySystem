package order

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/nicadragos/InventorySystem/orders-service/internal/metrics"
)

type Publisher interface {
	PublishOrderCreated(ctx context.Context, o Order) error
}

type Handler struct {
	store     *Store
	publisher Publisher
	logger    *slog.Logger
}

func NewHandler(store *Store, publisher Publisher, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, publisher: publisher, logger: logger}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.createOrder)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("GET /healthz", h.healthz)
}

func (h *Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "items must be a non-empty array")
		return
	}
	for i, item := range req.Items {
		if item.SKU == "" {
			writeError(w, http.StatusBadRequest, "items["+strconv.Itoa(i)+"].sku is required")
			return
		}
		if item.Quantity <= 0 {
			writeError(w, http.StatusBadRequest, "items["+strconv.Itoa(i)+"].quantity must be > 0")
			return
		}
		if item.UnitPrice < 0 {
			writeError(w, http.StatusBadRequest, "items["+strconv.Itoa(i)+"].unit_price must be >= 0")
			return
		}
	}

	now := time.Now().UTC()
	o := Order{
		ID:        uuid.NewString(),
		Items:     req.Items,
		Total:     ComputeTotal(req.Items),
		Status:    "created",
		Timestamp: now,
	}

	if err := h.store.Create(r.Context(), o); err != nil {
		h.logger.Error("store order", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to store order")
		return
	}

	if err := h.publisher.PublishOrderCreated(r.Context(), o); err != nil {
		metrics.KafkaPublishErrors.Inc()
		h.logger.Error("publish order.created", "order_id", o.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "order saved but failed to publish event")
		return
	}

	metrics.OrdersCreated.Inc()
	h.logger.Info("order created", "order_id", o.ID, "total", o.Total)
	writeJSON(w, http.StatusCreated, o)
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "order id is required")
		return
	}

	o, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		h.logger.Error("get order", "order_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch order")
		return
	}

	writeJSON(w, http.StatusOK, o)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
