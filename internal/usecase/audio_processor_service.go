package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AudioProcessorService implements audio processing functionality
type AudioProcessorService struct {
	storage    domain.StorageService
	tracks     domain.TrackRepository
	ai         domain.AIService
	processor  domain.AudioProcessor
	ffmpegPath string
}

// NewAudioProcessorService creates a new audio processor service
func NewAudioProcessorService(storage domain.StorageService, tracks domain.TrackRepository, ai domain.AIService, processor domain.AudioProcessor, ffmpegPath string) *AudioProcessorService {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg" // Use from PATH if not specified
	}
	return &AudioProcessorService{
		storage:    storage,
		tracks:     tracks,
		ai:         ai,
		processor:  processor,
		ffmpegPath: ffmpegPath,
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

// technicalData holds extracted technical metadata
type technicalData struct {
	duration    float64
	sampleRate  int
	sampleCount int64
	channels    int
	bitDepth    int
	bitrate     int
}

// ffprobeOutput represents the JSON output from ffprobe
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
		Size     string `json:"size"`
		Format   string `json:"format_name"`
		Tags     map[string]string
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate,omitempty"`
		Channels   int    `json:"channels,omitempty"`
		BitDepth   int    `json:"bits_per_sample,omitempty"`
		Duration   string `json:"duration"`
		Tags       map[string]string
	} `json:"streams"`
}

// extractTechnicalData extracts technical metadata using ffprobe
func (s *AudioProcessorService) extractTechnicalData(ctx context.Context, filepath string) (*technicalData, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filepath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	data := &technicalData{}

	// Parse duration
	if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		data.duration = duration
	}

	// Parse bitrate
	if bitrate, err := strconv.Atoi(probe.Format.BitRate); err == nil {
		data.bitrate = bitrate / 1000 // Convert to kbps
	}

	// Find audio stream
	for _, stream := range probe.Streams {
		if stream.CodecType == "audio" {
			// Parse sample rate
			if sampleRate, err := strconv.Atoi(stream.SampleRate); err == nil {
				data.sampleRate = sampleRate
			}
			data.channels = stream.Channels
			data.bitDepth = stream.BitDepth

			// Calculate sample count
			if data.duration > 0 && data.sampleRate > 0 {
				data.sampleCount = int64(data.duration * float64(data.sampleRate))
			}
			break
		}
	}

	return data, nil
}

// analyzeAudio performs audio analysis using ffmpeg
func (s *AudioProcessorService) analyzeAudio(ctx context.Context, filepath string, analysis *domain.AudioAnalysis) error {
	// Example analysis using ffmpeg's ebur128 filter
	cmd := exec.CommandContext(ctx, s.ffmpegPath,
		"-i", filepath,
		"-filter_complex", "ebur128=peak=true",
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg analysis failed: %w", err)
	}

	// Parse the output
	metrics := s.parseFFmpegOutput(output)

	// Update analysis with parsed values
	if loudness, ok := metrics["loudness"]; ok {
		analysis.Loudness = loudness
	}
	if energy, ok := metrics["energy"]; ok {
		analysis.Energy = energy
	}

	// Add example values for other metrics
	// In a real implementation, these would be calculated from the audio analysis
	analysis.BPM = 120.0        // Example value
	analysis.Key = "C"          // Example value
	analysis.Mode = "major"     // Example value
	analysis.Tempo = 120.0      // Example value
	analysis.Danceability = 0.8 // Example value
	analysis.Valence = 0.65     // Example value
	analysis.Complexity = 0.45  // Example value
	analysis.Intensity = 0.7    // Example value
	analysis.Mood = "energetic" // Example value

	// Add example segments
	analysis.Segments = []domain.AudioSegment{
		{
			Start:      0.0,
			Duration:   30.0,
			Loudness:   -14.0,
			Timbre:     []float64{1.0, 0.8, 0.6},
			Pitches:    []float64{0.9, 0.1, 0.3},
			Confidence: 0.95,
		},
		{
			Start:      30.0,
			Duration:   30.0,
			Loudness:   -13.5,
			Timbre:     []float64{0.9, 0.7, 0.5},
			Pitches:    []float64{0.8, 0.2, 0.4},
			Confidence: 0.92,
		},
	}

	return nil
}

// parseFFmpegOutput parses ffmpeg output to extract relevant data
func (s *AudioProcessorService) parseFFmpegOutput(output []byte) map[string]float64 {
	results := make(map[string]float64)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Integrated loudness:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if val, err := strconv.ParseFloat(parts[2], 64); err == nil {
					results["loudness"] = val
				}
			}
		} else if strings.Contains(line, "Loudness range:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if val, err := strconv.ParseFloat(parts[2], 64); err == nil {
					results["loudness_range"] = val
				}
			}
		} else if strings.Contains(line, "True peak:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if val, err := strconv.ParseFloat(parts[2], 64); err == nil {
					results["true_peak"] = val
				}
			}
		}
	}

	// Calculate energy from loudness and peak
	if loudness, ok := results["loudness"]; ok {
		if peak, ok := results["true_peak"]; ok {
			// Simple energy calculation (this is just an example)
			energy := (loudness + 23.0) / 23.0 * (peak + 3.0) / 3.0
			results["energy"] = energy
		}
	}

	return results
}
