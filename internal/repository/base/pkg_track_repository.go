package base

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PkgTrackRepository implements pkg/domain.TrackRepository using GORM
type PkgTrackRepository struct {
	db *gorm.DB
}

// NewPkgTrackRepository creates a new pkg/domain track repository
func NewPkgTrackRepository(db *gorm.DB) domain.TrackRepository {
	return &PkgTrackRepository{db: db}
}

// Create creates a new track
func (r *PkgTrackRepository) Create(ctx context.Context, track *domain.Track) error {
	track.CreatedAt = time.Now()
	track.UpdatedAt = time.Now()
	if track.ID == "" {
		track.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(track)
	if result.Error != nil {
		return fmt.Errorf("failed to create track: %w", result.Error)
	}

	return nil
}

// GetByID retrieves a track by ID
func (r *PkgTrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	var track domain.Track
	result := r.db.WithContext(ctx).First(&track, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get track: %w", result.Error)
	}

	return &track, nil
}

// Update updates an existing track
func (r *PkgTrackRepository) Update(ctx context.Context, track *domain.Track) error {
	track.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).Save(track)
	if result.Error != nil {
		return fmt.Errorf("failed to update track: %w", result.Error)
	}

	return nil
}

// Delete deletes a track
func (r *PkgTrackRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.Track{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete track: %w", result.Error)
	}

	return nil
}

// List retrieves tracks with pagination and filtering
func (r *PkgTrackRepository) List(ctx context.Context, filter map[string]interface{}, offset, limit int) ([]*domain.Track, error) {
	var tracks []*domain.Track
	db := r.db.WithContext(ctx)

	// Apply filters if any
	for field, value := range filter {
		db = db.Where(fmt.Sprintf("%s = ?", field), value)
	}

	result := db.Offset(offset).Limit(limit).Find(&tracks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", result.Error)
	}

	return tracks, nil
}

// SearchByMetadata searches tracks by metadata fields
func (r *PkgTrackRepository) SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*domain.Track, error) {
	var tracks []*domain.Track
	db := r.db.WithContext(ctx)

	// Build query dynamically based on metadata fields
	for field, value := range query {
		db = db.Where(fmt.Sprintf("metadata->>'%s' = ?", field), value)
	}

	result := db.Find(&tracks)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", result.Error)
	}

	return tracks, nil
}

// GetByISRC retrieves a track by ISRC
func (r *PkgTrackRepository) GetByISRC(ctx context.Context, isrc string) (*domain.Track, error) {
	var track domain.Track
	result := r.db.WithContext(ctx).First(&track, "metadata->>'isrc' = ?", isrc)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get track by ISRC: %w", result.Error)
	}

	return &track, nil
}

// BatchUpdate updates multiple tracks in a single transaction
func (r *PkgTrackRepository) BatchUpdate(ctx context.Context, tracks []*domain.Track) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, track := range tracks {
			track.UpdatedAt = time.Now()
			if err := tx.Save(track).Error; err != nil {
				return fmt.Errorf("failed to update track %s: %w", track.ID, err)
			}
		}
		return nil
	})
}
