package storage

import (
	"context"
	"fmt"
	"io"
	"metadatatool/internal/pkg/config"
	"metadatatool/internal/pkg/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Storage struct {
	client  *s3.Client
	bucket  string
	cfg     *config.StorageConfig
	quotaMu sync.RWMutex
}

// NewS3Storage creates a new S3 storage service
func NewS3Storage(cfg *config.StorageConfig) (pkgdomain.StorageService, error) {
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

// generateKey generates a storage key
func generateKey(pathType domain.StoragePathType, filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("%s/%s/%s%s", pathType.String(), timestamp[:8], timestamp, ext)
}

// Upload uploads a file to S3
func (s *s3Storage) Upload(ctx context.Context, file *domain.StorageFile) error {
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("s3_upload"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("s3_upload", "started").Inc()

	// Generate storage key if not provided
	if file.Key == "" {
		file.Key = generateKey(domain.StoragePathPerm, file.Name)
	}

	// Validate upload
	if err := s.ValidateUpload(ctx, file.Size, file.ContentType); err != nil {
		metrics.AudioOpErrors.WithLabelValues("s3_upload", "validation_error").Inc()
		return err
	}

	// Convert metadata to AWS format
	awsMetadata := make(map[string]string)
	for k, v := range file.Metadata {
		awsMetadata[k] = v
	}

	// Upload file
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(file.Key),
		Body:        file.Content,
		ContentType: aws.String(file.ContentType),
		Metadata:    awsMetadata,
	})

	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("s3_upload", "s3_error").Inc()
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	metrics.AudioOps.WithLabelValues("s3_upload", "completed").Inc()
	return nil
}

// Download downloads a file from S3
func (s *s3Storage) Download(ctx context.Context, key string) (*domain.StorageFile, error) {
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("s3_download"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("s3_download", "started").Inc()

	// Get object
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("s3_download", "s3_error").Inc()
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	// Convert metadata from AWS format
	metadata := make(map[string]string)
	for k, v := range result.Metadata {
		metadata[k] = v
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	file := &domain.StorageFile{
		Key:         key,
		Name:        filepath.Base(key),
		Size:        size,
		ContentType: aws.ToString(result.ContentType),
		Content:     result.Body,
		Metadata:    metadata,
	}

	metrics.AudioOps.WithLabelValues("s3_download", "completed").Inc()
	return file, nil
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

// GetURL generates a pre-signed URL for the file
func (s *s3Storage) GetURL(ctx context.Context, key string) (string, error) {
	timer := metrics.NewTimer(metrics.AudioOpDurations.WithLabelValues("s3_get_url"))
	defer timer.ObserveDuration()

	metrics.AudioOps.WithLabelValues("s3_get_url", "started").Inc()

	// Create presigner
	presigner := s3.NewPresignClient(s.client)

	// Generate presigned URL
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(15 * time.Minute)
	})

	if err != nil {
		metrics.AudioOpErrors.WithLabelValues("s3_get_url", "presign_error").Inc()
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	metrics.AudioOps.WithLabelValues("s3_get_url", "completed").Inc()
	return request.URL, nil
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

// GetQuotaUsage gets the total storage usage
func (s *s3Storage) GetQuotaUsage(ctx context.Context) (int64, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("get_quota"))
	defer timer.ObserveDuration()

	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
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

	metrics.StorageOperationSuccess.WithLabelValues("get_quota").Inc()
	return totalSize, nil
}

// GetUserQuotaUsage gets the total storage usage for a specific user
func (s *s3Storage) GetUserQuotaUsage(ctx context.Context, userID string) (int64, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("get_user_quota"))
	defer timer.ObserveDuration()

	var totalSize int64

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fmt.Sprintf("users/%s/", userID)),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			metrics.StorageOperationErrors.WithLabelValues("get_user_quota").Inc()
			return 0, fmt.Errorf("failed to get user quota usage: %w", err)
		}

		for _, obj := range page.Contents {
			totalSize += aws.ToInt64(obj.Size)
		}
	}

	metrics.StorageQuotaUsage.WithLabelValues(userID).Set(float64(totalSize))
	metrics.StorageOperationSuccess.WithLabelValues("get_user_quota").Inc()
	return totalSize, nil
}

