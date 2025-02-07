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
	processor *AudioProcessorService
}

// NewAudioStorageService creates a new audio storage service
func NewAudioStorageService(storage domain.StorageService, tracks domain.TrackRepository, processor *AudioProcessorService) domain.AudioService {
	return &AudioStorageService{
		storage:   storage,
		tracks:    tracks,
		processor: processor,
	}
}

// Upload stores an audio file and returns its URL
func (s *AudioStorageService) Upload(ctx context.Context, file *domain.StorageFile) (string, error) {
	if err := s.storage.Upload(ctx, file); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return file.Key, nil
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
