package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
)

// AudioStorageService handles audio file storage operations
type AudioStorageService struct {
	storage   domain.StorageService
	tracks    domain.TrackRepository
	processor domain.AudioProcessor
}

// NewAudioStorageService creates a new audio storage service
func NewAudioStorageService(storage domain.StorageService, tracks domain.TrackRepository, processor domain.AudioProcessor) domain.AudioService {
	return &AudioStorageService{
		storage:   storage,
		tracks:    tracks,
		processor: processor,
	}
}

// Process processes an audio file and returns the results
func (s *AudioStorageService) Process(ctx context.Context, file *domain.ProcessingAudioFile, options *domain.AudioProcessOptions) (*domain.AudioProcessResult, error) {
	result, err := s.processor.Process(ctx, file, options)
	if err != nil {
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}
	return result, nil
}

// Upload stores an audio file and returns its URL
func (s *AudioStorageService) Upload(ctx context.Context, file *domain.StorageFile) error {
	if err := s.storage.Upload(ctx, file); err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

// GetURL retrieves a pre-signed URL for an audio file
func (s *AudioStorageService) GetURL(ctx context.Context, id string) (string, error) {
	track, err := s.tracks.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		return "", fmt.Errorf("track not found: %s", id)
	}
	return s.storage.GetURL(ctx, track.StoragePath)
}

// Delete deletes an audio file from storage
func (s *AudioStorageService) Delete(ctx context.Context, url string) error {
	if err := s.storage.Delete(ctx, url); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Download downloads an audio file from storage
func (s *AudioStorageService) Download(ctx context.Context, url string) (*domain.StorageFile, error) {
	file, err := s.storage.Download(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	return file, nil
}
