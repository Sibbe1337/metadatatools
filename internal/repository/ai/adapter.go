package ai

import (
	"context"
	"fmt"
	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"time"
)

// AIServiceAdapter adapts the internal AIService to the pkg domain AIService interface
type AIServiceAdapter struct {
	internalService domain.AIService
}

// NewAIServiceAdapter creates a new AIServiceAdapter
func NewAIServiceAdapter(internalService domain.AIService) *AIServiceAdapter {
	return &AIServiceAdapter{
		internalService: internalService,
	}
}

// EnrichMetadata adapts the internal EnrichMetadata method to the pkg domain interface
func (a *AIServiceAdapter) EnrichMetadata(ctx context.Context, track *pkgdomain.Track) error {
	// Call the internal service with just the audio path string
	internalMetadata, err := a.internalService.EnrichMetadata(ctx, track.StoragePath)
	if err != nil {
		return fmt.Errorf("failed to enrich metadata: %w", err)
	}

	// Update track metadata using available setter methods
	track.SetTitle(internalMetadata.Title)
	track.SetArtist(internalMetadata.Artist)
	track.SetAlbum(internalMetadata.Album)
	if len(internalMetadata.Genre) > 0 {
		track.SetGenre(internalMetadata.Genre[0])
	}
	track.SetYear(internalMetadata.Year)
	track.Metadata.Additional.CustomFields["language"] = internalMetadata.Language
	if len(internalMetadata.Mood) > 0 {
		track.SetMood(internalMetadata.Mood[0])
	}
	track.SetBPM(internalMetadata.Tempo)
	track.SetKey(internalMetadata.Key)
	track.SetAudioFormat(internalMetadata.TimeSignature)
	track.SetDuration(internalMetadata.Duration)

	// Initialize AI metadata if not present
	if track.Metadata.AI == nil {
		track.Metadata.AI = &pkgdomain.TrackAIMetadata{
			Model:       "qwen2",
			Version:     "1.0",
			ProcessedAt: time.Now(),
			Tags:        []string{},
		}
	}

	// Update AI metadata fields
	track.Metadata.AI.Tags = internalMetadata.Tags
	track.Metadata.AI.Confidence = internalMetadata.Confidence
	track.Metadata.AI.NeedsReview = internalMetadata.Confidence < 0.85
	if track.Metadata.AI.NeedsReview {
		track.Metadata.AI.ReviewReason = "Low confidence score"
		track.Metadata.AI.ValidationIssues = []pkgdomain.ValidationIssue{
			{
				Field:       "confidence",
				Severity:    "warning",
				Description: "Confidence score below threshold",
			},
		}
	}

	return nil
}

// ValidateMetadata adapts the internal ValidateMetadata method to the pkg domain interface
func (a *AIServiceAdapter) ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (float64, error) {
	// Create internal metadata for validation using getter methods
	internalMetadata := &domain.AIMetadata{
		Title:         track.Title(),
		Artist:        track.Artist(),
		Album:         track.Album(),
		Genre:         []string{track.Genre()},
		Year:          track.Year(),
		Language:      track.Metadata.Additional.CustomFields["language"],
		Mood:          []string{track.Mood()},
		Tempo:         track.BPM(),
		Key:           track.Key(),
		TimeSignature: track.AudioFormat(),
		Duration:      track.Duration(),
		Tags:          track.Metadata.AI.Tags,
		Confidence:    track.Metadata.AI.Confidence,
	}

	// Call internal service
	valid, err := a.internalService.ValidateMetadata(ctx, internalMetadata)
	if err != nil {
		return 0.0, fmt.Errorf("failed to validate metadata: %w", err)
	}

	// Convert boolean to confidence score
	if valid {
		return 1.0, nil
	}
	return 0.0, nil
}

// BatchProcess adapts the internal BatchProcess method to the pkg domain interface
func (a *AIServiceAdapter) BatchProcess(ctx context.Context, tracks []*pkgdomain.Track) error {
	// Process tracks sequentially since internal service doesn't support batch processing
	for _, track := range tracks {
		if err := a.EnrichMetadata(ctx, track); err != nil {
			return fmt.Errorf("failed to process track %s: %w", track.ID, err)
		}
	}
	return nil
}
