package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// AudioService handles audio file operations
type AudioService struct {
	storage   domain.StorageService
	tracks    domain.TrackRepository
	processor *AudioProcessorService
}

// NewAudioService creates a new audio service
func NewAudioService(storage domain.StorageService, tracks domain.TrackRepository, processor *AudioProcessorService) domain.AudioService {
	return &AudioService{
		storage:   storage,
		tracks:    tracks,
		processor: processor,
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
	result, err := s.processor.ProcessAudio(ctx, &domain.ProcessingAudioFile{
		Name:    file.Name,
		Content: file.Content,
		Size:    file.Size,
	}, &domain.AudioProcessOptions{
		AnalyzeAudio:    true,
		ExtractMetadata: true,
	})
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "processing_failed").Inc()
		return nil, fmt.Errorf("audio processing failed: %w", err)
	}

	// Generate storage key
	storageKey := generateStorageKey(file.Name)

	// Upload to storage
	if err := s.storage.Upload(ctx, &domain.StorageFile{
		Key:     storageKey,
		Name:    file.Name,
		Size:    file.Size,
		Content: file.Content,
	}); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "upload_failed").Inc()
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	// Create track record
	track := &domain.Track{
		ID:          uuid.New().String(),
		StoragePath: storageKey,
		FileSize:    file.Size,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata: domain.CompleteTrackMetadata{
			BasicTrackMetadata: domain.BasicTrackMetadata{
				Title:     result.Metadata.Title,
				Artist:    result.Metadata.Artist,
				Album:     result.Metadata.Album,
				Year:      result.Metadata.Year,
				Duration:  result.Metadata.Duration,
				ISRC:      result.Metadata.ISRC,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Technical: domain.AudioTechnicalMetadata{
				Format:     domain.AudioFormat(result.Metadata.Technical.Format),
				SampleRate: result.Metadata.Technical.SampleRate,
				Bitrate:    result.Metadata.Technical.Bitrate,
				Channels:   result.Metadata.Technical.Channels,
				FileSize:   file.Size,
			},
			Musical: domain.MusicalMetadata{
				BPM:    result.Metadata.Musical.BPM,
				Key:    result.Metadata.Musical.Key,
				Mode:   result.Metadata.Musical.Mode,
				Mood:   result.Metadata.Musical.Mood,
				Genre:  result.Metadata.Musical.Genre,
				Energy: result.Metadata.Musical.Energy,
				Tempo:  result.Metadata.Musical.Tempo,
			},
		},
	}

	// Save track to database
	if err := s.tracks.Create(ctx, track); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("complete_process", "db_save_failed").Inc()
		// Try to cleanup uploaded file
		_ = s.storage.Delete(ctx, storageKey)
		return nil, fmt.Errorf("failed to save track: %w", err)
	}

	// Process audio file in background
	go func() {
		bgCtx := context.Background()
		_, err := s.processor.ProcessAudio(bgCtx, &domain.ProcessingAudioFile{
			Path:   track.StoragePath,
			Format: track.Metadata.Technical.Format,
		}, &domain.AudioProcessOptions{
			AnalyzeAudio:    true,
			ExtractMetadata: true,
		})
		if err != nil {
			metrics.AudioProcessingErrors.WithLabelValues("complete_process", "ai_analysis_failed").Inc()
			// Log error but don't fail the request
			fmt.Printf("AI analysis failed for track %s: %v\n", track.ID, err)
		}
	}()

	metrics.AudioProcessingSuccess.WithLabelValues("complete_process").Inc()
	return track, nil
}

// GetURL retrieves a pre-signed URL for an audio file
func (s *AudioService) GetURL(ctx context.Context, id string) (string, error) {
	track, err := s.tracks.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get track: %w", err)
	}
	if track == nil {
		return "", fmt.Errorf("track not found: %s", id)
	}
	return s.storage.GetURL(ctx, track.StoragePath)
}

// Upload stores an audio file and returns its URL
func (s *AudioService) Upload(ctx context.Context, file *domain.StorageFile) (string, error) {
	if err := s.storage.Upload(ctx, file); err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	return file.Key, nil
}

// generateStorageKey generates a unique storage key for a file
func generateStorageKey(filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("audio/%s/%s%s", timestamp[:8], timestamp, ext)
}

func (s *AudioService) ProcessFile(ctx context.Context, file *domain.UploadedFile) error {
	// Create processing file
	processingFile := &domain.ProcessingAudioFile{
		Name:    file.Filename,
		Content: file.File,
		Size:    file.Size,
	}

	// Process audio file
	result, err := s.processor.ProcessAudio(ctx, processingFile, &domain.AudioProcessOptions{
		AnalyzeAudio:    true,
		ExtractMetadata: true,
	})
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("process_file", err.Error()).Inc()
		return fmt.Errorf("failed to process audio file: %w", err)
	}

	// Create track from results
	track := &domain.Track{
		StoragePath: processingFile.Path,
		FileSize:    processingFile.Size,
		Metadata:    result.Metadata,
	}

	if err := s.tracks.Create(ctx, track); err != nil {
		return fmt.Errorf("failed to save track: %w", err)
	}

	return nil
}

func (s *AudioService) BatchProcess(ctx context.Context, files []*domain.UploadedFile) error {
	for _, file := range files {
		processingFile := &domain.ProcessingAudioFile{
			Name:    file.Filename,
			Content: file.File,
			Size:    file.Size,
		}

		result, err := s.processor.ProcessAudio(ctx, processingFile, &domain.AudioProcessOptions{
			AnalyzeAudio:    true,
			ExtractMetadata: true,
		})
		if err != nil {
			metrics.AudioProcessingErrors.WithLabelValues("batch_process", err.Error()).Inc()
			return fmt.Errorf("failed to process audio file %s: %w", file.Filename, err)
		}

		// Create track from results
		track := &domain.Track{
			StoragePath: processingFile.Path,
			FileSize:    processingFile.Size,
			Metadata:    result.Metadata,
		}

		if err := s.tracks.Create(ctx, track); err != nil {
			return fmt.Errorf("failed to save track: %w", err)
		}
	}

	return nil
}

func (s *AudioService) ProcessAudio(ctx context.Context, file *domain.ProcessingAudioFile, options *domain.AudioProcessOptions) (*domain.Track, error) {
	// Process audio file
	result, err := s.processor.ProcessAudio(ctx, file, options)
	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("complete_process", "processing_failed").Inc()
		return nil, fmt.Errorf("failed to process audio: %w", err)
	}

	// Create track from results
	track := &domain.Track{
		StoragePath: file.Path,
		FileSize:    file.Size,
		Metadata:    result.Metadata,
	}

	if err := s.tracks.Create(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to save track: %w", err)
	}

	return track, nil
}
