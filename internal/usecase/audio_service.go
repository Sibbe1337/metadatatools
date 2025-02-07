package usecase

import (
	"context"
	"fmt"
	"io"
	"metadatatool/internal/pkg/domain"
	"os"
	"path/filepath"
	"time"
)

// AudioService implements audio processing and storage functionality
type AudioService struct {
	processor domain.AudioProcessor
	storage   domain.StorageService
	tempDir   string
}

// NewAudioService creates a new audio service
func NewAudioService(processor domain.AudioProcessor, storage domain.StorageService, tempDir string) domain.AudioService {
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	return &AudioService{
		processor: processor,
		storage:   storage,
		tempDir:   tempDir,
	}
}

// Process processes an audio file with the given options
func (s *AudioService) Process(ctx context.Context, file *domain.ProcessingAudioFile, options *domain.AudioProcessOptions) (*domain.AudioProcessResult, error) {
	// Validate file
	if err := s.validateFile(file); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Process the file
	result, err := s.processor.Process(ctx, file, options)
	if err != nil {
		return nil, fmt.Errorf("audio processing failed: %w", err)
	}

	return result, nil
}

// Upload uploads an audio file to storage
func (s *AudioService) Upload(ctx context.Context, file *domain.StorageFile) error {
	// Validate file
	if err := s.validateStorageFile(file); err != nil {
		return fmt.Errorf("file validation failed: %w", err)
	}

	// Upload to storage
	if err := s.storage.Upload(ctx, file); err != nil {
		return fmt.Errorf("storage upload failed: %w", err)
	}

	return nil
}

// Download downloads an audio file from storage
func (s *AudioService) Download(ctx context.Context, url string) (*domain.StorageFile, error) {
	// Download from storage
	file, err := s.storage.Download(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("storage download failed: %w", err)
	}

	return file, nil
}

// Delete deletes an audio file from storage
func (s *AudioService) Delete(ctx context.Context, url string) error {
	// Delete from storage
	if err := s.storage.Delete(ctx, url); err != nil {
		return fmt.Errorf("storage delete failed: %w", err)
	}

	return nil
}

// GetURL gets the URL for an audio file in storage
func (s *AudioService) GetURL(ctx context.Context, id string) (string, error) {
	// Get URL from storage
	url, err := s.storage.GetURL(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to get storage URL: %w", err)
	}

	return url, nil
}

// validateFile validates a processing audio file
func (s *AudioService) validateFile(file *domain.ProcessingAudioFile) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}
	if file.Name == "" {
		return fmt.Errorf("file name is empty")
	}
	if file.Size <= 0 {
		return fmt.Errorf("invalid file size: %d", file.Size)
	}
	if !file.Format.IsValid() {
		return fmt.Errorf("unsupported format: %s", file.Format)
	}
	return nil
}

// validateStorageFile validates a storage file
func (s *AudioService) validateStorageFile(file *domain.StorageFile) error {
	if file == nil {
		return fmt.Errorf("file is nil")
	}
	if file.Name == "" {
		return fmt.Errorf("file name is empty")
	}
	if file.Size <= 0 {
		return fmt.Errorf("invalid file size: %d", file.Size)
	}
	if file.Content == nil {
		return fmt.Errorf("file content is nil")
	}
	return nil
}

// isSupportedFormat checks if the audio format is supported
func (s *AudioService) isSupportedFormat(format string) bool {
	supportedFormats := map[string]bool{
		"mp3":  true,
		"wav":  true,
		"flac": true,
		"aac":  true,
		"ogg":  true,
		"m4a":  true,
	}
	return supportedFormats[format]
}

// createTempFile creates a temporary file for processing
func (s *AudioService) createTempFile(reader io.Reader, ext string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp(s.tempDir, fmt.Sprintf("audio-*%s", ext))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy content to temp file
	if _, err := io.Copy(tmpFile, reader); err != nil {
		os.Remove(tmpFile.Name()) // Clean up on error
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// cleanupTempFile removes a temporary file
func (s *AudioService) cleanupTempFile(path string) {
	if path != "" {
		os.Remove(path)
	}
}

// generateStorageKey generates a unique storage key for a file
func generateStorageKey(filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("audio/%s/%s%s", timestamp[:8], timestamp, ext)
}
