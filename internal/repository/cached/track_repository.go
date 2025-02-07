package cached

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	trackKeyPrefix = "track:"
	trackTTL       = 24 * time.Hour
)

// CachedTrackRepository implements domain.TrackRepository with Redis caching
type CachedTrackRepository struct {
	client   *redis.Client
	delegate domain.TrackRepository
}

// NewTrackRepository creates a new cached track repository
func NewTrackRepository(client *redis.Client, delegate domain.TrackRepository) domain.TrackRepository {
	return &CachedTrackRepository{
		client:   client,
		delegate: delegate,
	}
}

// Create inserts a new track and invalidates cache
func (r *CachedTrackRepository) Create(ctx context.Context, track *domain.Track) error {
	err := r.delegate.Create(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	// Invalidate any existing cache entries
	r.invalidateCache(ctx, track)
	return nil
}

// GetByID retrieves a track by ID, using cache if available
func (r *CachedTrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	// Try cache first
	key := fmt.Sprintf("%s:id:%s", trackKeyPrefix, id)
	track, err := r.getFromCache(ctx, key)
	if err == nil && track != nil {
		metrics.CacheHits.WithLabelValues("track").Inc()
		return track, nil
	}

	// Cache miss, get from delegate
	metrics.CacheMisses.WithLabelValues("track").Inc()
	track, err = r.delegate.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	if track != nil {
		if err := r.setCache(ctx, key, track); err != nil {
			// Log error but don't fail the request
			fmt.Printf("failed to cache track: %v\n", err)
		}
	}

	return track, nil
}

// SearchByMetadata searches tracks by metadata fields
func (r *CachedTrackRepository) SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*domain.Track, error) {
	// Search operations are not cached as they can be complex and varied
	return r.delegate.SearchByMetadata(ctx, query)
}

// Update modifies an existing track and invalidates cache
func (r *CachedTrackRepository) Update(ctx context.Context, track *domain.Track) error {
	err := r.delegate.Update(ctx, track)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	// Invalidate cache entries
	r.invalidateCache(ctx, track)
	return nil
}

// Delete removes a track and invalidates cache
func (r *CachedTrackRepository) Delete(ctx context.Context, id string) error {
	// Get track first to invalidate cache
	track, err := r.delegate.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	if track != nil {
		r.invalidateCache(ctx, track)
	}

	return r.delegate.Delete(ctx, id)
}

// List retrieves a paginated list of tracks (not cached)
func (r *CachedTrackRepository) List(ctx context.Context, offset, limit int) ([]*domain.Track, error) {
	// List operations are not cached as they can vary widely
	return r.delegate.List(ctx, offset, limit)
}

// Helper functions for cache operations

func (r *CachedTrackRepository) getFromCache(ctx context.Context, key string) (*domain.Track, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var track domain.Track
	if err := json.Unmarshal(data, &track); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track: %w", err)
	}

	return &track, nil
}

func (r *CachedTrackRepository) setCache(ctx context.Context, key string, track *domain.Track) error {
	data, err := json.Marshal(track)
	if err != nil {
		return fmt.Errorf("failed to marshal track: %w", err)
	}

	if err := r.client.Set(ctx, key, data, trackTTL).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

func (r *CachedTrackRepository) invalidateCache(ctx context.Context, track *domain.Track) {
	keys := []string{
		fmt.Sprintf("%s:id:%s", trackKeyPrefix, track.ID),
		// Add other cache keys that need invalidation
	}

	if len(keys) > 0 {
		r.client.Del(ctx, keys...)
	}
}
