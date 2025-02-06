package cached

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"metadatatool/internal/pkg/domain"

	"github.com/redis/go-redis/v9"
)

const (
	trackKeyPrefix = "track:"
	trackTTL       = 24 * time.Hour
)

// CachedTrackRepository implements domain.TrackRepository with Redis caching
type CachedTrackRepository struct {
	redis    *redis.Client
	delegate domain.TrackRepository
}

// NewTrackRepository creates a new CachedTrackRepository
func NewTrackRepository(redis *redis.Client, delegate domain.TrackRepository) *CachedTrackRepository {
	return &CachedTrackRepository{
		redis:    redis,
		delegate: delegate,
	}
}

// Create inserts a new track and invalidates cache
func (r *CachedTrackRepository) Create(ctx context.Context, track *domain.Track) error {
	err := r.delegate.Create(ctx, track)
	if err != nil {
		return err
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
		return track, nil
	}

	// Cache miss, get from delegate
	track, err = r.delegate.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if track != nil {
		r.setCache(ctx, key, track)
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
		return err
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
		return err
	}
	if track != nil {
		r.invalidateCache(ctx, track)
	}
	return r.delegate.Delete(ctx, id)
}

// List retrieves a paginated list of tracks (not cached)
func (r *CachedTrackRepository) List(ctx context.Context, offset, limit int) ([]*domain.Track, error) {
	return r.delegate.List(ctx, offset, limit)
}

// Helper functions for cache operations

func (r *CachedTrackRepository) getFromCache(ctx context.Context, key string) (*domain.Track, error) {
	data, err := r.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var track domain.Track
	err = json.Unmarshal(data, &track)
	if err != nil {
		return nil, err
	}
	return &track, nil
}

func (r *CachedTrackRepository) setCache(ctx context.Context, key string, track *domain.Track) error {
	data, err := json.Marshal(track)
	if err != nil {
		return err
	}
	return r.redis.Set(ctx, key, data, trackTTL).Err()
}

func (r *CachedTrackRepository) invalidateCache(ctx context.Context, track *domain.Track) {
	keys := []string{
		fmt.Sprintf("%s:id:%s", trackKeyPrefix, track.ID),
	}
	r.redis.Del(ctx, keys...)
}
