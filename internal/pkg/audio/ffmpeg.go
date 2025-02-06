package audio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// AudioFormat represents supported output formats
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatFLAC AudioFormat = "flac"
	FormatWAV  AudioFormat = "wav"
	FormatAAC  AudioFormat = "aac"
	FormatOGG  AudioFormat = "ogg"
)

// AudioQuality represents audio quality settings
type AudioQuality struct {
	Bitrate      int     // in kbps
	SampleRate   int     // in Hz
	Channels     int     // number of channels
	NoiseLevel   float64 // in dB
	Clipping     float64 // percentage of clipped samples
	DynamicRange float64 // in dB
}

// ConversionOptions defines options for audio conversion
type ConversionOptions struct {
	Format     AudioFormat
	Bitrate    int     // in kbps
	SampleRate int     // in Hz
	Channels   int     // number of channels
	Normalize  bool    // apply audio normalization
	TargetLUFS float64 // target loudness (default: -14 LUFS for streaming)
	Quality    int     // encoding quality (0-9, where 0 is best)
}

// FFmpegProcessor handles audio processing using FFmpeg
type FFmpegProcessor struct {
	ffmpegPath  string
	ffprobePath string
	tempDir     string
}

// FFprobeOutput represents the JSON output from FFprobe
type FFprobeOutput struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		BitRate    string `json:"bit_rate"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
		Size     string `json:"size"`
	} `json:"format"`
}

// NewFFmpegProcessor creates a new FFmpeg processor
func NewFFmpegProcessor() (*FFmpegProcessor, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found: %w", err)
	}

	return &FFmpegProcessor{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		tempDir:     os.TempDir(),
	}, nil
}

// ConvertAudio converts audio to the specified format with given options
func (p *FFmpegProcessor) ConvertAudio(ctx context.Context, inputPath string, opts ConversionOptions) (string, error) {
	outputPath := filepath.Join(p.tempDir, fmt.Sprintf("%s.%s",
		strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)),
		opts.Format))

	args := []string{
		"-i", inputPath,
		"-y", // overwrite output file
	}

	// Add conversion options
	if opts.Bitrate > 0 {
		args = append(args, "-b:a", fmt.Sprintf("%dk", opts.Bitrate))
	}
	if opts.SampleRate > 0 {
		args = append(args, "-ar", strconv.Itoa(opts.SampleRate))
	}
	if opts.Channels > 0 {
		args = append(args, "-ac", strconv.Itoa(opts.Channels))
	}
	if opts.Quality >= 0 && opts.Quality <= 9 {
		args = append(args, "-q:a", strconv.Itoa(opts.Quality))
	}

	// Apply normalization if requested
	if opts.Normalize {
		// First pass: analyze loudness
		loudnessArgs := []string{
			"-i", inputPath,
			"-af", "loudnorm=print_format=json",
			"-f", "null",
			"-",
		}
		cmd := exec.CommandContext(ctx, p.ffmpegPath, loudnessArgs...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("loudness analysis failed: %w", err)
		}

		// Parse loudness stats
		var stats struct {
			InputI      float64 `json:"input_i"`
			InputTP     float64 `json:"input_tp"`
			InputLRA    float64 `json:"input_lra"`
			InputThresh float64 `json:"input_thresh"`
		}
		if err := json.Unmarshal(stderr.Bytes(), &stats); err != nil {
			return "", fmt.Errorf("failed to parse loudness stats: %w", err)
		}

		// Add normalization filter
		args = append(args, "-af", fmt.Sprintf(
			"loudnorm=I=%f:TP=-1.5:LRA=11",
			opts.TargetLUFS,
		))
	}

	// Add output path
	args = append(args, outputPath)

	// Run conversion
	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("conversion failed: %w", err)
	}

	return outputPath, nil
}

// AnalyzeAudio performs audio quality analysis
func (p *FFmpegProcessor) AnalyzeAudio(ctx context.Context, inputPath string) (*AudioQuality, error) {
	// Get basic audio information
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}
	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("audio analysis failed: %w", err)
	}

	// Parse FFprobe output
	var probeData struct {
		Streams []struct {
			CodecType  string `json:"codec_type"`
			BitRate    string `json:"bit_rate"`
			SampleRate string `json:"sample_rate"`
			Channels   int    `json:"channels"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("failed to parse audio info: %w", err)
	}

	// Find the audio stream
	var audioStream struct {
		CodecType  string `json:"codec_type"`
		BitRate    string `json:"bit_rate"`
		SampleRate string `json:"sample_rate"`
		Channels   int    `json:"channels"`
	}

	for _, stream := range probeData.Streams {
		if stream.CodecType == "audio" {
			audioStream = stream
			break
		}
	}

	// Convert bitrate to int
	bitrate, _ := strconv.Atoi(audioStream.BitRate)
	bitrate /= 1000 // Convert to kbps

	// Convert sample rate to int
	sampleRate, _ := strconv.Atoi(audioStream.SampleRate)

	// Analyze audio quality
	quality := &AudioQuality{
		Bitrate:    bitrate,
		SampleRate: sampleRate,
		Channels:   audioStream.Channels,
	}

	// Analyze noise level and clipping
	silenceArgs := []string{
		"-i", inputPath,
		"-af", "silencedetect=noise=-50dB:d=0.1",
		"-f", "null",
		"-",
	}
	cmd = exec.CommandContext(ctx, p.ffmpegPath, silenceArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("silence detection failed: %w", err)
	}

	// Parse silence detection output
	output = stderr.Bytes()
	if idx := bytes.Index(output, []byte("mean_volume:")); idx != -1 {
		fmt.Sscanf(string(output[idx:]), "mean_volume: %f dB", &quality.NoiseLevel)
	}

	// Analyze clipping
	volumeArgs := []string{
		"-i", inputPath,
		"-af", "volumedetect",
		"-f", "null",
		"-",
	}
	cmd = exec.CommandContext(ctx, p.ffmpegPath, volumeArgs...)
	stderr.Reset()
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("volume detection failed: %w", err)
	}

	// Parse volume detection output
	output = stderr.Bytes()
	var maxVolume float64
	if idx := bytes.Index(output, []byte("max_volume:")); idx != -1 {
		fmt.Sscanf(string(output[idx:]), "max_volume: %f dB", &maxVolume)
	}
	quality.Clipping = maxVolume / 0.0 // Calculate percentage of clipping

	// Calculate dynamic range
	quality.DynamicRange = maxVolume - quality.NoiseLevel

	return quality, nil
}

// GenerateWaveform generates a waveform data for visualization
func (p *FFmpegProcessor) GenerateWaveform(ctx context.Context, inputPath string, samples int) ([]float64, error) {
	// Use ffmpeg to generate waveform data
	args := []string{
		"-i", inputPath,
		"-filter_complex", fmt.Sprintf("aformat=channel_layouts=mono,compand,showwavespic=s=%d:1", samples),
		"-f", "null",
		"-",
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("waveform generation failed: %w", err)
	}

	// Parse waveform data
	waveform := make([]float64, samples)
	lines := strings.Split(stderr.String(), "\n")
	for i, line := range lines {
		if i >= samples {
			break
		}
		if value, err := strconv.ParseFloat(strings.TrimSpace(line), 64); err == nil {
			waveform[i] = value
		}
	}

	return waveform, nil
}
