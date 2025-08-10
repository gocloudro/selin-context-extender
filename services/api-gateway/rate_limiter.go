package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
}

func NewRateLimiter() *RateLimiter {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	limit := 60 // Default: 60 requests per minute for users
	if limitStr := os.Getenv("RATE_LIMIT"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	window := time.Minute // 1 minute window

	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	return &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
	}
}

func (rl *RateLimiter) IsAllowed(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		userID = "anonymous"
	}

	key := fmt.Sprintf("rate_limit:%s", userID)

	// Use sliding window with Redis
	now := time.Now()
	windowStart := now.Add(-rl.window)

	pipe := rl.client.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%.0f", float64(windowStart.UnixNano())))

	// Count current requests in window
	countCmd := pipe.ZCard(ctx, key)

	// Add current request
	pipe.ZAdd(ctx, key, &redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiration for the key
	pipe.Expire(ctx, key, rl.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	count := countCmd.Val()
	return count < int64(rl.limit), nil
}

func (rl *RateLimiter) Close() error {
	return rl.client.Close()
}

// Middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from header or context
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			// For now, use IP as identifier for anonymous users
			userID = r.RemoteAddr
		}

		allowed, err := rl.IsAllowed(r.Context(), userID)
		if err != nil {
			http.Error(w, "Rate limit check failed", http.StatusInternalServerError)
			return
		}

		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
