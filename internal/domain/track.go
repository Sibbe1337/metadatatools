package domain

import (
	"context"
	"time"
)

// Track represents a music track in the internal domain
type Track struct {
	ID        string
	LabelID   string
	Status    TrackStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  *AIMetadata
}

// TrackStatus represents the current status of a track
type TrackStatus string

const (
	// Track status constants
	TrackStatusDraft    TrackStatus = "draft"
	TrackStatusPending  TrackStatus = "pending"
	TrackStatusActive   TrackStatus = "active"
	TrackStatusInactive TrackStatus = "inactive"
	TrackStatusRejected TrackStatus = "rejected"
	TrackStatusDeleted  TrackStatus = "deleted"
)

// IsValid checks if the track status is valid
func (s TrackStatus) IsValid() bool {
	switch s {
	case TrackStatusDraft, TrackStatusPending, TrackStatusActive,
		TrackStatusInactive, TrackStatusRejected, TrackStatusDeleted:
		return true
	default:
		return false
	}
}

// String returns the string representation of the track status
func (s TrackStatus) String() string {
	return string(s)
}

// TrackRepository defines the interface for track persistence operations
type TrackRepository interface {
	Create(ctx context.Context, track *Track) error
	GetByID(ctx context.Context, id string) (*Track, error)
	Update(ctx context.Context, track *Track) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*Track, error)
	SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*Track, error)
	GetByISRC(ctx context.Context, isrc string) (*Track, error)
	BatchUpdate(ctx context.Context, tracks []*Track) error
}