// ValidateUpload validates a file upload request
func (s *s3Storage) ValidateUpload(ctx context.Context, fileSize int64, mimeType string) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("validate_upload"))
	defer timer.ObserveDuration()

	// Check file size
	if fileSize > s.cfg.MaxFileSize {
		return &domain.StorageError{
			Code:    "FILE_TOO_LARGE",
			Message: fmt.Sprintf("file size %d exceeds maximum allowed size %d", fileSize, s.cfg.MaxFileSize),
		}
	}

	// Check file type
	allowed := false
	for _, allowedType := range s.cfg.AllowedFileTypes {
		if mimeType == allowedType {
			allowed = true
			break
		}
	}
	if !allowed {
		return &domain.StorageError{
			Code:    "INVALID_FILE_TYPE",
			Message: fmt.Sprintf("file type %s is not allowed", mimeType),
		}
	}

	// Check total quota
	usage, err := s.GetQuotaUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to check quota: %w", err)
	}

	if usage+fileSize > s.cfg.TotalQuota {
		return &domain.StorageError{
			Code:    "QUOTA_EXCEEDED",
			Message: "total storage quota exceeded",
		}
	}

	return nil
}

// CleanupTempFiles removes expired temporary files
func (s *s3Storage) CleanupTempFiles(ctx context.Context) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("cleanup"))
	defer timer.ObserveDuration()

	var deletedCount int64
	var totalSize int64

	// List all temporary files
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(domain.StoragePathTemp.String()),
	})

	now := time.Now()
	var objectsToDelete []types.ObjectIdentifier

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			metrics.StorageOperationErrors.WithLabelValues("cleanup").Inc()
			return fmt.Errorf("failed to list temporary files: %w", err)
		}

		for _, obj := range page.Contents {
			// Check if file is expired based on LastModified
			if now.Sub(*obj.LastModified) > s.cfg.TempFileExpiry {
				objectsToDelete = append(objectsToDelete, types.ObjectIdentifier{
					Key: obj.Key,
				})
				deletedCount++
				totalSize += aws.ToInt64(obj.Size)

				// Delete in batches of 1000 (S3 limit)
				if len(objectsToDelete) >= 1000 {
					if err := s.deleteObjects(ctx, objectsToDelete); err != nil {
						return err
					}
					objectsToDelete = objectsToDelete[:0]
				}
			}
		}
	}

	// Delete any remaining objects
	if len(objectsToDelete) > 0 {
		if err := s.deleteObjects(ctx, objectsToDelete); err != nil {
			return err
		}
	}

	metrics.StorageCleanupFilesDeleted.Add(float64(deletedCount))
	metrics.StorageCleanupBytesReclaimed.Add(float64(totalSize))
	metrics.StorageOperationSuccess.WithLabelValues("cleanup").Inc()

	return nil
}

// deleteObjects deletes a batch of objects from S3
func (s *s3Storage) deleteObjects(ctx context.Context, objects []types.ObjectIdentifier) error {
	_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("cleanup_batch").Inc()
		return fmt.Errorf("failed to delete objects batch: %w", err)
	}
	return nil
}

// DeleteAudio deletes an audio file from storage
func (s *s3Storage) DeleteAudio(ctx context.Context, path string) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("delete_audio"))
	defer timer.ObserveDuration()

	// Delete file
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("delete_audio").Inc()
		return fmt.Errorf("failed to delete audio file: %w", err)
	}

	metrics.StorageOperationSuccess.WithLabelValues("delete_audio").Inc()
	return nil
}

// GetSignedURL generates a pre-signed URL for the file with expiry
func (s *s3Storage) GetSignedURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("get_signed_url"))
	defer timer.ObserveDuration()

	// Create presigner
	presigner := s3.NewPresignClient(s.client)

	// Generate presigned URL
	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("get_signed_url").Inc()
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	metrics.StorageOperationSuccess.WithLabelValues("get_signed_url").Inc()
	return request.URL, nil
}

// UploadAudio uploads an audio file to storage
func (s *s3Storage) UploadAudio(ctx context.Context, file io.Reader, path string) error {
	timer := metrics.NewTimer(metrics.StorageOperationDuration.WithLabelValues("upload_audio"))
	defer timer.ObserveDuration()

	// Upload file
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   file,
	})

	if err != nil {
		metrics.StorageOperationErrors.WithLabelValues("upload_audio").Inc()
		return fmt.Errorf("failed to upload audio file: %w", err)
	}

	metrics.StorageOperationSuccess.WithLabelValues("upload_audio").Inc()
	return nil
}
