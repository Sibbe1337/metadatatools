package storage

import (
	"context"
	"fmt"
	"io"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Storage struct {
	client  *s3.Client
	bucket  string
	cfg     *config.StorageConfig
	quotaMu sync.RWMutex
}

// NewS3Storage creates a new S3 storage service
func NewS3Storage(cfg *config.StorageConfig) (domain.StorageService, error) {
	if cfg.Bucket == "" {
		return nil, &domain.StorageError{
			Code:    "InvalidConfig",
			Message: "S3 bucket name is required",
			Op:      "NewS3Storage",
		}
	}

	// Create AWS credentials
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")

	// Configure AWS client
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithCredentialsProvider(creds),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	return &s3Storage{
		client: client,
		bucket: cfg.Bucket,
		cfg:    cfg,
	}, nil
}

// Upload stores a file in S3
func (s *s3Storage) Upload(ctx context.Context, key string, data io.Reader) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("upload"))
	defer timer.ObserveDuration()

	// Upload file
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   data,
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("upload").Inc()
		return fmt.Errorf("failed to upload file: %w", err)
	}

	metrics.StorageOperationSuccess.WithLabelValues("upload").Inc()
	return nil
}

// Download retrieves a file from S3
func (s *s3Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("download"))
	defer timer.ObserveDuration()

	// Download file
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("download").Inc()
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	metrics.StorageOperationSuccess.WithLabelValues("download").Inc()
	return result.Body, nil
}

// Delete removes a file from S3
func (s *s3Storage) Delete(ctx context.Context, key string) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("delete"))
	defer timer.ObserveDuration()

	// Delete file
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("delete").Inc()
		return fmt.Errorf("failed to delete file: %w", err)
	}

	metrics.StorageOperationSuccess.WithLabelValues("delete").Inc()
	return nil
}

// GetSignedURL generates a pre-signed URL for direct upload/download
func (s *s3Storage) GetSignedURL(ctx context.Context, key string, operation domain.SignedURLOperation, expiry time.Duration) (string, error) {
	// TODO: Implement pre-signed URL generation
	return "", fmt.Errorf("not implemented")
}

// GetMetadata retrieves metadata for a file
func (s *s3Storage) GetMetadata(ctx context.Context, key string) (*domain.FileMetadata, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("get_metadata"))
	defer timer.ObserveDuration()

	// Get object metadata
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("get_metadata").Inc()
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	metadata := &domain.FileMetadata{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		ContentType:  aws.ToString(result.ContentType),
		LastModified: *result.LastModified,
		ETag:         aws.ToString(result.ETag),
	}

	metrics.StorageOperationSuccess.WithLabelValues("get_metadata").Inc()
	return metadata, nil
}

// ListFiles lists files with the given prefix
func (s *s3Storage) ListFiles(ctx context.Context, prefix string) ([]*domain.FileMetadata, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("list_files"))
	defer timer.ObserveDuration()

	var files []*domain.FileMetadata

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			metrics.StorageOperationErrors.WithLabelValues("list_files").Inc()
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		for _, obj := range page.Contents {
			files = append(files, &domain.FileMetadata{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: *obj.LastModified,
				ETag:         aws.ToString(obj.ETag),
				StorageClass: string(obj.StorageClass),
			})
		}
	}

	metrics.StorageOperationSuccess.WithLabelValues("list_files").Inc()
	return files, nil
}

// GetQuotaUsage gets the total storage usage for a user
func (s *s3Storage) GetQuotaUsage(ctx context.Context, userID string) (int64, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("get_quota"))
	defer timer.ObserveDuration()

	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fmt.Sprintf("users/%s/", userID)),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			metrics.StorageOperationErrors.WithLabelValues("get_quota").Inc()
			return 0, fmt.Errorf("failed to get quota usage: %w", err)
		}

		for _, obj := range page.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}
	}

	metrics.StorageQuotaUsage.WithLabelValues(userID).Set(float64(totalSize))
	metrics.StorageOperationSuccess.WithLabelValues("get_quota").Inc()
	return totalSize, nil
}

// ValidateUpload validates a file upload request
func (s *s3Storage) ValidateUpload(ctx context.Context, filename string, size int64, userID string) error {
	// Check file size
	if size > s.cfg.MaxFileSize {
		return &domain.StorageError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("file size %d exceeds maximum allowed size %d", size, s.cfg.MaxFileSize),
		}
	}

	// Check file type
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := false
	for _, allowedType := range s.cfg.AllowedFileTypes {
		if ext == allowedType {
			allowed = true
			break
		}
	}
	if !allowed {
		return &domain.StorageError{
			Code:    "INVALID_FILE_TYPE",
			Message: fmt.Sprintf("file type %s is not allowed", ext),
		}
	}

	// Check user quota
	usage, err := s.GetQuotaUsage(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check quota: %w", err)
	}

	if usage+size > s.cfg.UserQuota {
		return &domain.StorageError{
			Code:    "QUOTA_EXCEEDED",
			Message: "user storage quota exceeded",
		}
	}

	return nil
}

// CleanupTempFiles removes expired temporary files
func (s *s3Storage) CleanupTempFiles(ctx context.Context) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("cleanup"))
	defer timer.ObserveDuration()

	var tempFiles int

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String("temp/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			metrics.StorageOperationErrors.WithLabelValues("cleanup").Inc()
			return fmt.Errorf("failed to list temp files: %w", err)
		}

		for _, obj := range page.Contents {
			if time.Since(*obj.LastModified) > s.cfg.TempFileExpiry {
				err := s.Delete(ctx, aws.ToString(obj.Key))
				if err != nil {
					metrics.StorageOperationErrors.WithLabelValues("cleanup").Inc()
					continue
				}
				tempFiles++
			}
		}
	}

	metrics.StorageTempFileCount.Set(float64(tempFiles))
	metrics.StorageOperationSuccess.WithLabelValues("cleanup").Inc()
	return nil
}
