package resolvers

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/google/uuid"
)

func (r *mutationResolver) CreateTrack(ctx context.Context, input domain.CreateTrackInput) (*domain.Track, error) {
	// Create track
	track := &domain.Track{
		ID:        uuid.New().String(),
		Title:     input.Title,
		Artist:    input.Artist,
		Album:     stringValue(input.Album),
		Genre:     stringValue(input.Genre),
		Year:      intValue(input.Year),
		Label:     stringValue(input.Label),
		Territory: stringValue(input.Territory),
		ISRC:      stringValue(input.ISRC),
		ISWC:      stringValue(input.ISWC),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Handle audio file if provided
	if input.AudioFile != nil {
		// Store file
		key := fmt.Sprintf("audio/%s/%s", track.ID, input.AudioFile.Filename)
		if err := r.StorageService.Upload(ctx, key, input.AudioFile.File); err != nil {
			return nil, fmt.Errorf("failed to upload audio file: %w", err)
		}
		track.FilePath = key
		track.FileSize = input.AudioFile.Size
	}

	// Save track
	if err := r.TrackRepo.Create(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to create track: %w", err)
	}

	return track, nil
}

// Helper functions for handling optional values
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func (r *mutationResolver) UpdateTrack(ctx context.Context, input domain.UpdateTrackInput) (*domain.Track, error) {
	// Get existing track
	track, err := r.TrackRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	// Update fields if provided
	if input.Title != nil {
		track.Title = *input.Title
	}
	if input.Artist != nil {
		track.Artist = *input.Artist
	}
	if input.Album != nil {
		track.Album = *input.Album
	}
	if input.Genre != nil {
		track.Genre = *input.Genre
	}
	if input.Year != nil {
		track.Year = *input.Year
	}
	if input.Label != nil {
		track.Label = *input.Label
	}
	if input.Territory != nil {
		track.Territory = *input.Territory
	}
	if input.ISRC != nil {
		track.ISRC = *input.ISRC
	}
	if input.ISWC != nil {
		track.ISWC = *input.ISWC
	}

	// Update metadata if provided
	if input.Metadata != nil {
		if input.Metadata.ISRC != nil {
			track.Metadata.ISRC = *input.Metadata.ISRC
		}
		if input.Metadata.ISWC != nil {
			track.Metadata.ISWC = *input.Metadata.ISWC
		}
		if input.Metadata.BPM != nil {
			track.Metadata.BPM = *input.Metadata.BPM
		}
		if input.Metadata.Key != nil {
			track.Metadata.Key = *input.Metadata.Key
		}
		if input.Metadata.Mood != nil {
			track.Metadata.Mood = *input.Metadata.Mood
		}
		if input.Metadata.Labels != nil {
			track.Metadata.Labels = input.Metadata.Labels
		}
		if input.Metadata.CustomFields != nil {
			track.Metadata.CustomFields = input.Metadata.CustomFields
		}
	}

	track.UpdatedAt = time.Now()

	// Save changes
	if err := r.TrackRepo.Update(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to update track: %w", err)
	}

	return track, nil
}

func (r *mutationResolver) DeleteTrack(ctx context.Context, id string) (bool, error) {
	// Get track to check if it exists
	track, err := r.TrackRepo.GetByID(ctx, id)
	if err != nil {
		return false, fmt.Errorf("failed to get track: %w", err)
	}

	// Delete audio file if exists
	if track.FilePath != "" {
		if err := r.StorageService.Delete(ctx, track.FilePath); err != nil {
			return false, fmt.Errorf("failed to delete audio file: %w", err)
		}
	}

	// Delete track
	if err := r.TrackRepo.Delete(ctx, id); err != nil {
		return false, fmt.Errorf("failed to delete track: %w", err)
	}

	return true, nil
}

func (r *mutationResolver) BatchProcessTracks(ctx context.Context, ids []string) (*domain.BatchResult, error) {
	result := &domain.BatchResult{}

	for _, id := range ids {
		track, err := r.TrackRepo.GetByID(ctx, id)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, &domain.BatchError{
				TrackID: id,
				Message: fmt.Sprintf("failed to get track: %v", err),
				Code:    "NOT_FOUND",
			})
			continue
		}

		if err := r.AIService.EnrichMetadata(ctx, track); err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, &domain.BatchError{
				TrackID: id,
				Message: fmt.Sprintf("failed to enrich metadata: %v", err),
				Code:    "AI_ERROR",
			})
			continue
		}

		result.SuccessCount++
	}

	return result, nil
}

func (r *mutationResolver) EnrichTrackMetadata(ctx context.Context, id string) (*domain.Track, error) {
	// Get track
	track, err := r.TrackRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}

	// Enrich metadata
	if err := r.AIService.EnrichMetadata(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to enrich metadata: %w", err)
	}

	// Save changes
	if err := r.TrackRepo.Update(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to update track: %w", err)
	}

	return track, nil
}

func (r *mutationResolver) ValidateTrackMetadata(ctx context.Context, id string) (*domain.BatchResult, error) {
	result := &domain.BatchResult{}

	// Get track
	track, err := r.TrackRepo.GetByID(ctx, id)
	if err != nil {
		result.FailureCount = 1
		result.Errors = append(result.Errors, &domain.BatchError{
			TrackID: id,
			Message: fmt.Sprintf("failed to get track: %v", err),
			Code:    "NOT_FOUND",
		})
		return result, nil
	}

	// Validate metadata
	confidence, err := r.AIService.ValidateMetadata(ctx, track)
	if err != nil {
		result.FailureCount = 1
		result.Errors = append(result.Errors, &domain.BatchError{
			TrackID: id,
			Message: fmt.Sprintf("failed to validate metadata: %v", err),
			Code:    "VALIDATION_ERROR",
		})
		return result, nil
	}

	if confidence < 0.85 {
		result.FailureCount = 1
		result.Errors = append(result.Errors, &domain.BatchError{
			TrackID: id,
			Message: fmt.Sprintf("low confidence score: %.2f", confidence),
			Code:    "LOW_CONFIDENCE",
		})
	} else {
		result.SuccessCount = 1
	}

	return result, nil
}

func (r *mutationResolver) ExportToDDEX(ctx context.Context, ids []string) (string, error) {
	var tracks []*domain.Track
	for _, id := range ids {
		track, err := r.TrackRepo.GetByID(ctx, id)
		if err != nil {
			return "", fmt.Errorf("failed to get track %s: %w", id, err)
		}
		tracks = append(tracks, track)
	}

	// Export to DDEX
	ddexXML, err := r.DDEXService.ExportTracks(ctx, tracks)
	if err != nil {
		return "", fmt.Errorf("failed to export tracks to DDEX: %w", err)
	}

	return ddexXML, nil
}
