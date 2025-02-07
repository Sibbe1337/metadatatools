package resolvers

import (
	"context"
	"encoding/base64"
	"fmt"
	"metadatatool/internal/pkg/domain"
)

func (r *queryResolver) Track(ctx context.Context, id string) (*domain.Track, error) {
	track, err := r.TrackRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	return track, nil
}

func (r *queryResolver) Tracks(ctx context.Context, first *int, after *string, filter *domain.TrackFilter, orderBy *string) (*domain.TrackConnection, error) {
	// Default values
	limit := 10
	if first != nil {
		limit = *first
	}

	// Convert cursor to offset
	offset := 0
	if after != nil {
		// TODO: Implement cursor-based pagination
		// For now, using simple offset-based pagination
		offset = 0
	}

	// Build query from filter
	query := make(map[string]interface{})
	if filter != nil {
		if filter.Title != nil {
			query["title"] = *filter.Title
		}
		if filter.Artist != nil {
			query["artist"] = *filter.Artist
		}
		// ... add other filter fields
	}

	// Get tracks
	tracks, err := r.TrackRepo.SearchByMetadata(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}

	// Build connection
	edges := make([]*domain.TrackEdge, 0, len(tracks))
	for _, track := range tracks {
		edges = append(edges, &domain.TrackEdge{
			Node:   track,
			Cursor: encodeCursor(track.ID),
		})
	}

	return &domain.TrackConnection{
		Edges: edges,
		PageInfo: &domain.PageInfo{
			HasNextPage:     len(tracks) == limit,
			HasPreviousPage: offset > 0,
			StartCursor:     encodeCursor(tracks[0].ID),
			EndCursor:       encodeCursor(tracks[len(tracks)-1].ID),
		},
		TotalCount: len(tracks),
	}, nil
}

func (r *queryResolver) SearchTracks(ctx context.Context, query string) ([]*domain.Track, error) {
	// Implement full-text search
	searchQuery := map[string]interface{}{
		"$text": map[string]interface{}{
			"$search": query,
		},
	}

	tracks, err := r.TrackRepo.SearchByMetadata(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search tracks: %w", err)
	}
	return tracks, nil
}

func (r *queryResolver) TracksNeedingReview(ctx context.Context, first *int, after *string) (*domain.TrackConnection, error) {
	// Default values
	limit := 10
	if first != nil {
		limit = *first
	}

	// Query tracks needing review
	query := map[string]interface{}{
		"needs_review": true,
	}

	tracks, err := r.TrackRepo.SearchByMetadata(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tracks needing review: %w", err)
	}

	// Build connection
	edges := make([]*domain.TrackEdge, 0, len(tracks))
	for _, track := range tracks {
		edges = append(edges, &domain.TrackEdge{
			Node:   track,
			Cursor: encodeCursor(track.ID),
		})
	}

	return &domain.TrackConnection{
		Edges: edges,
		PageInfo: &domain.PageInfo{
			HasNextPage:     len(tracks) == limit,
			HasPreviousPage: false, // First page
			StartCursor:     encodeCursor(tracks[0].ID),
			EndCursor:       encodeCursor(tracks[len(tracks)-1].ID),
		},
		TotalCount: len(tracks),
	}, nil
}

// Helper function to encode cursor
func encodeCursor(id string) string {
	return base64.StdEncoding.EncodeToString([]byte(id))
}

// Helper function to decode cursor
func decodeCursor(cursor string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("invalid cursor: %w", err)
	}
	return string(bytes), nil
}
