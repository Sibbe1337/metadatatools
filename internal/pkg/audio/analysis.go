package audio

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MusicalKey represents a musical key
type MusicalKey struct {
	Root       string // e.g., "C", "F#"
	Mode       string // "major" or "minor"
	Camelot    string // Camelot notation (e.g., "8A")
	OpenKey    string // Open Key notation (e.g., "8d")
	Confidence float64
}

// AudioAnalysis contains detailed audio analysis results
type AudioAnalysis struct {
	BPM           float64    // Beats per minute
	BPMConfidence float64    // Confidence level of BPM detection
	Key           MusicalKey // Detected musical key
	Beats         []float64  // Beat positions in seconds
	Segments      []Segment  // Audio segments analysis
	Energy        float64    // Overall energy level
	Danceability  float64    // Danceability score
}

// Segment represents an analyzed segment of audio
type Segment struct {
	Start      float64   // Start time in seconds
	Duration   float64   // Duration in seconds
	Loudness   float64   // Segment loudness in dB
	Pitches    []float64 // Pitch values for 12 semitones
	Timbre     []float64 // Timbre coefficients
	Confidence float64   // Analysis confidence
}

// AudioAnalyzer handles advanced audio analysis
type AudioAnalyzer struct {
	ffmpeg   *FFmpegProcessor
	essentia string // Path to Essentia extractors
	aubio    string // Path to Aubio tools
}

// NewAudioAnalyzer creates a new audio analyzer
func NewAudioAnalyzer(ffmpeg *FFmpegProcessor) (*AudioAnalyzer, error) {
	// Check for Essentia tools
	essentia, err := exec.LookPath("essentia_streaming_extractor_music")
	if err != nil {
		essentia = "" // Optional: will fall back to alternative methods
	}

	// Check for Aubio tools
	aubio, err := exec.LookPath("aubio")
	if err != nil {
		aubio = "" // Optional: will fall back to alternative methods
	}

	return &AudioAnalyzer{
		ffmpeg:   ffmpeg,
		essentia: essentia,
		aubio:    aubio,
	}, nil
}

// AnalyzeTrack performs comprehensive audio analysis
func (a *AudioAnalyzer) AnalyzeTrack(ctx context.Context, inputPath string) (*AudioAnalysis, error) {
	analysis := &AudioAnalysis{}

	// Detect BPM using multiple methods for accuracy
	bpm, confidence, err := a.detectBPM(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("BPM detection failed: %w", err)
	}
	analysis.BPM = bpm
	analysis.BPMConfidence = confidence

	// Detect musical key
	key, err := a.detectKey(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("key detection failed: %w", err)
	}
	analysis.Key = key

	// Detect beats
	beats, err := a.detectBeats(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("beat detection failed: %w", err)
	}
	analysis.Beats = beats

	// Analyze segments if Essentia is available
	if a.essentia != "" {
		segments, err := a.analyzeSegments(ctx, inputPath)
		if err != nil {
			return nil, fmt.Errorf("segment analysis failed: %w", err)
		}
		analysis.Segments = segments
	}

	// Calculate energy and danceability
	energy, danceability, err := a.calculateFeatures(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("feature calculation failed: %w", err)
	}
	analysis.Energy = energy
	analysis.Danceability = danceability

	return analysis, nil
}

// detectBPM detects beats per minute using multiple methods
func (a *AudioAnalyzer) detectBPM(ctx context.Context, inputPath string) (float64, float64, error) {
	var bpms []float64

	// Method 1: FFmpeg with ebur128 filter
	args := []string{
		"-i", inputPath,
		"-filter:a", "ebur128=framelog=verbose",
		"-f", "null",
		"-",
	}
	cmd := exec.CommandContext(ctx, a.ffmpeg.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		// Parse BPM from output
		if bpm := parseBPMFromFFmpeg(string(output)); bpm > 0 {
			bpms = append(bpms, bpm)
		}
	}

	// Method 2: Aubio if available
	if a.aubio != "" {
		args = []string{
			"tempo",
			inputPath,
		}
		cmd = exec.CommandContext(ctx, a.aubio, args...)
		output, err = cmd.Output()
		if err == nil {
			if bpm, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64); err == nil {
				bpms = append(bpms, bpm)
			}
		}
	}

	// Calculate average BPM and confidence
	if len(bpms) == 0 {
		return 0, 0, fmt.Errorf("no valid BPM detection results")
	}

	var sum float64
	for _, bpm := range bpms {
		sum += bpm
	}
	avgBPM := sum / float64(len(bpms))

	// Calculate confidence based on consistency between methods
	var confidence float64
	if len(bpms) > 1 {
		var variance float64
		for _, bpm := range bpms {
			diff := bpm - avgBPM
			variance += diff * diff
		}
		variance /= float64(len(bpms))
		confidence = 1.0 / (1.0 + variance/10.0) // Normalize confidence
	} else {
		confidence = 0.7 // Single method confidence
	}

	return avgBPM, confidence, nil
}

