package audio

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// AudioEffect represents a type of audio effect
type AudioEffect string

const (
	EffectCompression AudioEffect = "compression"
	EffectEQ          AudioEffect = "eq"
	EffectReverb      AudioEffect = "reverb"
	EffectDelay       AudioEffect = "delay"
	EffectLimiter     AudioEffect = "limiter"
)

// CompressorSettings defines audio compression parameters
type CompressorSettings struct {
	Threshold  float64 // dB level where compression begins
	Ratio      float64 // compression ratio (e.g., 4.0 for 4:1)
	Attack     float64 // attack time in milliseconds
	Release    float64 // release time in milliseconds
	MakeupGain float64 // makeup gain in dB
	KneeWidth  float64 // knee width in dB
}

// EQSettings defines equalizer parameters
type EQSettings struct {
	Bands []EQBand
}

// EQBand represents a single equalizer band
type EQBand struct {
	Frequency float64 // center frequency in Hz
	Gain      float64 // gain in dB
	Q         float64 // Q factor (bandwidth)
}

// ReverbSettings defines reverb parameters
type ReverbSettings struct {
	RoomSize float64 // room size (0.0-1.0)
	Damping  float64 // high frequency damping (0.0-1.0)
	WetLevel float64 // wet (processed) signal level (0.0-1.0)
	DryLevel float64 // dry (unprocessed) signal level (0.0-1.0)
	Width    float64 // stereo width (0.0-1.0)
	PreDelay float64 // pre-delay in milliseconds
}

// DelaySettings defines delay parameters
type DelaySettings struct {
	Time     float64 // delay time in milliseconds
	Feedback float64 // feedback amount (0.0-1.0)
	Mix      float64 // wet/dry mix (0.0-1.0)
}

// LimiterSettings defines limiter parameters
type LimiterSettings struct {
	Threshold float64 // limiting threshold in dB
	Release   float64 // release time in milliseconds
}

// Effect represents an audio effect
type Effect interface {
	// GetFFmpegFilter returns the FFmpeg filter string for this effect
	GetFFmpegFilter() string
}

// EffectChain represents a chain of audio effects
type EffectChain struct {
	Effects []Effect
}

// CompressorEffect represents dynamic range compression
type CompressorEffect struct {
	Threshold  float64 // dB threshold (e.g., -20)
	Ratio      float64 // compression ratio (e.g., 4)
	Attack     float64 // attack time in milliseconds
	Release    float64 // release time in milliseconds
	MakeupGain float64 // makeup gain in dB
}

// GetFFmpegFilter implements the Effect interface
func (e CompressorEffect) GetFFmpegFilter() string {
	return fmt.Sprintf("acompressor=threshold=%f:ratio=%f:attack=%f:release=%f:makeup=%f",
		e.Threshold, e.Ratio, e.Attack, e.Release, e.MakeupGain)
}

// EQEffect represents a parametric equalizer band
type EQEffect struct {
	Frequency float64 // center frequency in Hz
	Gain      float64 // gain in dB
	Q         float64 // Q factor (bandwidth)
}

// GetFFmpegFilter implements the Effect interface
func (e EQEffect) GetFFmpegFilter() string {
	return fmt.Sprintf("equalizer=f=%f:t=h:w=%f:g=%f",
		e.Frequency, e.Q, e.Gain)
}

// ReverbEffect represents reverb parameters
type ReverbEffect struct {
	RoomSize float64 // 0-1 room size
	Damping  float64 // 0-1 damping factor
	WetLevel float64 // 0-1 wet level
	DryLevel float64 // 0-1 dry level
	Width    float64 // 0-1 stereo width
	PreDelay float64 // pre-delay in ms
}

// GetFFmpegFilter implements the Effect interface
func (e ReverbEffect) GetFFmpegFilter() string {
	return fmt.Sprintf("aecho=0.8:%f:%f:0.5",
		e.PreDelay, e.RoomSize*1000)
}

// DelayEffect represents delay parameters
type DelayEffect struct {
	Time     float64 // delay time in milliseconds
	Feedback float64 // 0-1 feedback amount
	Mix      float64 // 0-1 wet/dry mix
}

// GetFFmpegFilter implements the Effect interface
func (e DelayEffect) GetFFmpegFilter() string {
	return fmt.Sprintf("adelay=%d|%d,amix=2:1",
		int(e.Time), int(e.Time))
}

// LimiterEffect represents a peak limiter
type LimiterEffect struct {
	Threshold float64 // dB threshold
	Release   float64 // release time in seconds
}

// GetFFmpegFilter implements the Effect interface
func (e LimiterEffect) GetFFmpegFilter() string {
	return fmt.Sprintf("alimiter=level_in=%f:level_out=%f:limit=%f:release=%f",
		1.0, 1.0, e.Threshold, e.Release)
}

