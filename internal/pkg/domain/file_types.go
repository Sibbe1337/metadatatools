package domain

import (
	"context"
	"io"
	"time"
)

// ProcessingAudioFile represents an audio file being processed
type ProcessingAudioFile struct {
	Path     string      // Path to the file
	Name     string      // Original filename
	Size     int64       // File size in bytes
	Format   AudioFormat // Audio format
	Reader   io.Reader   // File reader (optional)
	Content  io.Reader   // File content for processing
	Metadata *AudioMetadata
}

// StorageFile represents a file to be stored
type StorageFile struct {
	Key         string    // Unique identifier/path in storage
	Name        string    // Original filename
	Size        int64     // File size in bytes
	ContentType string    // MIME type
	Content     io.Reader // File content
	Metadata    map[string]string
	UploadedAt  time.Time
}

// FileMetadata represents metadata for a stored file
type FileMetadata struct {
	Key          string
	Name         string
	Size         int64
	ContentType  string
	UploadedAt   time.Time
	LastModified time.Time
	ETag         string
	StorageClass string
	Metadata     map[string]string
}

// StorageService defines the interface for storage operations
type StorageService interface {
	// Core file operations
	Upload(ctx context.Context, file *StorageFile) error
	Download(ctx context.Context, key string) (*StorageFile, error)
	Delete(ctx context.Context, key string) error
	GetURL(ctx context.Context, key string) (string, error)
	GetMetadata(ctx context.Context, key string) (*FileMetadata, error)
	ListFiles(ctx context.Context, prefix string) ([]*FileMetadata, error)

	// Audio-specific operations
	UploadAudio(ctx context.Context, file io.Reader, path string) error
	DeleteAudio(ctx context.Context, path string) error
	GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// Quota and validation
	GetQuotaUsage(ctx context.Context) (int64, error)
	ValidateUpload(ctx context.Context, fileSize int64, mimeType string) error
}

// StorageClient defines the interface for low-level storage operations
type StorageClient interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, key string, content io.Reader, options map[string]string) error

	// Download downloads a file from storage
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, key string) error

	// GetURL gets a pre-signed URL for a file
	GetURL(ctx context.Context, key string, operation SignedURLOperation) (string, error)

	// GetMetadata gets metadata for a file
	GetMetadata(ctx context.Context, key string) (*FileMetadata, error)

	// List lists files with a prefix
	List(ctx context.Context, prefix string) ([]*FileMetadata, error)
}
