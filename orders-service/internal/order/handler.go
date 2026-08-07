package order

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/nicadragos/InventorySystem/orders-service/internal/metrics"
)

type Handler struct {
	store  *Store
	worker *OutboxWorker
	logger *slog.Logger
}

func NewHandler(store *Store, worker *OutboxWorker, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, worker: worker, logger: logger}
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

	eventID := uuid.NewString()
	payload, err := json.Marshal(map[string]any{
		"event_id":   eventID,
		"event_type": "order.created",
		"order_id":   o.ID,
		"items":      o.Items,
		"total":      o.Total,
		"status":     o.Status,
		"timestamp":  o.Timestamp,
	})
	if err != nil {
		h.logger.Error("marshal outbox payload", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to store order")
		return
	}

	if err := h.store.CreateWithOutbox(r.Context(), o, eventID, payload); err != nil {
		h.logger.Error("store order + outbox", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to store order")
		return
	}

	h.worker.Notify()
	metrics.OrdersCreated.Inc()
	h.logger.Debug("order created", "order_id", o.ID, "event_id", eventID, "total", o.Total)
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
