package domain

import (
	"context"
	"time"
)

// Track represents a music track
type Track struct {
	// Core fields
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`

	// Storage details
	StoragePath string `json:"storagePath"`
	FilePath    string `json:"filePath"` // Deprecated: use StoragePath
	FileSize    int64  `json:"fileSize"`
	AudioData   []byte `json:"-"` // In-memory audio data for processing

	// Track metadata
	Metadata CompleteTrackMetadata `json:"metadata"`
}

// Helper methods to access metadata fields
func (t *Track) Title() string       { return t.Metadata.Title }
func (t *Track) Artist() string      { return t.Metadata.Artist }
func (t *Track) Album() string       { return t.Metadata.Album }
func (t *Track) Year() int           { return t.Metadata.Year }
func (t *Track) Duration() float64   { return t.Metadata.Duration }
func (t *Track) ISRC() string        { return t.Metadata.ISRC }
func (t *Track) ISWC() string        { return t.Metadata.Additional.CustomFields["iswc"] }
func (t *Track) Label() string       { return t.Metadata.Additional.CustomFields["label"] }
func (t *Track) Territory() string   { return t.Metadata.Additional.CustomFields["territory"] }
func (t *Track) Genre() string       { return t.Metadata.Musical.Genre }
func (t *Track) BPM() float64        { return t.Metadata.Musical.BPM }
func (t *Track) Key() string         { return t.Metadata.Musical.Key }
func (t *Track) Mood() string        { return t.Metadata.Musical.Mood }
func (t *Track) AudioFormat() string { return string(t.Metadata.Technical.Format) }
func (t *Track) SampleRate() int     { return t.Metadata.Technical.SampleRate }
func (t *Track) Bitrate() int        { return t.Metadata.Technical.Bitrate }
func (t *Track) Channels() int       { return t.Metadata.Technical.Channels }
func (t *Track) Publisher() string   { return t.Metadata.Additional.Publisher }
func (t *Track) Copyright() string   { return t.Metadata.Additional.Copyright }
func (t *Track) Lyrics() string      { return t.Metadata.Additional.Lyrics }

// AI-related fields
func (t *Track) AITags() []string      { return t.Metadata.AI.Tags }
func (t *Track) AIConfidence() float64 { return t.Metadata.AI.Confidence }
func (t *Track) ModelVersion() string  { return t.Metadata.AI.Version }
func (t *Track) NeedsReview() bool     { return t.Metadata.AI.NeedsReview }

// Helper methods to set metadata fields
func (t *Track) SetTitle(v string)       { t.Metadata.Title = v }
func (t *Track) SetArtist(v string)      { t.Metadata.Artist = v }
func (t *Track) SetAlbum(v string)       { t.Metadata.Album = v }
func (t *Track) SetYear(v int)           { t.Metadata.Year = v }
func (t *Track) SetDuration(v float64)   { t.Metadata.Duration = v }
func (t *Track) SetISRC(v string)        { t.Metadata.ISRC = v }
func (t *Track) SetISWC(v string)        { t.Metadata.Additional.CustomFields["iswc"] = v }
func (t *Track) SetLabel(v string)       { t.Metadata.Additional.CustomFields["label"] = v }
func (t *Track) SetTerritory(v string)   { t.Metadata.Additional.CustomFields["territory"] = v }
func (t *Track) SetGenre(v string)       { t.Metadata.Musical.Genre = v }
func (t *Track) SetBPM(v float64)        { t.Metadata.Musical.BPM = v }
func (t *Track) SetKey(v string)         { t.Metadata.Musical.Key = v }
func (t *Track) SetMood(v string)        { t.Metadata.Musical.Mood = v }
func (t *Track) SetAudioFormat(v string) { t.Metadata.Technical.Format = AudioFormat(v) }
func (t *Track) SetSampleRate(v int)     { t.Metadata.Technical.SampleRate = v }
func (t *Track) SetBitrate(v int)        { t.Metadata.Technical.Bitrate = v }
func (t *Track) SetChannels(v int)       { t.Metadata.Technical.Channels = v }
func (t *Track) SetPublisher(v string)   { t.Metadata.Additional.Publisher = v }
func (t *Track) SetCopyright(v string)   { t.Metadata.Additional.Copyright = v }
func (t *Track) SetLyrics(v string)      { t.Metadata.Additional.Lyrics = v }

// TrackRepository defines the interface for track data operations
type TrackRepository interface {
	// Create creates a new track
	Create(ctx context.Context, track *Track) error

	// GetByID retrieves a track by ID
	GetByID(ctx context.Context, id string) (*Track, error)

	// Update updates an existing track
	Update(ctx context.Context, track *Track) error

	// Delete soft-deletes a track
	Delete(ctx context.Context, id string) error

	// List retrieves tracks based on filters with pagination
	List(ctx context.Context, filters map[string]interface{}, offset, limit int) ([]*Track, error)

	// SearchByMetadata searches tracks by metadata fields
	SearchByMetadata(ctx context.Context, query map[string]interface{}) ([]*Track, error)

	// GetByISRC retrieves a track by ISRC
	GetByISRC(ctx context.Context, isrc string) (*Track, error)

	// BatchUpdate updates multiple tracks in a single transaction
	BatchUpdate(ctx context.Context, tracks []*Track) error
}
