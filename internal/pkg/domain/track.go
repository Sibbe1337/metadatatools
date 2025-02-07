package domain

import (
	"context"
	"time"
)

// Track represents a music track entity
type Track struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Artist    string  `json:"artist"`
	Album     string  `json:"album,omitempty"`
	Genre     string  `json:"genre,omitempty"`
	Duration  float64 `json:"duration"`
	FilePath  string  `json:"file_path"`
	Year      int     `json:"year,omitempty"`
	Label     string  `json:"label,omitempty"`
	Territory string  `json:"territory,omitempty"`
	ISRC      string  `json:"isrc,omitempty"`
	ISWC      string  `json:"iswc,omitempty"`
	BPM       float64 `json:"bpm,omitempty"`
	Key       string  `json:"key,omitempty"`
	Mood      string  `json:"mood,omitempty"`
	Publisher string  `json:"publisher,omitempty"`

	// Audio data
	AudioData   []byte `json:"-"`
	AudioFormat string `json:"audio_format,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`

	// AI-related fields
	AITags       []string `json:"ai_tags,omitempty"`
	AIConfidence float64  `json:"ai_confidence,omitempty"`
	ModelVersion string   `json:"model_version,omitempty"`
	NeedsReview  bool     `json:"needs_review,omitempty"`

	// AI-generated metadata
	AIMetadata *AIMetadata `json:"ai_metadata,omitempty"`

	Metadata  Metadata   `json:"metadata"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Metadata represents additional track metadata
type Metadata struct {
	ISRC         string            `json:"isrc,omitempty"`
	ISWC         string            `json:"iswc,omitempty"`
	BPM          float64           `json:"bpm,omitempty"`
	Key          string            `json:"key,omitempty"`
	Mood         string            `json:"mood,omitempty"`
	Labels       []string          `json:"labels,omitempty"`
	AITags       []string          `json:"ai_tags,omitempty"`
	Confidence   float64           `json:"confidence,omitempty"`
	ModelVersion string            `json:"model_version,omitempty"`
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// TrackRepository defines the interface for track data persistence
type TrackRepository interface {
	Create(ctx context.Context, track *Track) error
	GetByID(ctx context.Context, id string) (*Track, error)
	Update(ctx context.Context, track *Track) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*Track, error)
	SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*Track, error)
}
