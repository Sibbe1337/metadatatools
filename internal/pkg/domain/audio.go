package domain

import "context"

// AudioService handles audio file operations
type AudioService interface {
	// Upload stores an audio file and returns its URL
	Upload(ctx context.Context, file *File) (string, error)

	// GetURL retrieves a pre-signed URL for an audio file
	GetURL(ctx context.Context, id string) (string, error)
}
