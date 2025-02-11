package storage

import (
	"context"
	"fmt"
	"io"
	pkgconfig "metadatatool/internal/pkg/config"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageService struct {
	client *s3.Client
	bucket string
	cfg    *pkgconfig.StorageConfig
}

// NewStorageService creates a new storage service
func NewStorageService(cfg pkgconfig.StorageConfig) (domain.StorageService, error) {
	// Create AWS config
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg)

	return &StorageService{
		client: client,
		bucket: cfg.Bucket,
		cfg:    &cfg,
	}, nil
}

// Upload uploads a file to storage
func (s *StorageService) Upload(ctx context.Context, file *domain.StorageFile) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(file.Key),
		Body:        file.Content,
		ContentType: aws.String(file.ContentType),
		Metadata:    file.Metadata,
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadAudio uploads an audio file to storage
func (s *StorageService) UploadAudio(ctx context.Context, file io.Reader, path string) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   file,
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload audio file: %w", err)
	}

	return nil
}

// GetURL returns a pre-signed URL for the given key
func (s *StorageService) GetURL(ctx context.Context, key string) (string, error) {
	presignClient := s3.NewPresignClient(s.client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour * 24 // 24 hour expiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}
	return request.URL, nil
}

// Download downloads a file from storage
func (s *StorageService) Download(ctx context.Context, key string) (*domain.StorageFile, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	// Get object metadata
	head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	// Create StorageFile with metadata
	file := &domain.StorageFile{
		Key:         key,
		Name:        filepath.Base(key),
		Size:        aws.ToInt64(head.ContentLength),
		ContentType: aws.ToString(head.ContentType),
		Content:     result.Body,
		Metadata:    make(map[string]string),
	}

	// Copy metadata
	for k, v := range head.Metadata {
		file.Metadata[k] = v // v is already a string, no need for aws.ToString
	}

	return file, nil
}

// Delete deletes a file from storage
func (s *StorageService) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// DeleteAudio deletes an audio file from storage
func (s *StorageService) DeleteAudio(ctx context.Context, path string) error {
	return s.Delete(ctx, path)
}

// GetSignedURL gets a pre-signed URL for a file
func (s *StorageService) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)
	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate pre-signed URL: %w", err)
	}
	return request.URL, nil
}

// GetMetadata gets metadata for a file
func (s *StorageService) GetMetadata(ctx context.Context, key string) (*domain.FileMetadata, error) {
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	return &domain.FileMetadata{
		Key:          key,
		Size:         aws.ToInt64(result.ContentLength),
		ContentType:  aws.ToString(result.ContentType),
		LastModified: *result.LastModified,
		ETag:         aws.ToString(result.ETag),
		StorageClass: string(result.StorageClass),
	}, nil
}

// ListFiles lists files with the given prefix
func (s *StorageService) ListFiles(ctx context.Context, prefix string) ([]*domain.FileMetadata, error) {
	var files []*domain.FileMetadata

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
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

	return files, nil
}

// GetQuotaUsage gets the total storage usage
func (s *StorageService) GetQuotaUsage(ctx context.Context) (int64, error) {
	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get quota usage: %w", err)
		}

		for _, obj := range page.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}
	}

	return totalSize, nil
}

// ValidateUpload validates a file upload request
func (s *StorageService) ValidateUpload(ctx context.Context, fileSize int64, mimeType string) error {
	// Check file size
	if fileSize > s.cfg.MaxFileSize {
		return &domain.StorageError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("File size %d exceeds maximum allowed size %d", fileSize, s.cfg.MaxFileSize),
		}
	}

	// Check file type
	if !s.isAllowedFileType(mimeType) {
		return &domain.StorageError{
			Code:    "INVALID_FILE_TYPE",
			Message: fmt.Sprintf("File type %s is not allowed", mimeType),
		}
	}

	// Check quota
	quotaUsage, err := s.GetQuotaUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to check quota: %w", err)
	}

	if quotaUsage+fileSize > s.cfg.TotalQuota {
		return &domain.StorageError{
			Code:    "QUOTA_EXCEEDED",
			Message: "Storage quota exceeded",
		}
	}

	return nil
}

// isAllowedFileType checks if a file type is allowed
func (s *StorageService) isAllowedFileType(mimeType string) bool {
	for _, allowed := range s.cfg.AllowedFileTypes {
		if strings.EqualFold(mimeType, allowed) {
			return true
		}
	}
	return false
}

// CleanupTempFiles removes expired temporary files
func (s *StorageService) CleanupTempFiles(ctx context.Context) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("cleanup"))
	defer timer.ObserveDuration()

	var tempFiles int

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(domain.StoragePathTemp.String()),
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
