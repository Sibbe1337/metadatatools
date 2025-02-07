package domain

import (
	"context"
	"time"
)

// AudioFormat represents the format of an audio file
type AudioFormat string

const (
	AudioFormatMP3  AudioFormat = "mp3"
	AudioFormatWAV  AudioFormat = "wav"
	AudioFormatFLAC AudioFormat = "flac"
	AudioFormatM4A  AudioFormat = "m4a"
	AudioFormatAAC  AudioFormat = "aac"
	AudioFormatOGG  AudioFormat = "ogg"
)

// IsValid checks if the audio format is supported
func (f AudioFormat) IsValid() bool {
	switch f {
	case AudioFormatMP3, AudioFormatWAV, AudioFormatFLAC, AudioFormatM4A, AudioFormatAAC, AudioFormatOGG:
		return true
	default:
		return false
	}
}

// String returns the string representation of the audio format
func (f AudioFormat) String() string {
	return string(f)
}

// AudioMetadata represents metadata extracted from an audio file
type AudioMetadata struct {
	// Basic metadata
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Year     int    `json:"year"`
	Duration int    `json:"duration"` // Duration in seconds
	ISRC     string `json:"isrc"`

	// Technical details
	Format     AudioFormat `json:"format"`
	Bitrate    int         `json:"bitrate"`    // Bitrate in kbps
	SampleRate int         `json:"sampleRate"` // Sample rate in Hz
	Channels   int         `json:"channels"`   // Number of audio channels

	// Musical attributes
	BPM float64 `json:"bpm"` // Beats per minute
	Key string  `json:"key"` // Musical key

	// Additional metadata
	Genre        string
	TrackNumber  int
	BitDepth     int
	FileSize     int64
	IsLossless   bool
	IsVariable   bool
	EncodingTool string
	Composer     string
	Publisher    string
	Copyright    string
	Lyrics       string
	CoverArt     []byte
	Comments     string
	Rating       int
	ReplayGain   float64
	ModifiedAt   time.Time
	EncodedAt    time.Time
	CustomTags   map[string]string
}

// AudioProcessOptions contains options for audio processing
type AudioProcessOptions struct {
	FilePath        string      // Path to the audio file
	Format          AudioFormat // Audio format
	AnalyzeAudio    bool        // Whether to perform audio analysis
	ExtractMetadata bool        // Whether to extract metadata
}

// AudioProcessor defines the interface for audio processing
type AudioProcessor interface {
	// Process processes an audio file with the given options
	Process(ctx context.Context, file *ProcessingAudioFile, options *AudioProcessOptions) (*AudioProcessResult, error)
}

// AudioProcessResult contains the results of audio processing
type AudioProcessResult struct {
	Metadata     *CompleteTrackMetadata // Complete track metadata
	AnalyzerInfo string                 // Information about the analyzer used
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
	Upload(ctx context.Context, file *StorageFile) (string, error)

	// GetURL retrieves a pre-signed URL for an audio file
	GetURL(ctx context.Context, id string) (string, error)
}

// StorageClient defines the interface for storage operations
type StorageClient interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, file *StorageFile) error

	// Download downloads a file from storage
	Download(ctx context.Context, path string) (*StorageFile, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, path string) error
}
