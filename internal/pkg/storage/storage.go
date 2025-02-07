package storage

import (
	"context"
	"fmt"
	"metadatatool/internal/config"
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
	cfg    *config.StorageConfig
}

// NewStorageService creates a new storage service
func NewStorageService(cfg config.StorageConfig) (domain.StorageService, error) {
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
	metadata := make(map[string]string)
	for k, v := range file.Metadata {
		metadata[k] = v
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(file.Key),
		Body:        file.Content,
		ContentType: aws.String(file.ContentType),
		Metadata:    metadata,
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
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

// GetSignedURL generates a pre-signed URL for a file
func (s *StorageService) GetSignedURL(ctx context.Context, key string, operation domain.SignedURLOperation, expiry time.Duration) (string, error) {
	var presignClient *s3.PresignClient
	presignClient = s3.NewPresignClient(s.client)

	var presignedURL string
	var err error

	switch operation {
	case domain.SignedURLUpload:
		req, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(expiry))
		if err != nil {
			return "", fmt.Errorf("failed to generate upload URL: %w", err)
		}
		presignedURL = req.URL

	case domain.SignedURLDownload:
		req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(expiry))
		if err != nil {
			return "", fmt.Errorf("failed to generate download URL: %w", err)
		}
		presignedURL = req.URL

	default:
		return "", fmt.Errorf("unsupported signed URL operation: %v", operation)
	}

	return presignedURL, err
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

// GetQuotaUsage gets the total storage usage for a user
func (s *StorageService) GetQuotaUsage(ctx context.Context, userID string) (int64, error) {
	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fmt.Sprintf("users/%s/", userID)),
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
func (s *StorageService) ValidateUpload(ctx context.Context, filename string, size int64, userID string) error {
	// Check file size
	if size > s.cfg.MaxFileSize {
		return &domain.StorageError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("file size %d exceeds maximum allowed size %d", size, s.cfg.MaxFileSize),
			Op:      "ValidateUpload",
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
			Op:      "ValidateUpload",
		}
	}

	// Check user quota if userID is provided
	if userID != "" {
		usage, err := s.GetQuotaUsage(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to check quota: %w", err)
		}

		if usage+size > s.cfg.UserQuota {
			return &domain.StorageError{
				Code:    "QUOTA_EXCEEDED",
				Message: "user storage quota exceeded",
				Op:      "ValidateUpload",
			}
		}
	}

	return nil
}

// CleanupTempFiles removes expired temporary files
func (s *StorageService) CleanupTempFiles(ctx context.Context) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("cleanup"))
	defer timer.ObserveDuration()

	var tempFiles int

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(domain.StoragePathTemp),
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
