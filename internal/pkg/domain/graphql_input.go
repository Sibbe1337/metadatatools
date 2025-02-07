package domain

import "io"

// CreateTrackInput represents input for creating a new track
type CreateTrackInput struct {
	Title     string
	Artist    string
	Album     *string
	Genre     *string
	Year      *int
	Label     *string
	Territory *string
	ISRC      *string
	ISWC      *string
	AudioFile *AudioFile
}

// UpdateTrackInput represents input for updating a track
type UpdateTrackInput struct {
	ID        string
	Title     *string
	Artist    *string
	Album     *string
	Genre     *string
	Year      *int
	Label     *string
	Territory *string
	ISRC      *string
	ISWC      *string
	Metadata  *MetadataInput
}

// MetadataInput represents input for updating track metadata
type MetadataInput struct {
	ISRC         *string
	ISWC         *string
	BPM          *float64
	Key          *string
	Mood         *string
	Labels       []string
	CustomFields map[string]string
}

// AudioFile represents an uploaded audio file
type AudioFile struct {
	File     io.Reader
	Filename string
	Size     int64
}
