package ratelimit

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Middleware rate-limits POST /orders by X-Client-Id (or client IP).
func Middleware(limiter *Limiter, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/orders" {
				next.ServeHTTP(w, r)
				return
			}

			clientID := clientKey(r)
			result, err := limiter.Allow(r.Context(), clientID)
			if err != nil {
				logger.Error("rate limit check failed", "client", clientID, "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limiter unavailable",
				})
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))

			if !result.Allowed {
				secs := int(result.RetryAfter.Seconds())
				if secs < 1 {
					secs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":       "rate limit exceeded",
					"limit":       limiter.Limit(),
					"window_sec":  limiter.WindowSeconds(),
					"retry_after": secs,
					"client":      clientID,
				})
				logger.Warn("rate limit exceeded",
					"client", clientID,
					"count", result.Count,
					"limit", limiter.Limit(),
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientKey(r *http.Request) string {
	if id := strings.TrimSpace(r.Header.Get("X-Client-Id")); id != "" {
		return id
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
