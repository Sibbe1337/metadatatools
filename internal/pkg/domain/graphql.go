package domain

import "time"

// TrackConnection represents a paginated connection of tracks
type TrackConnection struct {
	Edges      []*TrackEdge
	PageInfo   *PageInfo
	TotalCount int
}

// TrackEdge represents a single edge in a track connection
type TrackEdge struct {
	Node   *Track
	Cursor string
}

// PageInfo contains information about pagination
type PageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     string
	EndCursor       string
}

// TrackFilter represents filter options for track queries
type TrackFilter struct {
	Title       *string
	Artist      *string
	Album       *string
	Genre       *string
	Label       *string
	ISRC        *string
	ISWC        *string
	NeedsReview *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	SuccessCount int
	FailureCount int
	Errors       []*BatchError
}

// BatchError represents an error in a batch operation
type BatchError struct {
	TrackID string
	Message string
	Code    string
}
