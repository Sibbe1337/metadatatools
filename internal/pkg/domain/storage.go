package domain

import (
	"context"
	"fmt"
	"io"
	"time"
)

// StorageConfig holds cloud storage configuration
type StorageConfig struct {
	Provider         string
	Region           string
	Bucket           string
	AccessKey        string
	SecretKey        string
	Endpoint         string
	UseSSL           bool
	UploadPartSize   int64
	MaxUploadRetries int

	// File restrictions
	MaxFileSize      int64
	AllowedFileTypes []string

	// Quota settings
	UserQuota       int64 // per user storage quota in bytes
	TotalQuota      int64 // total storage quota in bytes
	QuotaWarningPct int   // percentage at which to warn about quota

	// Cleanup settings
	TempFileExpiry  time.Duration // how long to keep temp files
	CleanupInterval time.Duration // how often to run cleanup

	// Performance settings
	UploadBufferSize int64 // size of upload buffer
	DownloadTimeout  time.Duration
	UploadTimeout    time.Duration
}

// FileMetadata represents metadata about a stored file
type FileMetadata struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
	StorageClass string
	UserID       string
	Checksum     string
	IsTemporary  bool
	ExpiresAt    *time.Time
}

// StorageError represents storage-specific errors
type StorageError struct {
	Code    string
	Message string
	Op      string
	Err     error
}

func (e *StorageError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s) - %v", e.Op, e.Message, e.Code, e.Err)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Op, e.Message, e.Code)
}

// StorageService defines the interface for storage operations
type StorageService interface {
	// Upload stores a file in cloud storage
	Upload(ctx context.Context, file *StorageFile) error

	// Download retrieves a file from cloud storage
	Download(ctx context.Context, key string) (*StorageFile, error)

	// Delete removes a file from cloud storage
	Delete(ctx context.Context, key string) error

	// GetURL retrieves a pre-signed URL for an audio file
	GetURL(ctx context.Context, key string) (string, error)

	// GetMetadata retrieves metadata for a file
	GetMetadata(ctx context.Context, key string) (*FileMetadata, error)

	// ListFiles lists files with the given prefix
	ListFiles(ctx context.Context, prefix string) ([]*FileMetadata, error)

	// GetQuotaUsage gets storage usage for a user
	GetQuotaUsage(ctx context.Context, userID string) (int64, error)

	// ValidateUpload validates if a file can be uploaded
	ValidateUpload(ctx context.Context, filename string, size int64, userID string) error

	// CleanupTempFiles cleans up temporary files
	CleanupTempFiles(ctx context.Context) error
}

// File represents a file in storage
type File struct {
	Key         string    // Unique identifier/path in storage
	Name        string    // Original filename
	Size        int64     // File size in bytes
	ContentType string    // MIME type
	Content     io.Reader // File content (for upload)
	Metadata    map[string]string
	UploadedAt  time.Time
}

// SignedURLOperation represents the type of operation for a pre-signed URL
type SignedURLOperation string

const (
	SignedURLUpload   SignedURLOperation = "upload"
	SignedURLDownload SignedURLOperation = "download"
)

// Common storage paths/prefixes
const (
	StoragePathAudio = "audio/"
	StoragePathTemp  = "temp/"
)

// CacheService defines the interface for caching operations
type CacheService interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, expiration time.Duration) error
	Delete(key string) error
	PreWarm(keys []string) error
	RefreshProbabilistic(key string, threshold time.Duration, probability float64) error
}

// CacheConfig holds configuration for the cache service
type CacheConfig struct {
	PreWarmKeys     []string      // Keys to pre-warm on startup
	RefreshInterval time.Duration // How often to check for refresh
	RefreshProb     float64       // Probability of refresh when TTL is low
	TTLThreshold    time.Duration // When to start considering refresh
	DefaultTTL      time.Duration // Default TTL for cached items
}

// CacheKey types for different entities
const (
	CacheKeyTrack     = "track:%s"      // track:${id}
	CacheKeyArtist    = "artist:%s"     // artist:${name}
	CacheKeyLabel     = "label:%s"      // label:${name}
	CacheKeyTopTracks = "tracks:top:%d" // tracks:top:${limit}
)
