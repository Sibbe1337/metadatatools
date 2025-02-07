package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"time"

	"github.com/google/uuid"
)

// AudioProcessorService handles audio processing operations
type AudioProcessorService struct {
	storage   domain.StorageService
	tracks    domain.TrackRepository
	ai        domain.AIService
	processor domain.AudioProcessor
}

// NewAudioProcessorService creates a new audio processor service
func NewAudioProcessorService(storage domain.StorageService, tracks domain.TrackRepository, ai domain.AIService, processor domain.AudioProcessor) *AudioProcessorService {
	return &AudioProcessorService{
		storage:   storage,
		tracks:    tracks,
		ai:        ai,
		processor: processor,
	}
}

// ProcessAudio processes an audio file and creates a track record
func (s *AudioProcessorService) ProcessAudio(ctx context.Context, file *domain.ProcessingAudioFile, options *domain.AudioProcessOptions) (*domain.Track, error) {
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("complete_process"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("complete_process", "started").Inc()

	// Process audio file
	result, err := s.processor.Process(ctx, file, options)
	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("complete_process", "processing_failed").Inc()
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	// Create track record with complete metadata
	track := &domain.Track{
		ID:          uuid.New().String(),
		StoragePath: options.FilePath,
		FileSize:    result.Metadata.Technical.FileSize,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    *result.Metadata,
	}

	// Save track to database
	if err := s.tracks.Create(ctx, track); err != nil {
		metrics.AudioOpErrors.WithLabelValues("complete_process", "db_save_failed").Inc()
		// Try to cleanup uploaded file
		_ = s.storage.Delete(ctx, options.FilePath)
		return nil, fmt.Errorf("failed to save track: %w", err)
	}

	// Trigger AI analysis in background
	go func() {
		bgCtx := context.Background()
		if err := s.ai.EnrichMetadata(bgCtx, track); err != nil {
			metrics.AudioOpErrors.WithLabelValues("complete_process", "ai_analysis_failed").Inc()
			// Log error but don't fail the request
			fmt.Printf("AI analysis failed for track %s: %v\n", track.ID, err)
		}
	}()

	metrics.AudioOps.WithLabelValues("complete_process", "completed").Inc()
	return track, nil
}

// BatchProcess processes multiple audio files
func (s *AudioProcessorService) BatchProcess(ctx context.Context, files []*domain.ProcessingAudioFile, options *domain.AudioProcessOptions) ([]*domain.Track, error) {
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("batch_process"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("batch_process", "started").Inc()

	var tracks []*domain.Track
	for _, file := range files {
		track, err := s.ProcessAudio(ctx, file, options)
		if err != nil {
			metrics.AudioOpErrors.WithLabelValues("batch_process", "processing_failed").Inc()
			return nil, fmt.Errorf("failed to process audio file: %w", err)
		}
		tracks = append(tracks, track)
	}

	metrics.AudioOps.WithLabelValues("batch_process", "completed").Inc()
	return tracks, nil
}
