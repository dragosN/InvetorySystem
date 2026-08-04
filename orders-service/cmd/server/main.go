package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/nicadragos/InventorySystem/orders-service/internal/kafka"
	"github.com/nicadragos/InventorySystem/orders-service/internal/order"
	"github.com/nicadragos/InventorySystem/orders-service/internal/ratelimit"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg := loadConfig()
	if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0o755); err != nil {
		logger.Error("create data dir", "path", cfg.SQLitePath, "error", err)
		os.Exit(1)
	}

	store, err := order.NewStore(cfg.SQLitePath)
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	publisher := kafka.NewPublisher(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer publisher.Close()

	limiter, err := ratelimit.New(cfg.RedisAddr, cfg.RateLimit, cfg.RateWindowSec)
	if err != nil {
		logger.Error("connect redis", "addr", cfg.RedisAddr, "error", err)
		os.Exit(1)
	}
	defer limiter.Close()

	mux := http.NewServeMux()
	order.NewHandler(store, publisher, logger).Register(mux)

	var handler http.Handler = mux
	handler = ratelimit.Middleware(limiter, logger)(handler)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("orders-service listening",
			"addr", cfg.HTTPAddr,
			"kafka_brokers", strings.Join(cfg.KafkaBrokers, ","),
			"kafka_topic", cfg.KafkaTopic,
			"sqlite", cfg.SQLitePath,
			"redis", cfg.RedisAddr,
			"rate_limit", cfg.RateLimit,
			"rate_window_sec", cfg.RateWindowSec,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("orders-service stopped")
}

type config struct {
	HTTPAddr       string
	KafkaBrokers   []string
	KafkaTopic     string
	SQLitePath     string
	RedisAddr      string
	RateLimit      int
	RateWindowSec  int
}

func loadConfig() config {
	return config{
		HTTPAddr:      envOr("HTTP_ADDR", ":8080"),
		KafkaBrokers:  strings.Split(envOr("KAFKA_BROKERS", "localhost:19092"), ","),
		KafkaTopic:    envOr("KAFKA_TOPIC", "order.created"),
		SQLitePath:    envOr("SQLITE_PATH", "data/orders.db"),
		RedisAddr:     envOr("REDIS_ADDR", "localhost:6379"),
		RateLimit:     ratelimit.MustAtoi(envOr("RATE_LIMIT", "5"), 5),
		RateWindowSec: ratelimit.MustAtoi(envOr("RATE_WINDOW_SEC", "60"), 60),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
