package domain

import (
	"context"
	"io"
)

// StorageService defines the interface for file storage operations
type StorageService interface {
	// Upload uploads a file to storage
	Upload(ctx context.Context, path string, content io.Reader) error

	// Download downloads a file from storage
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, path string) error

	// GetURL returns a URL for accessing the file
	GetURL(ctx context.Context, path string) (string, error)
}