// detectKey detects the musical key
func (a *AudioAnalyzer) detectKey(ctx context.Context, inputPath string) (MusicalKey, error) {
	// Use Essentia if available for accurate key detection
	if a.essentia != "" {
		return a.detectKeyWithEssentia(ctx, inputPath)
	}

	// Fallback to FFmpeg chromagram analysis
	args := []string{
		"-i", inputPath,
		"-filter:a", "achromagram=s=4096:overlap=0.75",
		"-f", "null",
		"-",
	}
	cmd := exec.CommandContext(ctx, a.ffmpeg.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return MusicalKey{}, fmt.Errorf("key detection failed: %w", err)
	}

	// Parse key from chromagram
	key := parseKeyFromChromagram(string(output))
	return key, nil
}

// detectBeats detects beat positions
func (a *AudioAnalyzer) detectBeats(ctx context.Context, inputPath string) ([]float64, error) {
	args := []string{
		"-i", inputPath,
		"-filter:a", "abeat",
		"-f", "null",
		"-",
	}
	cmd := exec.CommandContext(ctx, a.ffmpeg.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("beat detection failed: %w", err)
	}

	return parseBeatsFromOutput(string(output)), nil
}

// analyzeSegments performs detailed segment analysis
func (a *AudioAnalyzer) analyzeSegments(ctx context.Context, inputPath string) ([]Segment, error) {
	if a.essentia == "" {
		return nil, fmt.Errorf("Essentia not available for segment analysis")
	}

	args := []string{
		inputPath,
		inputPath + ".json",
	}
	cmd := exec.CommandContext(ctx, a.essentia, args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("segment analysis failed: %w", err)
	}

	// Parse segments from Essentia output
	segments, err := parseEssentiaSegments(inputPath + ".json")
	if err != nil {
		return nil, fmt.Errorf("failed to parse segments: %w", err)
	}

	return segments, nil
}

// calculateFeatures calculates energy and danceability
func (a *AudioAnalyzer) calculateFeatures(ctx context.Context, inputPath string) (energy float64, danceability float64, err error) {
	// Calculate energy from RMS levels
	args := []string{
		"-i", inputPath,
		"-filter:a", "volumedetect,astats=measure_perchannel=0:measure_overall=1",
		"-f", "null",
		"-",
	}
	cmd := exec.CommandContext(ctx, a.ffmpeg.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("feature calculation failed: %w", err)
	}

	// Parse features from output
	energy = parseEnergyFromStats(string(output))

	// Get BPM for danceability calculation
	bpm, _, err := a.detectBPM(ctx, inputPath)
	if err != nil {
		return energy, 0, fmt.Errorf("failed to get BPM for danceability: %w", err)
	}
	danceability = calculateDanceability(energy, bpm)

	return energy, danceability, nil
}

// Helper functions for parsing outputs
func parseBPMFromFFmpeg(output string) float64 {
	// Implementation for parsing BPM from FFmpeg output
	return 0
}

func parseKeyFromChromagram(output string) MusicalKey {
	// Implementation for parsing key from chromagram
	return MusicalKey{}
}

func parseBeatsFromOutput(output string) []float64 {
	// Implementation for parsing beat positions
	return nil
}

func parseEssentiaSegments(jsonPath string) ([]Segment, error) {
	// Implementation for parsing Essentia JSON output
	return nil, nil
}

func parseEnergyFromStats(output string) float64 {
	// Implementation for parsing energy from stats
	return 0
}

func calculateDanceability(energy float64, bpm float64) float64 {
	// Implementation for calculating danceability score
	return 0
}

// detectKeyWithEssentia detects musical key using Essentia
func (a *AudioAnalyzer) detectKeyWithEssentia(ctx context.Context, inputPath string) (MusicalKey, error) {
	args := []string{
		inputPath,
		inputPath + ".json",
		"--music",
	}
	cmd := exec.CommandContext(ctx, a.essentia, args...)
	if err := cmd.Run(); err != nil {
		return MusicalKey{}, fmt.Errorf("Essentia key detection failed: %w", err)
	}

	// Parse key from Essentia output
	key, err := parseEssentiaKey(inputPath + ".json")
	if err != nil {
		return MusicalKey{}, fmt.Errorf("failed to parse key: %w", err)
	}

	return key, nil
}

func parseEssentiaKey(jsonPath string) (MusicalKey, error) {
	// Implementation for parsing key from Essentia JSON output
	return MusicalKey{}, nil
}
