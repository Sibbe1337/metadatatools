package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TrackModel represents the database model for tracks
type TrackModel struct {
	ID          string `gorm:"primaryKey;type:uuid"`
	StoragePath string
	FileSize    int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time      `gorm:"index"`
	Metadata    json.RawMessage `gorm:"type:jsonb"`
}

// PostgresTrackRepository implements domain.TrackRepository
type PostgresTrackRepository struct {
	db *gorm.DB
}

// NewPostgresTrackRepository creates a new PostgreSQL track repository
func NewPostgresTrackRepository(db *gorm.DB) *PostgresTrackRepository {
	return &PostgresTrackRepository{db: db}
}

// Create inserts a new track
func (r *PostgresTrackRepository) Create(ctx context.Context, track *domain.Track) error {
	start := time.Now()
	defer func() {
		metrics.DatabaseQueryDuration.WithLabelValues("track_create").Observe(time.Since(start).Seconds())
	}()

	if track.ID == "" {
		track.ID = uuid.New().String()
	}

	model, err := toModel(track)
	if err != nil {
		return fmt.Errorf("failed to convert track to model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	return nil
}

// GetByID retrieves a track by ID
func (r *PostgresTrackRepository) GetByID(ctx context.Context, id string) (*domain.Track, error) {
	start := time.Now()
	defer func() {
		metrics.DatabaseQueryDuration.WithLabelValues("track_get").Observe(time.Since(start).Seconds())
	}()

	var model TrackModel
	if err := r.db.WithContext(ctx).First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	track := &domain.Track{
		ID:          model.ID,
		StoragePath: model.StoragePath,
		FileSize:    model.FileSize,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		DeletedAt:   model.DeletedAt,
	}

	if err := json.Unmarshal(model.Metadata, &track.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return track, nil
}

// Update modifies an existing track
func (r *PostgresTrackRepository) Update(ctx context.Context, track *domain.Track) error {
	start := time.Now()
	defer func() {
		metrics.DatabaseQueryDuration.WithLabelValues("track_update").Observe(time.Since(start).Seconds())
	}()

	model, err := toModel(track)
	if err != nil {
		return fmt.Errorf("failed to convert track to model: %w", err)
	}

	result := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", track.ID).Updates(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update track: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("track not found: %s", track.ID)
	}

	return nil
}

// Delete soft deletes a track
func (r *PostgresTrackRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer func() {
		metrics.DatabaseQueryDuration.WithLabelValues("track_delete").Observe(time.Since(start).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(&TrackModel{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", time.Now())
	if result.Error != nil {
		return fmt.Errorf("failed to delete track: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("track not found: %s", id)
	}

	return nil
}

// List retrieves a paginated list of tracks
func (r *PostgresTrackRepository) List(ctx context.Context, offset, limit int) ([]*domain.Track, error) {
	start := time.Now()
	defer func() {
		metrics.DatabaseQueryDuration.WithLabelValues("track_list").Observe(time.Since(start).Seconds())
	}()

	var models []TrackModel
	result := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&models)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", result.Error)
	}

	tracks := make([]*domain.Track, len(models))
	for i, model := range models {
		track, err := model.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to track: %w", err)
		}
		tracks[i] = track
	}

	return tracks, nil
}

// SearchByMetadata searches tracks by metadata fields
func (r *PostgresTrackRepository) SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*domain.Track, error) {
	start := time.Now()
	defer func() {
		metrics.DatabaseQueryDuration.WithLabelValues("track_search").Observe(time.Since(start).Seconds())
	}()

	db := r.db.WithContext(ctx).Model(&TrackModel{}).Where("deleted_at IS NULL")

	// Build query dynamically based on metadata fields
	for field, value := range query {
		switch field {
		case "title", "artist", "album", "genre", "label", "publisher":
			db = db.Where(fmt.Sprintf("%s ILIKE ?", field), fmt.Sprintf("%%%v%%", value))
		case "year", "bpm":
			db = db.Where(fmt.Sprintf("%s = ?", field), value)
		case "isrc", "iswc":
			db = db.Where(fmt.Sprintf("%s = ?", field), value)
		case "ai_confidence":
			db = db.Where("ai_confidence >= ?", value)
		case "needs_review":
			db = db.Where("needs_review = ?", value)
		case "created_after":
			if t, ok := value.(time.Time); ok {
				db = db.Where("created_at >= ?", t)
			}
		case "created_before":
			if t, ok := value.(time.Time); ok {
				db = db.Where("created_at <= ?", t)
			}
		}
	}

	var models []TrackModel
	result := db.Find(&models)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", result.Error)
	}

	tracks := make([]*domain.Track, len(models))
	for i, model := range models {
		track, err := model.toDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert model to track: %w", err)
		}
		tracks[i] = track
	}

	return tracks, nil
}

// Helper functions

func toModel(track *domain.Track) (*TrackModel, error) {
	metadata, err := json.Marshal(track.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return &TrackModel{
		ID:          track.ID,
		StoragePath: track.StoragePath,
		FileSize:    track.FileSize,
		CreatedAt:   track.CreatedAt,
		UpdatedAt:   track.UpdatedAt,
		DeletedAt:   track.DeletedAt,
		Metadata:    metadata,
	}, nil
}

func (m *TrackModel) toDomain() (*domain.Track, error) {
	var metadata domain.CompleteTrackMetadata
	if m.Metadata != nil {
		if err := json.Unmarshal(m.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	track := &domain.Track{
		ID:          m.ID,
		StoragePath: m.StoragePath,
		FileSize:    m.FileSize,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
		Metadata:    metadata,
	}
	return track, nil
}
