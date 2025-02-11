package domain

import "time"

// MetadataProvider defines the interface for any type that can provide metadata
type MetadataProvider interface {
	GetBasicMetadata() BasicTrackMetadata
}

// Metadata represents the GraphQL metadata type
type Metadata struct {
	ISRC         string            `json:"isrc"`
	ISWC         string            `json:"iswc"`
	BPM          float64           `json:"bpm"`
	Key          string            `json:"key"`
	Mood         string            `json:"mood"`
	Labels       []string          `json:"labels"`
	AITags       []string          `json:"aiTags"`
	Confidence   float64           `json:"confidence"`
	ModelVersion string            `json:"modelVersion"`
	CustomFields map[string]string `json:"customFields"`
}

// BasicTrackMetadata contains the fundamental metadata fields shared across all types
type BasicTrackMetadata struct {
	Title     string    `json:"title"`
	Artist    string    `json:"artist"`
	Album     string    `json:"album"`
	Year      int       `json:"year"`
	Duration  float64   `json:"duration"`
	ISRC      string    `json:"isrc"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CompleteTrackMetadata represents the complete metadata for a track
type CompleteTrackMetadata struct {
	BasicTrackMetadata `json:"basic"`
	Technical          AudioTechnicalMetadata `json:"technical"`
	Musical            MusicalMetadata        `json:"musical"`
	AI                 *TrackAIMetadata       `json:"ai,omitempty"`
	Additional         AdditionalMetadata     `json:"additional"`
}

// AudioTechnicalMetadata contains audio file technical details
type AudioTechnicalMetadata struct {
	Format     AudioFormat `json:"format"`
	SampleRate int         `json:"sampleRate"` // Hz
	Bitrate    int         `json:"bitrate"`    // kbps
	Channels   int         `json:"channels"`
	FileSize   int64       `json:"fileSize"` // bytes
}

// MusicalMetadata contains musical attributes
type MusicalMetadata struct {
	BPM    float64 `json:"bpm"`
	Key    string  `json:"key"`
	Mode   string  `json:"mode"`
	Mood   string  `json:"mood"`
	Genre  string  `json:"genre"`
	Energy float64 `json:"energy"`
	Tempo  float64 `json:"tempo"`
}

// TrackAIMetadata contains AI-generated metadata and processing information
type TrackAIMetadata struct {
	Tags                  []string               `json:"tags"`
	Confidence            float64                `json:"confidence"`
	Model                 string                 `json:"model"`
	Version               string                 `json:"version"`
	ProcessedAt           time.Time              `json:"processedAt"`
	NeedsReview           bool                   `json:"needsReview"`
	ReviewReason          string                 `json:"reviewReason,omitempty"`
	Analysis              string                 `json:"analysis,omitempty"`
	ValidationIssues      []ValidationIssue      `json:"validationIssues,omitempty"`
	ValidationSuggestions []ValidationSuggestion `json:"validationSuggestions,omitempty"`
}

// AdditionalMetadata contains supplementary metadata fields
type AdditionalMetadata struct {
	Publisher    string            `json:"publisher"`
	Copyright    string            `json:"copyright"`
	Lyrics       string            `json:"lyrics"`
	CustomTags   map[string]string `json:"customTags"`
	CustomFields map[string]string `json:"customFields"`
}

// GetBasicMetadata implements MetadataProvider interface
func (t CompleteTrackMetadata) GetBasicMetadata() BasicTrackMetadata {
	return t.BasicTrackMetadata
}
