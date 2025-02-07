package generated

import (
	"context"
	"metadatatool/internal/pkg/domain"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	TrackRepo      domain.TrackRepository
	AIService      domain.AIService
	DDEXService    domain.DDEXService
	AuthService    domain.AuthService
	StorageService domain.StorageService
}

// QueryResolver defines the query resolver interface
type QueryResolver interface {
	Track(ctx context.Context, id string) (*domain.Track, error)
	Tracks(ctx context.Context, first *int, after *string, filter *domain.TrackFilter, orderBy *string) (*domain.TrackConnection, error)
	SearchTracks(ctx context.Context, query string) ([]*domain.Track, error)
	TracksNeedingReview(ctx context.Context, first *int, after *string) (*domain.TrackConnection, error)
}

// MutationResolver defines the mutation resolver interface
type MutationResolver interface {
	CreateTrack(ctx context.Context, input domain.CreateTrackInput) (*domain.Track, error)
	UpdateTrack(ctx context.Context, input domain.UpdateTrackInput) (*domain.Track, error)
	DeleteTrack(ctx context.Context, id string) (bool, error)
	BatchProcessTracks(ctx context.Context, ids []string) (*domain.BatchResult, error)
	EnrichTrackMetadata(ctx context.Context, id string) (*domain.Track, error)
	ValidateTrackMetadata(ctx context.Context, id string) (*domain.BatchResult, error)
	ExportToDDEX(ctx context.Context, ids []string) (string, error)
}

// SubscriptionResolver defines the subscription resolver interface
type SubscriptionResolver interface {
	TrackUpdated(ctx context.Context, id string) (<-chan *domain.Track, error)
	BatchProcessingProgress(ctx context.Context, batchID string) (<-chan *domain.BatchResult, error)
}

// TrackResolver defines the track resolver interface
type TrackResolver interface {
	AIMetadata(ctx context.Context, obj *domain.Track) (*domain.AIMetadata, error)
	Metadata(ctx context.Context, obj *domain.Track) (*domain.Metadata, error)
}

func NewResolver(
	trackRepo domain.TrackRepository,
	aiService domain.AIService,
	ddexService domain.DDEXService,
	authService domain.AuthService,
	storageService domain.StorageService,
) *Resolver {
	return &Resolver{
		TrackRepo:      trackRepo,
		AIService:      aiService,
		DDEXService:    ddexService,
		AuthService:    authService,
		StorageService: storageService,
	}
}

// ResolverRoot defines the root resolver interface
type ResolverRoot interface {
	Query() QueryResolver
	Mutation() MutationResolver
	Subscription() SubscriptionResolver
	Track() TrackResolver
}
