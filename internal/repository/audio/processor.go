package audio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// Processor implements domain.AudioProcessor
type Processor struct {
	// Dependencies could be added here
}

// NewProcessor creates a new audio processor
func NewProcessor() domain.AudioProcessor {
	return &Processor{}
}

// Process processes an audio file and returns its metadata
func (p *Processor) Process(ctx context.Context, file io.Reader, filename string) (*domain.AudioMetadata, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("process"))
	defer timer.ObserveDuration()

	// Read entire file into memory to allow seeking
	data, err := io.ReadAll(file)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("process", "read_failed").Inc()
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// First validate the format
	format, err := p.detectFormat(filename)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("process", "invalid_format").Inc()
		return nil, fmt.Errorf("invalid audio format: %w", err)
	}

	if err := p.ValidateFormat(ctx, bytes.NewReader(data), format); err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("process", "validation_failed").Inc()
		return nil, fmt.Errorf("audio validation failed: %w", err)
	}

	// Extract metadata
	metadata, err := p.ExtractMetadata(ctx, bytes.NewReader(data), format)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("process", "metadata_extraction_failed").Inc()
		return nil, fmt.Errorf("metadata extraction failed: %w", err)
	}

	// Update file size
	metadata.FileSize = int64(len(data))
	metrics.AudioFileSize.WithLabelValues(string(format)).Observe(float64(metadata.FileSize))
	metrics.AudioFilesProcessed.WithLabelValues(string(format)).Inc()
	metrics.AudioProcessingSuccess.WithLabelValues("process").Inc()

	return metadata, nil
}

// Convert converts an audio file to a different format
func (p *Processor) Convert(ctx context.Context, file io.Reader, from, to domain.AudioFormat) (io.Reader, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("convert"))
	defer timer.ObserveDuration()

	// TODO: Implement audio format conversion
	// This would typically use ffmpeg or a similar tool
	return nil, fmt.Errorf("audio conversion not implemented")
}

// ExtractMetadata extracts metadata from an audio file
func (p *Processor) ExtractMetadata(ctx context.Context, file io.Reader, format domain.AudioFormat) (*domain.AudioMetadata, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("extract_metadata"))
	defer timer.ObserveDuration()

	// Convert to ReadSeeker if not already
	var readSeeker io.ReadSeeker
	if seeker, ok := file.(io.ReadSeeker); ok {
		readSeeker = seeker
	} else {
		// Read entire file into memory
		data, err := io.ReadAll(file)
		if err != nil {
			metrics.AudioProcessingErrors.WithLabelValues("extract_metadata", "read_failed").Inc()
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		readSeeker = bytes.NewReader(data)
	}

	// Read ID3 tags using the tag library
	m, err := tag.ReadFrom(readSeeker)
	if err != nil {
		metrics.AudioProcessingErrors.WithLabelValues("extract_metadata", "tag_read_failed").Inc()
		return nil, fmt.Errorf("failed to read audio tags: %w", err)
	}

	// Get track number, handling multiple return values
	trackNum, _ := m.Track()

	metadata := &domain.AudioMetadata{
		Title:       m.Title(),
		Artist:      m.Artist(),
		Album:       m.Album(),
		Year:        m.Year(),
		Genre:       m.Genre(),
		TrackNumber: trackNum,
		Format:      format,
		ModifiedAt:  time.Now(),
	}

	// Get additional metadata if available
	if comment, ok := m.Raw()["Comment"]; ok {
		metadata.Comments = comment.(string)
	}
	if composer, ok := m.Raw()["Composer"]; ok {
		metadata.Composer = composer.(string)
	}
	if publisher, ok := m.Raw()["Publisher"]; ok {
		metadata.Publisher = publisher.(string)
	}
	if lyrics, ok := m.Raw()["Lyrics"]; ok {
		metadata.Lyrics = lyrics.(string)
	}

	// Get cover art if available
	if picture := m.Picture(); picture != nil {
		metadata.CoverArt = picture.Data
	}

	// Store any custom tags
	metadata.CustomTags = make(map[string]string)
	for key, value := range m.Raw() {
		if str, ok := value.(string); ok {
			metadata.CustomTags[key] = str
		}
	}

	metrics.AudioProcessingSuccess.WithLabelValues("extract_metadata").Inc()
	return metadata, nil
}

// ValidateFormat validates if the file is a valid audio file
func (p *Processor) ValidateFormat(ctx context.Context, file io.Reader, format domain.AudioFormat) error {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("validate"))
	defer timer.ObserveDuration()

	// TODO: Implement proper audio format validation
	// This would typically check file headers and structure

	switch format {
	case domain.FormatMP3, domain.FormatWAV, domain.FormatFLAC, domain.FormatM4A, domain.FormatAAC:
		return nil
	default:
		metrics.AudioProcessingErrors.WithLabelValues("validate", "unsupported_format").Inc()
		return fmt.Errorf("unsupported audio format: %s", format)
	}
}

// AnalyzeAudio performs audio analysis
func (p *Processor) AnalyzeAudio(ctx context.Context, file io.Reader, format domain.AudioFormat) (*domain.AudioAnalysis, error) {
	timer := metrics.NewTimer(metrics.AudioProcessingDuration.WithLabelValues("analyze"))
	defer timer.ObserveDuration()

	// TODO: Implement audio analysis
	// This would typically use a DSP library or external service
	return nil, fmt.Errorf("audio analysis not implemented")
}

// detectFormat detects the audio format from the filename
func (p *Processor) detectFormat(filename string) (domain.AudioFormat, error) {
	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".mp3"):
		return domain.FormatMP3, nil
	case strings.HasSuffix(strings.ToLower(filename), ".wav"):
		return domain.FormatWAV, nil
	case strings.HasSuffix(strings.ToLower(filename), ".flac"):
		return domain.FormatFLAC, nil
	case strings.HasSuffix(strings.ToLower(filename), ".m4a"):
		return domain.FormatM4A, nil
	case strings.HasSuffix(strings.ToLower(filename), ".aac"):
		return domain.FormatAAC, nil
	default:
		return "", fmt.Errorf("unsupported file extension: %s", filename)
	}
}
