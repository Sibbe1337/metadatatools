package domain

import (
	"context"
	"io"
	"time"
)

// AudioFormat represents supported audio formats
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatWAV  AudioFormat = "wav"
	FormatFLAC AudioFormat = "flac"
	FormatM4A  AudioFormat = "m4a"
	FormatAAC  AudioFormat = "aac"
)

// AudioMetadata represents extracted audio metadata
type AudioMetadata struct {
	// Basic metadata
	Title       string
	Artist      string
	Album       string
	Year        int
	Genre       string
	TrackNumber int
	Duration    float64

	// Technical details
	Format       AudioFormat
	Bitrate      int
	SampleRate   int
	Channels     int
	BitDepth     int
	FileSize     int64
	IsLossless   bool
	IsVariable   bool
	EncodingTool string

	// Additional metadata
	Composer   string
	Publisher  string
	ISRC       string
	Copyright  string
	Lyrics     string
	CoverArt   []byte
	Comments   string
	Rating     int
	BPM        float64
	ReplayGain float64
	ModifiedAt time.Time
	EncodedAt  time.Time
	CustomTags map[string]string
}

// AudioProcessor handles audio file processing
type AudioProcessor interface {
	// Process processes an audio file and returns its metadata
	Process(ctx context.Context, file io.Reader, filename string) (*AudioMetadata, error)

	// Convert converts an audio file to a different format
	Convert(ctx context.Context, file io.Reader, from, to AudioFormat) (io.Reader, error)

	// ExtractMetadata extracts metadata from an audio file
	ExtractMetadata(ctx context.Context, file io.Reader, format AudioFormat) (*AudioMetadata, error)

	// ValidateFormat validates if the file is a valid audio file
	ValidateFormat(ctx context.Context, file io.Reader, format AudioFormat) error

	// AnalyzeAudio performs audio analysis (BPM, key detection, etc.)
	AnalyzeAudio(ctx context.Context, file io.Reader, format AudioFormat) (*AudioAnalysis, error)
}

// AudioAnalysis represents the results of audio analysis
type AudioAnalysis struct {
	// Temporal features
	BPM           float64
	TimeSignature string
	Key           string
	Mode          string
	Tempo         float64
	BeatsPerBar   int

	// Spectral features
	Loudness      float64
	Energy        float64
	Brightness    float64
	Timbre        float64
	SpectralFlux  float64
	SpectralRoll  float64
	SpectralSlope float64

	// Perceptual features
	Danceability float64
	Valence      float64
	Arousal      float64
	Complexity   float64
	Intensity    float64
	Mood         string

	// Segments and structure
	Segments     []AudioSegment
	Transitions  []float64
	SectionCount int

	// Analysis metadata
	AnalyzedAt   time.Time
	Duration     float64
	SampleCount  int64
	WindowSize   int
	HopSize      int
	SampleRate   int
	AnalyzerInfo string
}

// AudioSegment represents a segment in the audio file
type AudioSegment struct {
	Start      float64
	Duration   float64
	Loudness   float64
	Timbre     []float64
	Pitches    []float64
	Confidence float64
}

// AudioService handles audio file operations
type AudioService interface {
	// Upload stores an audio file and returns its URL
	Upload(ctx context.Context, file *File) (string, error)

	// GetURL retrieves a pre-signed URL for an audio file
	GetURL(ctx context.Context, id string) (string, error)
}
