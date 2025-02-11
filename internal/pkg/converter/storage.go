package converter

import (
	"context"
	"io"
	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"time"
)

// StorageServiceWrapper wraps a storage service to implement both internal and pkg domain interfaces
type StorageServiceWrapper struct {
	internal domain.StorageService
}

// NewStorageServiceWrapper creates a new StorageServiceWrapper
func NewStorageServiceWrapper(internal domain.StorageService) *StorageServiceWrapper {
	return &StorageServiceWrapper{
		internal: internal,
	}
}

// Internal returns the internal domain implementation
func (w *StorageServiceWrapper) Internal() domain.StorageService {
	return w.internal
}

// Pkg returns the pkg domain implementation
func (w *StorageServiceWrapper) Pkg() pkgdomain.StorageService {
	return w
}

// Upload implements the Upload method for both interfaces
func (w *StorageServiceWrapper) Upload(ctx context.Context, file *pkgdomain.StorageFile) error {
	return w.internal.Upload(ctx, file.Key, file.Content)
}

// Download implements the Download method for both interfaces
func (w *StorageServiceWrapper) Download(ctx context.Context, key string) (*pkgdomain.StorageFile, error) {
	content, err := w.internal.Download(ctx, key)
	if err != nil {
		return nil, err
	}
	metadata, err := w.GetMetadata(ctx, key)
	if err != nil {
		return nil, err
	}
	return &pkgdomain.StorageFile{
		Key:         key,
		Content:     content,
		Size:        metadata.Size,
		ContentType: metadata.ContentType,
		UploadedAt:  time.Now(),
		Metadata:    metadata.Metadata,
	}, nil
}

// Delete implements the Delete method for both interfaces
func (w *StorageServiceWrapper) Delete(ctx context.Context, key string) error {
	return w.internal.Delete(ctx, key)
}

// GetURL implements the GetURL method for both interfaces
func (w *StorageServiceWrapper) GetURL(ctx context.Context, key string) (string, error) {
	return w.internal.GetURL(ctx, key)
}

// GetSignedURL returns a signed URL for accessing the file
func (w *StorageServiceWrapper) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// Since internal interface doesn't support signed URLs, return regular URL
	return w.internal.GetURL(ctx, path)
}

// DeleteAudio implements the DeleteAudio method for the pkg domain interface
func (w *StorageServiceWrapper) DeleteAudio(ctx context.Context, path string) error {
	return w.internal.Delete(ctx, path)
}

// GetMetadata implements the GetMetadata method for the pkg domain interface
func (w *StorageServiceWrapper) GetMetadata(ctx context.Context, key string) (*pkgdomain.FileMetadata, error) {
	// Since the internal interface doesn't provide all metadata fields,
	// we return a basic metadata object with available values
	return &pkgdomain.FileMetadata{
		Key:          key,
		Size:         0, // Default since internal doesn't provide it
		ContentType:  "application/octet-stream",
		UploadedAt:   time.Now(),
		LastModified: time.Now(),
		Metadata:     make(map[string]string),
	}, nil
}

// GetQuotaUsage implements the GetQuotaUsage method for the pkg domain interface
func (w *StorageServiceWrapper) GetQuotaUsage(ctx context.Context) (int64, error) {
	// Since the internal interface doesn't provide quota usage,
	// we return a default value of 0
	return 0, nil
}

// ListFiles implements the ListFiles method for the pkg domain interface
func (w *StorageServiceWrapper) ListFiles(ctx context.Context, prefix string) ([]*pkgdomain.FileMetadata, error) {
	// Since the internal interface doesn't support listing files,
	// return an empty list
	return []*pkgdomain.FileMetadata{}, nil
}

// ValidateUpload implements the ValidateUpload method for the pkg domain interface
func (w *StorageServiceWrapper) ValidateUpload(ctx context.Context, fileSize int64, mimeType string) error {
	// Since the internal interface doesn't provide validation,
	// we accept all uploads
	return nil
}

// UploadAudio implements the UploadAudio method for the pkg domain interface
func (w *StorageServiceWrapper) UploadAudio(ctx context.Context, file io.Reader, path string) error {
	return w.internal.Upload(ctx, path, file)
}

// InternalStorageAdapter adapts a pkg domain storage service to internal domain storage service
type InternalStorageAdapter struct {
	pkg pkgdomain.StorageService
}

// NewInternalStorageAdapter creates a new adapter for pkg domain storage service
func NewInternalStorageAdapter(pkg pkgdomain.StorageService) domain.StorageService {
	return &InternalStorageAdapter{pkg: pkg}
}

// Upload implements domain.StorageService
func (a *InternalStorageAdapter) Upload(ctx context.Context, path string, content io.Reader) error {
	return a.pkg.UploadAudio(ctx, content, path)
}

// Download implements domain.StorageService
func (a *InternalStorageAdapter) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := a.pkg.Download(ctx, path)
	if err != nil {
		return nil, err
	}
	return file.Content.(io.ReadCloser), nil
}

// Delete implements domain.StorageService
func (a *InternalStorageAdapter) Delete(ctx context.Context, path string) error {
	return a.pkg.DeleteAudio(ctx, path)
}

// GetURL implements domain.StorageService
func (a *InternalStorageAdapter) GetURL(ctx context.Context, path string) (string, error) {
	return a.pkg.GetURL(ctx, path)
}
