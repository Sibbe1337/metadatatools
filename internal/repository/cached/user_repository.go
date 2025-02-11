package cached

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	userKeyPrefix = "user:"
	userTTL       = 24 * time.Hour
)

// CachedUserRepository implements domain.UserRepository with Redis caching
type CachedUserRepository struct {
	client   *redis.Client
	delegate domain.UserRepository
}

// NewUserRepository creates a new cached user repository
func NewUserRepository(client *redis.Client, delegate domain.UserRepository) domain.UserRepository {
	return &CachedUserRepository{
		client:   client,
		delegate: delegate,
	}
}

// Create inserts a new user and invalidates cache
func (r *CachedUserRepository) Create(ctx context.Context, user *domain.User) error {
	err := r.delegate.Create(ctx, user)
	if err != nil {
		return err
	}
	// Invalidate any existing cache entries
	r.invalidateCache(ctx, user)
	return nil
}

// GetByID retrieves a user by ID, using cache if available
func (r *CachedUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	// Try cache first
	key := fmt.Sprintf("%s:id:%s", userKeyPrefix, id)
	user, err := r.getFromCache(ctx, key)
	if err == nil && user != nil {
		return user, nil
	}

	// Cache miss, get from delegate
	user, err = r.delegate.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user != nil {
		r.setCache(ctx, key, user)
	}
	return user, nil
}

// GetByEmail retrieves a user by email, using cache if available
func (r *CachedUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Try cache first
	key := fmt.Sprintf("%s:email:%s", userKeyPrefix, email)
	user, err := r.getFromCache(ctx, key)
	if err == nil && user != nil {
		return user, nil
	}

	// Cache miss, get from delegate
	user, err = r.delegate.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		r.setCache(ctx, key, user)
	}
	return user, nil
}

// GetByAPIKey retrieves a user by API key, using cache if available
func (r *CachedUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	// Try cache first
	key := fmt.Sprintf("%s:apikey:%s", userKeyPrefix, apiKey)
	user, err := r.getFromCache(ctx, key)
	if err == nil && user != nil {
		return user, nil
	}

	// Cache miss, get from delegate
	user, err = r.delegate.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	if user != nil {
		r.setCache(ctx, key, user)
	}
	return user, nil
}

// Update modifies an existing user and invalidates cache
func (r *CachedUserRepository) Update(ctx context.Context, user *domain.User) error {
	err := r.delegate.Update(ctx, user)
	if err != nil {
		return err
	}
	// Invalidate cache entries
	r.invalidateCache(ctx, user)
	return nil
}

// Delete removes a user and invalidates cache
func (r *CachedUserRepository) Delete(ctx context.Context, id string) error {
	// Get user first to invalidate cache
	user, err := r.delegate.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user != nil {
		r.invalidateCache(ctx, user)
	}
	return r.delegate.Delete(ctx, id)
}

// List retrieves a paginated list of users (not cached)
func (r *CachedUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	return r.delegate.List(ctx, offset, limit)
}

// UpdateAPIKey updates the API key for a user
func (r *CachedUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	// Update in database
	if err := r.delegate.UpdateAPIKey(ctx, userID, apiKey); err != nil {
		return err
	}

	// Get user to invalidate all cache keys
	user, err := r.delegate.GetByID(ctx, userID)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("failed to get user for cache invalidation: %v\n", err)
		return nil
	}

	// Invalidate cache
	if err := r.invalidateCache(ctx, user); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("failed to invalidate user cache: %v\n", err)
	}

	return nil
}

// Helper functions for cache operations

func (r *CachedUserRepository) getFromCache(ctx context.Context, key string) (*domain.User, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var user domain.User
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *CachedUserRepository) setCache(ctx context.Context, key string, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, userTTL).Err()
}

func (r *CachedUserRepository) invalidateCache(ctx context.Context, user *domain.User) error {
	keys := []string{
		fmt.Sprintf("%s:id:%s", userKeyPrefix, user.ID),
		fmt.Sprintf("%s:email:%s", userKeyPrefix, user.Email),
		fmt.Sprintf("%s:apikey:%s", userKeyPrefix, user.APIKey),
	}
	return r.client.Del(ctx, keys...).Err()
}
