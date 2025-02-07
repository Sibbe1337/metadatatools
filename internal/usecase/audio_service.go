package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"path/filepath"
	"time"
)

// AudioService handles the complete audio processing workflow
type AudioService struct {
	storage   domain.StorageService
	processor domain.AudioProcessor
	tracks    domain.TrackRepository
	ai        domain.AIService
}

// NewAudioService creates a new audio service
func NewAudioService(
	storage domain.StorageService,
	processor domain.AudioProcessor,
	tracks domain.TrackRepository,
	ai domain.AIService,
) domain.AudioService {
	return &AudioService{
		storage:   storage,
		processor: processor,
		tracks:    tracks,
		ai:        ai,
	}
}

// ProcessAudioFile processes an uploaded audio file
func (s *AudioService) ProcessAudioFile(ctx context.Context, file *domain.File) (*domain.Track, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("complete_process"))
	defer timer.ObserveDuration()

	// Validate file
	if err := s.storage.ValidateUpload(ctx, file.Name, file.Size, ""); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "validation_failed").Inc()
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Process audio file
	metadata, err := s.processor.Process(ctx, file.Content, file.Name)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "processing_failed").Inc()
		return nil, fmt.Errorf("audio processing failed: %w", err)
	}

	// Generate storage key
	storageKey := generateStorageKey(file.Name)

	// Upload to storage
	if err := s.storage.Upload(ctx, storageKey, file.Content); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "upload_failed").Inc()
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	// Create track record
	track := &domain.Track{
		Title:       metadata.Title,
		Artist:      metadata.Artist,
		Album:       metadata.Album,
		Genre:       metadata.Genre,
		Year:        metadata.Year,
		Duration:    metadata.Duration,
		FilePath:    storageKey,
		FileSize:    metadata.FileSize,
		AudioFormat: string(metadata.Format),
		ISRC:        metadata.ISRC,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save track to database
	if err := s.tracks.Create(ctx, track); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "db_save_failed").Inc()
		// Try to cleanup uploaded file
		_ = s.storage.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to save track: %w", err)
	}

	// Trigger AI analysis in background
	go func() {
		bgCtx := context.Background()
		if err := s.ai.EnrichMetadata(bgCtx, track); err != nil {
			metrics.AudioProcessingErrors.WithLabelValues("complete_process", "ai_analysis_failed").Inc()
			// Log error but don't fail the request
			fmt.Printf("AI analysis failed for track %s: %v\n", track.ID, err)
		}
	}()

	metrics.AudioProcessingSuccess.WithLabelValues("complete_process").Inc()
	return track, nil
}

// GetURL generates a pre-signed URL for file access
func (s *AudioService) GetURL(ctx context.Context, trackID string) (string, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("get_url"))
	defer timer.ObserveDuration()

	// Get track from database
	track, err := s.tracks.GetByID(ctx, trackID)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("get_url", "track_not_found").Inc()
		return "", fmt.Errorf("track not found: %w", err)
	}

	// Generate pre-signed URL
	url, err := s.storage.GetSignedURL(ctx, track.FilePath, domain.SignedURLDownload, 15*time.Minute)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("get_url", "url_generation_failed").Inc()
		return "", fmt.Errorf("failed to generate URL: %w", err)
	}

	metrics.AudioProcessingSuccess.WithLabelValues("get_url").Inc()
	return url, nil
}

// Upload handles the complete audio file upload workflow
func (s *AudioService) Upload(ctx context.Context, file *domain.File) (string, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("upload"))
	defer timer.ObserveDuration()

	// Validate file
	if err := s.storage.ValidateUpload(ctx, file.Name, file.Size, ""); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("upload", "validation_failed").Inc()
		return "", fmt.Errorf("file validation failed: %w", err)
	}

	// Process audio file
	metadata, err := s.processor.Process(ctx, file.Content, file.Name)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("upload", "processing_failed").Inc()
		return "", fmt.Errorf("audio processing failed: %w", err)
	}

	// Generate storage key
	storageKey := generateStorageKey(file.Name)

	// Upload to storage
	if err := s.storage.Upload(ctx, storageKey, file.Content); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("upload", "upload_failed").Inc()
		return "", fmt.Errorf("file upload failed: %w", err)
	}

	// Create track record
	track := &domain.Track{
		Title:       metadata.Title,
		Artist:      metadata.Artist,
		Album:       metadata.Album,
		Genre:       metadata.Genre,
		Year:        metadata.Year,
		Duration:    metadata.Duration,
		FilePath:    storageKey,
		FileSize:    metadata.FileSize,
		AudioFormat: string(metadata.Format),
		ISRC:        metadata.ISRC,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save track to database
	if err := s.tracks.Create(ctx, track); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("upload", "db_save_failed").Inc()
		// Try to cleanup uploaded file
		_ = s.storage.Delete(ctx, storageKey)
		return "", fmt.Errorf("failed to save track: %w", err)
	}

	// Trigger AI analysis in background
	go func() {
		bgCtx := context.Background()
		if err := s.ai.EnrichMetadata(bgCtx, track); err != nil {
			metrics.AudioProcessingErrors.WithLabelValues("upload", "ai_analysis_failed").Inc()
			// Log error but don't fail the request
			fmt.Printf("AI analysis failed for track %s: %v\n", track.ID, err)
		}
	}()

	metrics.AudioProcessingSuccess.WithLabelValues("upload").Inc()
	return storageKey, nil
}

// generateStorageKey generates a unique storage key for a file
func generateStorageKey(filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("audio/%s/%s%s", timestamp[:8], timestamp, ext)
}
