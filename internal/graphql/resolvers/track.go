package resolvers

import (
	"context"
	"metadatatool/internal/pkg/domain"
)

func (r *trackResolver) AIMetadata(ctx context.Context, obj *domain.Track) (*domain.AIMetadata, error) {
	// Convert TrackAIMetadata to AIMetadata
	if obj.Metadata.AI == nil {
		return nil, nil
	}

	return &domain.AIMetadata{
		Provider:     domain.AIProviderQwen2,
		Energy:       0, // These fields are not available in TrackAIMetadata
		Danceability: 0,
		ProcessedAt:  obj.Metadata.AI.ProcessedAt,
		ProcessingMs: 0,
		NeedsReview:  obj.Metadata.AI.NeedsReview,
		ReviewReason: obj.Metadata.AI.ReviewReason,
	}, nil
}

func (r *trackResolver) Metadata(ctx context.Context, obj *domain.Track) (*domain.Metadata, error) {
	// Convert CompleteTrackMetadata to Metadata
	var labels []string
	for tag := range obj.Metadata.Additional.CustomTags {
		labels = append(labels, tag)
	}

	return &domain.Metadata{
		ISRC:         obj.ISRC(),
		ISWC:         obj.ISWC(),
		BPM:          obj.BPM(),
		Key:          obj.Key(),
		Mood:         obj.Mood(),
		Labels:       labels,
		AITags:       obj.AITags(),
		Confidence:   obj.AIConfidence(),
		ModelVersion: obj.ModelVersion(),
		CustomFields: obj.Metadata.Additional.CustomFields,
	}, nil
}
