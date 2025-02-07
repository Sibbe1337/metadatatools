package domain

import "io"

// StorageFile represents a generic file in the system
type StorageFile struct {
	Key         string            // Unique identifier/path in storage
	Name        string            // Original filename
	Size        int64             // File size in bytes
	ContentType string            // MIME type
	Content     io.Reader         // File content (for upload)
	Metadata    map[string]string // File metadata
}

// ProcessingAudioFile represents an audio file being processed
type ProcessingAudioFile struct {
	Name     string
	Path     string
	Size     int64
	Format   AudioFormat
	Content  io.Reader
	Metadata *CompleteTrackMetadata
}

// UploadedFile represents a file that was uploaded through HTTP
type UploadedFile struct {
	File     io.Reader
	Filename string
	Size     int64
}
