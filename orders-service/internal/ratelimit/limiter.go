package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Fixed-window counter: max N requests per windowSeconds per client key.
type Limiter struct {
	rdb           *redis.Client
	limit         int
	windowSeconds int
	prefix        string
}

func New(redisAddr string, limit, windowSeconds int) (*Limiter, error) {
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	if limit <= 0 {
		limit = 10
	}
	if windowSeconds <= 0 {
		windowSeconds = 60
	}
	return &Limiter{
		rdb:           rdb,
		limit:         limit,
		windowSeconds: windowSeconds,
		prefix:        "ratelimit:orders",
	}, nil
}

func (l *Limiter) Close() error {
	return l.rdb.Close()
}

func (l *Limiter) Limit() int { return l.limit }

func (l *Limiter) WindowSeconds() int { return l.windowSeconds }

type Result struct {
	Allowed    bool
	Count      int
	Remaining  int
	RetryAfter time.Duration
}

// Allow increments the fixed-window counter for clientID and reports whether
// the request is within the limit.
func (l *Limiter) Allow(ctx context.Context, clientID string) (Result, error) {
	window := time.Now().UTC().Unix() / int64(l.windowSeconds)
	key := fmt.Sprintf("%s:%s:%d", l.prefix, clientID, window)

	// Atomic INCR + EXPIRE on first hit.
	count, err := incrWithExpire.Run(ctx, l.rdb, []string{key}, l.windowSeconds).Int()
	if err != nil {
		return Result{}, fmt.Errorf("redis incr: %w", err)
	}

	remaining := l.limit - count
	if remaining < 0 {
		remaining = 0
	}

	if count > l.limit {
		elapsed := time.Now().UTC().Unix() % int64(l.windowSeconds)
		retry := time.Duration(l.windowSeconds-int(elapsed)) * time.Second
		if retry <= 0 {
			retry = time.Duration(l.windowSeconds) * time.Second
		}
		return Result{
			Allowed:    false,
			Count:      count,
			Remaining:  0,
			RetryAfter: retry,
		}, nil
	}

	return Result{
		Allowed:   true,
		Count:     count,
		Remaining: remaining,
	}, nil
}

var incrWithExpire = redis.NewScript(`
local n = redis.call("INCR", KEYS[1])
if n == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return n
`)

// ParseLimit helpers for config.
func MustAtoi(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
