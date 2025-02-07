package resolvers

import (
	"context"
	"metadatatool/internal/pkg/domain"
)

func (r *trackResolver) AIMetadata(ctx context.Context, obj *domain.Track) (*domain.AIMetadata, error) {
	// Return AI metadata directly from track object
	return obj.AIMetadata, nil
}

func (r *trackResolver) Metadata(ctx context.Context, obj *domain.Track) (*domain.Metadata, error) {
	// Return metadata directly from track object
	return &obj.Metadata, nil
}
