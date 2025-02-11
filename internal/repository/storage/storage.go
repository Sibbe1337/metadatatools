package storage

import (
	"context"
	"io"
)

// Storage defines the interface for storage operations
type Storage interface {
	Upload(ctx context.Context, key string, data io.Reader) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	GetURL(key string) string
}
