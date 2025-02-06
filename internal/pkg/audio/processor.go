package audio

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

// ProcessingOptions defines options for audio file processing
type ProcessingOptions struct {
	ValidateOnly bool              // Only validate without processing
	MaxDuration  time.Duration     // Maximum allowed audio duration
	Progress     chan float64      // Optional channel for progress updates
	Metadata     map[string]string // Additional metadata to store
	Format       AudioFormat       // Target format for conversion
	Normalize    bool              // Apply audio normalization
	TargetLUFS   float64           // Target loudness for normalization
	Quality      int               // Encoding quality (0-9)
}

// ProcessingResult contains the results of audio processing
type ProcessingResult struct {
	Duration    time.Duration // Audio duration
	Format      string        // Audio format
	Bitrate     int           // Audio bitrate
	Channels    int           // Number of audio channels
	Metadata    tag.Metadata  // Extracted metadata
	FileSize    int64         // Processed file size
	ProcessedAt time.Time     // Processing timestamp
	Quality     *AudioQuality // Quality analysis results
	Waveform    []float64     // Waveform data for visualization
}

// ProcessAudio processes an audio file with the given options
func ProcessAudio(ctx context.Context, content io.Reader, size int64, filename string, opts *ProcessingOptions) (*ProcessingResult, error) {
	// Set default options if not provided
	if opts == nil {
		opts = &ProcessingOptions{
			MaxDuration: 4 * time.Hour,
			Format:      FormatMP3,
			TargetLUFS:  -14.0, // Standard streaming target
			Quality:     3,     // Medium-high quality
		}
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "audio-*"+filepath.Ext(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy content to temp file
	if _, err := io.Copy(tempFile, content); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Initialize FFmpeg processor
	ffmpeg, err := NewFFmpegProcessor()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize FFmpeg: %w", err)
	}

	// Analyze original audio quality
	quality, err := ffmpeg.AnalyzeAudio(ctx, tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to analyze audio: %w", err)
	}

	// Generate waveform data
	waveform, err := ffmpeg.GenerateWaveform(ctx, tempFile.Name(), 200) // 200 samples for visualization
	if err != nil {
		return nil, fmt.Errorf("failed to generate waveform: %w", err)
	}

	// If only validating, return analysis results
	if opts.ValidateOnly {
		return &ProcessingResult{
			Format:      filepath.Ext(filename)[1:],
			FileSize:    size,
			ProcessedAt: time.Now(),
			Quality:     quality,
			Waveform:    waveform,
		}, nil
	}

	// Convert audio if needed
	outputPath, err := ffmpeg.ConvertAudio(ctx, tempFile.Name(), ConversionOptions{
		Format:     opts.Format,
		Normalize:  opts.Normalize,
		TargetLUFS: opts.TargetLUFS,
		Quality:    opts.Quality,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio: %w", err)
	}
	defer os.Remove(outputPath)

	// Read converted file for metadata
	convertedFile, err := os.Open(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open converted file: %w", err)
	}
	defer convertedFile.Close()

	// Extract metadata from converted file
	metadata, err := tag.ReadFrom(convertedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Analyze converted audio quality
	convertedQuality, err := ffmpeg.AnalyzeAudio(ctx, outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze converted audio: %w", err)
	}

	// Get file info
	fileInfo, err := convertedFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Send progress updates if channel is provided
	if opts.Progress != nil {
		opts.Progress <- 1.0 // 100% complete
	}

	return &ProcessingResult{
		Format:      string(opts.Format),
		Bitrate:     convertedQuality.Bitrate,
		Channels:    convertedQuality.Channels,
		Metadata:    metadata,
		FileSize:    fileInfo.Size(),
		ProcessedAt: time.Now(),
		Quality:     convertedQuality,
		Waveform:    waveform,
	}, nil
}

// ValidateAndExtractMetadata combines validation and metadata extraction
func ValidateAndExtractMetadata(content io.Reader, size int64, filename string) (tag.Metadata, error) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "audio-*"+filepath.Ext(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy content to temp file
	if _, err := io.Copy(tempFile, content); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Reopen file for reading
	file, err := os.Open(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to open temp file: %w", err)
	}
	defer file.Close()

	// Extract metadata
	return tag.ReadFrom(file)
}

// ProcessAudio processes an audio file with the given options
func (p *FFmpegProcessor) ProcessAudio(ctx context.Context, inputPath string, outputPath string, opts ProcessingOptions) error {
	args := []string{
		"-i", inputPath,
	}

	// Add audio filters based on options
	var filters []string

	// Normalize audio if requested
	if opts.Normalize {
		filters = append(filters, fmt.Sprintf("loudnorm=I=%f:TP=-1.0:LRA=11", opts.TargetLUFS))
	}

	// Add quality-specific options
	switch opts.Format {
	case FormatMP3:
		args = append(args, "-c:a", "libmp3lame")
		args = append(args, "-q:a", fmt.Sprintf("%d", opts.Quality))
	case FormatFLAC:
		args = append(args, "-c:a", "flac")
		args = append(args, "-compression_level", fmt.Sprintf("%d", opts.Quality))
	case FormatAAC:
		args = append(args, "-c:a", "aac")
		args = append(args, "-b:a", fmt.Sprintf("%dk", 96+(32*opts.Quality))) // Scale bitrate with quality
	case FormatOGG:
		args = append(args, "-c:a", "libvorbis")
		args = append(args, "-q:a", fmt.Sprintf("%d", opts.Quality))
	default:
		return fmt.Errorf("unsupported format: %s", opts.Format)
	}

	// Apply filters if any
	if len(filters) > 0 {
		args = append(args, "-af", strings.Join(filters, ","))
	}

	// Add output path
	args = append(args, outputPath)

	// Run FFmpeg command
	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FFmpeg processing failed: %w", err)
	}

	return nil
}
