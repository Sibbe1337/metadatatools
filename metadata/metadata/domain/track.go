package domain

import (
	"context"
	"time"
)

// Track represents a music track entity
type Track struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Artist    string    `json:"artist"`
	Album     string    `json:"album,omitempty"`
	Genre     string    `json:"genre,omitempty"`
	Duration  float64   `json:"duration"`
	Metadata  Metadata  `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
}

// Constants for storage paths
const (
	StoragePathAudio = "audio/"
	CacheKeyTrack    = "track:%s"
)
