package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"metadatatool/internal/pkg/domain"
	"path/filepath"
	"time"

	"github.com/dhowden/tag"
	"github.com/google/uuid"
)

// AudioService defines the interface for audio file operations
type AudioService interface {
	ProcessAudioFile(ctx context.Context, file *domain.File) (*domain.Track, error)
	GetAudioFileURL(ctx context.Context, trackID string) (string, error)
}

type audioServiceImpl struct {
	storage domain.StorageService
	tracks  domain.TrackRepository
}

// NewAudioService creates a new audio service
func NewAudioService(storage domain.StorageService, tracks domain.TrackRepository) domain.AudioService {
	return &audioServiceImpl{
		storage: storage,
		tracks:  tracks,
	}
}

// Upload handles audio file upload and metadata extraction
func (s *audioServiceImpl) Upload(ctx context.Context, file *domain.File) (string, error) {
	// Generate unique file key
	ext := filepath.Ext(file.Name)
	file.Key = fmt.Sprintf("%s%s/%s%s",
		"audio/",
		time.Now().Format("2006/01/02"),
		uuid.New().String(),
		ext,
	)

	// Read the entire file into memory for metadata extraction
	data, err := io.ReadAll(file.Content)
	if err != nil {
		return "", fmt.Errorf("failed to read file content: %w", err)
	}

	// Create a new reader for metadata extraction
	metadataReader := bytes.NewReader(data)
	metadata, err := tag.ReadFrom(metadataReader)
	if err != nil {
		return "", fmt.Errorf("failed to read audio metadata: %w", err)
	}

	// Create track from metadata
	track := &domain.Track{
		ID:        uuid.New().String(),
		Title:     metadata.Title(),
		Artist:    metadata.Artist(),
		Album:     metadata.Album(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a new reader for upload
	file.Content = bytes.NewReader(data)

	// Upload file to storage
	if err := s.storage.Upload(ctx, file); err != nil {
		return "", fmt.Errorf("failed to upload audio file: %w", err)
	}

	// Save track to database
	if err := s.tracks.Create(ctx, track); err != nil {
		// Try to cleanup uploaded file if database save fails
		_ = s.storage.Delete(ctx, file.Key)
		return "", fmt.Errorf("failed to save track metadata: %w", err)
	}

	return file.Key, nil
}

// GetURL generates a pre-signed URL for audio file download
func (s *audioServiceImpl) GetURL(ctx context.Context, id string) (string, error) {
	track, err := s.tracks.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		return "", fmt.Errorf("track not found: %s", id)
	}

	// Generate pre-signed URL with 1-hour expiry
	url, err := s.storage.GetSignedURL(ctx, track.ID, domain.SignedURLDownload, 1*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}
