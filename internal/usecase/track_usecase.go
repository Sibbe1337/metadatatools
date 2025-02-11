package usecase

import (
	"context"
	"fmt"
	"io"
	"time"

	"metadatatool/internal/pkg/domain"
)

// TrackFilter represents filter criteria for listing tracks
type TrackFilter struct {
	Title  string
	Artist string
	Album  string
	Genre  string
	ISRC   string
	Offset int
	Limit  int
}

type TrackUseCase struct {
	trackRepo      domain.TrackRepository
	storageService domain.StorageService
	aiService      domain.AIService
	queueService   domain.QueueService
}

func NewTrackUseCase(
	trackRepo domain.TrackRepository,
	storageService domain.StorageService,
	aiService domain.AIService,
	queueService domain.QueueService,
) *TrackUseCase {
	return &TrackUseCase{
		trackRepo:      trackRepo,
		storageService: storageService,
		aiService:      aiService,
		queueService:   queueService,
	}
}

func (u *TrackUseCase) CreateTrack(ctx context.Context, track *domain.Track, audioData io.Reader) error {
	// Upload audio file
	if err := u.storageService.UploadAudio(ctx, audioData, track.ID); err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	// Save track to database
	if err := u.trackRepo.Create(ctx, track); err != nil {
		return fmt.Errorf("failed to create track: %w", err)
	}

	// Queue track for processing
	msg := &domain.Message{
		ID:        track.ID,
		Type:      "track_processing",
		Data:      map[string]interface{}{"track_id": track.ID},
		Status:    domain.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := u.queueService.Publish(ctx, "track_processing", msg, domain.PriorityMedium); err != nil {
		return fmt.Errorf("failed to queue track for processing: %w", err)
	}

	return nil
}

func (u *TrackUseCase) GetTrack(ctx context.Context, id string) (*domain.Track, error) {
	track, err := u.trackRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get track: %w", err)
	}
	return track, nil
}

func (u *TrackUseCase) UpdateTrack(ctx context.Context, track *domain.Track) error {
	if err := u.trackRepo.Update(ctx, track); err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}
	return nil
}

func (u *TrackUseCase) DeleteTrack(ctx context.Context, id string) error {
	track, err := u.trackRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Delete audio file
	if err := u.storageService.DeleteAudio(ctx, track.StoragePath); err != nil {
		return fmt.Errorf("failed to delete audio: %w", err)
	}

	// Delete track from database
	if err := u.trackRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}

	return nil
}

func (u *TrackUseCase) ListTracks(ctx context.Context, filter TrackFilter) ([]*domain.Track, error) {
	// Convert our filter to map[string]interface{}
	filterMap := map[string]interface{}{
		"title":  filter.Title,
		"artist": filter.Artist,
		"album":  filter.Album,
		"genre":  filter.Genre,
		"isrc":   filter.ISRC,
	}

	// Get tracks from repository with pagination
	tracks, err := u.trackRepo.List(ctx, filterMap, filter.Offset, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}

	return tracks, nil
}

func (u *TrackUseCase) ValidateERN(ctx context.Context, data []byte) error {
	// TODO: Implement ERN validation
	return nil
}

func (u *TrackUseCase) ImportERN(ctx context.Context, data []byte) error {
	// TODO: Implement ERN import
	return nil
}

func (u *TrackUseCase) ExportERN(ctx context.Context, trackIDs []string) ([]byte, error) {
	// TODO: Implement ERN export
	return nil, nil
}

func (u *TrackUseCase) GetAudioURL(ctx context.Context, id string) (string, error) {
	track, err := u.trackRepo.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get track: %w", err)
	}

	// Generate signed URL for audio file
	signedURL, err := u.storageService.GetSignedURL(ctx, track.StoragePath, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return signedURL, nil
}

func (u *TrackUseCase) UploadAudio(ctx context.Context, id string, audioData io.Reader) error {
	track, err := u.trackRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get track: %w", err)
	}

	// Delete old audio file if it exists
	if track.StoragePath != "" {
		if err := u.storageService.DeleteAudio(ctx, track.StoragePath); err != nil {
			return fmt.Errorf("failed to delete old audio: %w", err)
		}
	}

	// Upload new audio file
	if err := u.storageService.UploadAudio(ctx, audioData, id); err != nil {
		return fmt.Errorf("failed to upload audio: %w", err)
	}

	// Update track with new storage path
	track.StoragePath = id
	if err := u.trackRepo.Update(ctx, track); err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}

	return nil
}
