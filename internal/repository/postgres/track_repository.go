package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"

	"gorm.io/gorm"
)

// TrackModel is the GORM model for tracks
type TrackModel struct {
	gorm.Model
	ID       string `gorm:"primaryKey"`
	Title    string
	Artist   string
	Album    string
	Genre    string
	Duration float64
	Metadata []byte `gorm:"type:jsonb"` // Store metadata as JSON
}

// PostgresTrackRepository implements domain.TrackRepository
type PostgresTrackRepository struct {
	db *gorm.DB
}

// NewPostgresTrackRepository creates a new PostgresTrackRepository
func NewPostgresTrackRepository(db *gorm.DB) *PostgresTrackRepository {
	return &PostgresTrackRepository{db: db}
}

// toModel converts a domain Track to a TrackModel
func toModel(track *domain.Track) (*TrackModel, error) {
	metadata, err := json.Marshal(track.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return &TrackModel{
		ID:       track.ID,
		Title:    track.Title,
		Artist:   track.Artist,
		Album:    track.Album,
		Genre:    track.Genre,
		Duration: track.Duration,
		Metadata: metadata,
	}, nil
}

// toDomain converts a TrackModel to a domain Track
func (m *TrackModel) toDomain() (*domain.Track, error) {
	var metadata domain.Metadata
	if err := json.Unmarshal(m.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &domain.Track{
		ID:       m.ID,
		Title:    m.Title,
		Artist:   m.Artist,
		Album:    m.Album,
		Genre:    m.Genre,
		Duration: m.Duration,
		Metadata: metadata,
	}, nil
}

// Create stores a new track
func (r *PostgresTrackRepository) Create(ctx context.Context, track *domain.Track) error {
	model, err := toModel(track)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return fmt.Errorf("failed to create track: %w", result.Error)
	}

	return nil
}

// GetByID retrieves a track by ID
func (r *PostgresTrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	var model TrackModel
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get track: %w", result.Error)
	}

	return model.toDomain()
}

// Update modifies an existing track
func (r *PostgresTrackRepository) Update(ctx context.Context, track *domain.Track) error {
	model, err := toModel(track)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update track: %w", result.Error)
	}

	return nil
}

// Delete removes a track
func (r *PostgresTrackRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&TrackModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete track: %w", result.Error)
	}

	return nil
}

// List retrieves a paginated list of tracks
func (r *PostgresTrackRepository) List(ctx context.Context, offset, limit int) ([]*domain.Track, error) {
	var models []TrackModel
	result := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", result.Error)
	}

	tracks := make([]*domain.Track, len(models))
	for i, model := range models {
		track, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		tracks[i] = track
	}

	return tracks, nil
}

// SearchByMetadata searches tracks by metadata fields
func (r *PostgresTrackRepository) SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*domain.Track, error) {
	var models []TrackModel
	db := r.db.WithContext(ctx)

	// Build the query dynamically based on the search criteria
	for field, value := range query {
		switch field {
		case "title", "artist", "album", "genre":
			db = db.Where(fmt.Sprintf("%s ILIKE ?", field), fmt.Sprintf("%%%v%%", value))
		case "year", "duration":
			db = db.Where(fmt.Sprintf("%s = ?", field), value)
		case "metadata":
			// For metadata fields, use JSONB containment operator
			if metadataQuery, ok := value.(map[string]interface{}); ok {
				for metaField, metaValue := range metadataQuery {
					db = db.Where("metadata @> ?", map[string]interface{}{metaField: metaValue})
				}
			}
		}
	}

	result := db.Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", result.Error)
	}

	tracks := make([]*domain.Track, len(models))
	for i, model := range models {
		track, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		tracks[i] = track
	}

	return tracks, nil
}