// StereoWidthEffect represents stereo width adjustment
type StereoWidthEffect struct {
	Width float64 // 0-2 width factor (1 = normal, 0 = mono, 2 = extra wide)
}

// GetFFmpegFilter implements the Effect interface
func (e StereoWidthEffect) GetFFmpegFilter() string {
	return fmt.Sprintf("stereotools=mwidth=%f",
		e.Width)
}

// NewEffectChain creates a new effect chain
func NewEffectChain() *EffectChain {
	return &EffectChain{
		Effects: make([]Effect, 0),
	}
}

// Add adds an effect to the chain
func (c *EffectChain) Add(effect Effect) {
	c.Effects = append(c.Effects, effect)
}

// GetFFmpegFilterChain returns the complete FFmpeg filter chain
func (c *EffectChain) GetFFmpegFilterChain() string {
	var filters []string
	for _, effect := range c.Effects {
		filters = append(filters, effect.GetFFmpegFilter())
	}
	return strings.Join(filters, ",")
}

// Common presets
var (
	DefaultCompressor = CompressorEffect{
		Threshold:  -20,
		Ratio:      4,
		Attack:     20,
		Release:    100,
		MakeupGain: 0,
	}

	VocalCompressor = CompressorEffect{
		Threshold:  -24,
		Ratio:      6,
		Attack:     10,
		Release:    60,
		MakeupGain: 3,
	}

	SmallRoom = ReverbEffect{
		RoomSize: 0.2,
		Damping:  0.3,
		WetLevel: 0.3,
		DryLevel: 0.7,
		Width:    1.0,
		PreDelay: 20,
	}

	LargeHall = ReverbEffect{
		RoomSize: 0.8,
		Damping:  0.2,
		WetLevel: 0.4,
		DryLevel: 0.6,
		Width:    1.0,
		PreDelay: 40,
	}

	QuarterNote = DelayEffect{
		Time:     250, // Assuming 120 BPM
		Feedback: 0.3,
		Mix:      0.4,
	}

	MasterLimiter = LimiterEffect{
		Threshold: -1.0,
		Release:   0.1,
	}
)

// CreatePresetChain creates an effect chain with common presets
func CreatePresetChain(preset string) *EffectChain {
	chain := NewEffectChain()

	switch strings.ToLower(preset) {
	case "vocal":
		chain.Add(VocalCompressor)
		chain.Add(SmallRoom)
		chain.Add(MasterLimiter)
	case "master":
		chain.Add(DefaultCompressor)
		chain.Add(MasterLimiter)
	case "ambient":
		chain.Add(LargeHall)
		chain.Add(StereoWidthEffect{Width: 1.5})
		chain.Add(MasterLimiter)
	}

	return chain
}

// ApplyEffects applies a chain of audio effects to the input file
func (p *FFmpegProcessor) ApplyEffects(ctx context.Context, inputPath string, chain EffectChain) (string, error) {
	outputPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_processed%s",
		strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)),
		filepath.Ext(inputPath)))

	// Build FFmpeg filter chain
	var filters []string
	for _, effect := range chain.Effects {
		filter, err := buildEffectFilter(effect)
		if err != nil {
			return "", fmt.Errorf("failed to build effect filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Construct FFmpeg command
	args := []string{
		"-i", inputPath,
		"-filter:a", strings.Join(filters, ","),
		"-y", // overwrite output file
		outputPath,
	}

	// Run FFmpeg
	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to apply effects: %w", err)
	}

	return outputPath, nil
}

// buildEffectFilter constructs FFmpeg filter string for an effect
func buildEffectFilter(effect Effect) (string, error) {
	switch e := effect.(type) {
	case CompressorEffect:
		return fmt.Sprintf(
			"acompressor=threshold=%f:ratio=%f:attack=%f:release=%f:makeup=%f",
			e.Threshold,
			e.Ratio,
			e.Attack,
			e.Release,
			e.MakeupGain,
		), nil

	case EQEffect:
		return fmt.Sprintf(
			"equalizer=f=%f:width_type=q:w=%f:g=%f",
			e.Frequency,
			e.Q,
			e.Gain,
		), nil

	case ReverbEffect:
		return fmt.Sprintf(
			"aecho=0.8:%f:%f:0.5",
			e.PreDelay,
			e.RoomSize*1000,
		), nil

	case DelayEffect:
		return fmt.Sprintf(
			"adelay=%d|%d,amix=2:1",
			int(e.Time),
			int(e.Time),
		), nil

	case LimiterEffect:
		return fmt.Sprintf(
			"alimiter=level_in=%f:level_out=%f:limit=%f:release=%f",
			1.0,
			1.0,
			e.Threshold,
			e.Release,
		), nil

	case StereoWidthEffect:
		return fmt.Sprintf(
			"stereotools=mwidth=%f",
			e.Width,
		), nil

	default:
		return "", fmt.Errorf("unsupported effect type: %T", effect)
	}
}
