package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_http_requests_total",
			Help: "Total HTTP requests handled by orders-service",
		},
		[]string{"method", "path", "status"},
	)

	HTTPDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "orders_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	OrdersCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Orders successfully created and published",
		},
	)

	KafkaPublishErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_kafka_publish_errors_total",
			Help: "Failed Kafka publishes of order.created",
		},
	)

	RateLimitRejected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_rate_limit_rejected_total",
			Help: "POST /orders rejected by Redis rate limiter",
		},
	)
)

func Handler() http.Handler {
	return promhttp.Handler()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware records request count and latency for all routes except /metrics.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		path := normalizePath(r)
		HTTPRequests.WithLabelValues(r.Method, path, strconv.Itoa(rec.status)).Inc()
		HTTPDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())

		if r.Method == http.MethodPost && path == "/orders" && rec.status == http.StatusTooManyRequests {
			RateLimitRejected.Inc()
		}
	})
}

func normalizePath(r *http.Request) string {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/orders":
		return "/orders"
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/orders/"):
		return "/orders/:id"
	case r.URL.Path == "/healthz":
		return "/healthz"
	default:
		return r.URL.Path
	}
}
