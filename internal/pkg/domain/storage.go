package domain

import (
	"context"
	"io"
	"time"
)

// StorageService defines the interface for cloud storage operations
type StorageService interface {
	// Upload stores a file in cloud storage
	Upload(ctx context.Context, file *File) error

	// Download retrieves a file from cloud storage
	Download(ctx context.Context, key string) (*File, error)

	// Delete removes a file from cloud storage
	Delete(ctx context.Context, key string) error

	// GetSignedURL generates a pre-signed URL for direct upload/download
	GetSignedURL(ctx context.Context, key string, operation SignedURLOperation, expiry time.Duration) (string, error)
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
