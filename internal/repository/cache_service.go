// Package repository implements the data access layer
package repository

import (
	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client *redis.Client
}

// NewCacheService creates a new Redis cache service
func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{
		client: client,
	}
}

// ... existing code ...
