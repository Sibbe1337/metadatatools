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
	ID        string `gorm:"primaryKey;type:uuid"`
	Title     string `gorm:"not null"`
	Artist    string `gorm:"not null"`
	Album     string
	Genre     string
	Duration  float64
	FilePath  string
	Year      int
	Label     string
	Territory string
	ISRC      string `gorm:"index"`
	ISWC      string
	BPM       float64
	Key       string
	Mood      string
	Publisher string

	// Audio metadata
	AudioFormat string
	FileSize    int64

	// AI-related fields
	AITags       []string `gorm:"type:text[]"`
	AIConfidence float64
	ModelVersion string
	NeedsReview  bool
	AIMetadata   json.RawMessage `gorm:"type:jsonb"`

	// Base metadata
	Metadata json.RawMessage `gorm:"type:jsonb"`

	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
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

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return fmt.Errorf("failed to create track: %w", result.Error)
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
	result := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&model)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get track: %w", result.Error)
	}

	track, err := model.toDomain()
	if err != nil {
		return nil, fmt.Errorf("failed to convert model to track: %w", err)
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
	aiMetadata, err := json.Marshal(track.AIMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI metadata: %w", err)
	}

	metadata, err := json.Marshal(track.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return &TrackModel{
		ID:           track.ID,
		Title:        track.Title,
		Artist:       track.Artist,
		Album:        track.Album,
		Genre:        track.Genre,
		Duration:     track.Duration,
		FilePath:     track.FilePath,
		Year:         track.Year,
		Label:        track.Label,
		Territory:    track.Territory,
		ISRC:         track.ISRC,
		ISWC:         track.ISWC,
		BPM:          track.BPM,
		Key:          track.Key,
		Mood:         track.Mood,
		Publisher:    track.Publisher,
		AudioFormat:  track.AudioFormat,
		FileSize:     track.FileSize,
		AITags:       track.AITags,
		AIConfidence: track.AIConfidence,
		ModelVersion: track.ModelVersion,
		NeedsReview:  track.NeedsReview,
		AIMetadata:   aiMetadata,
		Metadata:     metadata,
		CreatedAt:    track.CreatedAt,
		UpdatedAt:    track.UpdatedAt,
		DeletedAt:    track.DeletedAt,
	}, nil
}

func (m *TrackModel) toDomain() (*domain.Track, error) {
	var aiMetadata *domain.AIMetadata
	if m.AIMetadata != nil {
		if err := json.Unmarshal(m.AIMetadata, &aiMetadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal AI metadata: %w", err)
		}
	}

	var metadata domain.Metadata
	if m.Metadata != nil {
		if err := json.Unmarshal(m.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &domain.Track{
		ID:           m.ID,
		Title:        m.Title,
		Artist:       m.Artist,
		Album:        m.Album,
		Genre:        m.Genre,
		Duration:     m.Duration,
		FilePath:     m.FilePath,
		Year:         m.Year,
		Label:        m.Label,
		Territory:    m.Territory,
		ISRC:         m.ISRC,
		ISWC:         m.ISWC,
		BPM:          m.BPM,
		Key:          m.Key,
		Mood:         m.Mood,
		Publisher:    m.Publisher,
		AudioFormat:  m.AudioFormat,
		FileSize:     m.FileSize,
		AITags:       m.AITags,
		AIConfidence: m.AIConfidence,
		ModelVersion: m.ModelVersion,
		NeedsReview:  m.NeedsReview,
		AIMetadata:   aiMetadata,
		Metadata:     metadata,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}, nil
}
