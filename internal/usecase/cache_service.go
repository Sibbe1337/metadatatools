package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/config"
	"metadatatool/internal/pkg/metrics"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheService handles caching operations
type CacheService struct {
	client *redis.Client
	cfg    *config.RedisConfig
}

// NewCacheService creates a new cache service
func NewCacheService(client *redis.Client, cfg *config.RedisConfig) *CacheService {
	return &CacheService{
		client: client,
		cfg:    cfg,
	}
}

// Get retrieves a value from cache
func (s *CacheService) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		metrics.CacheMisses.WithLabelValues("redis").Inc()
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	metrics.CacheHits.WithLabelValues("redis").Inc()
	return val, nil
}

// Set stores a value in cache
func (s *CacheService) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	err := s.client.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set in cache: %w", err)
	}
	return nil
}

// Delete removes a value from cache
func (s *CacheService) Delete(ctx context.Context, key string) error {
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}
	return nil
}

// PreWarm pre-warms the cache with frequently accessed data
func (s *CacheService) PreWarm(ctx context.Context, keys []string) error {
	pipe := s.client.Pipeline()
	for _, key := range keys {
		pipe.Get(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to pre-warm cache: %w", err)
	}
	return nil
}
