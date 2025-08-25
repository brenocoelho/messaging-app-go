package redisconn

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyService struct {
	client *redis.Client
	ttl    time.Duration
}

func NewIdempotencyService(client *redis.Client, ttlMinutes int) *IdempotencyService {
	if ttlMinutes <= 0 {
		ttlMinutes = 10 // Default to 10 minutes
	}

	return &IdempotencyService{
		client: client,
		ttl:    time.Duration(ttlMinutes) * time.Minute,
	}
}
func (s *IdempotencyService) CheckAndSetIdempotency(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("idempotency:%s", key)

	exists, err := s.client.Exists(ctx, redisKey).Result()
	if err != nil {
		slog.Error("Error checking Redis for idempotency key", "key", key, "error", err)
		return false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	if exists > 0 {
		slog.Info("Idempotency key found in Redis", "key", key)
		return true, nil
	}

	err = s.client.Set(ctx, redisKey, "1", s.ttl).Err()
	if err != nil {
		slog.Error("Error setting idempotency key in Redis", "key", key, "error", err)
		return false, fmt.Errorf("failed to set idempotency key: %w", err)
	}

	slog.Info("Idempotency key set successfully", "key", key, "ttl", s.ttl)
	return false, nil
}

func (s *IdempotencyService) GetTTL() time.Duration {
	return s.ttl
}
