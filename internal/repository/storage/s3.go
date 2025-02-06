package storage

import (
	"context"
	"fmt"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage creates a new S3 storage service
func NewS3Storage(cfg *config.StorageConfig) (domain.StorageService, error) {
	// Create custom endpoint resolver if endpoint is specified
	var endpointResolver aws.EndpointResolverWithOptions
	if cfg.Endpoint != "" {
		endpointResolver = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		})
	}

	// Configure AWS SDK
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
		awsconfig.WithEndpointResolverWithOptions(endpointResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for some S3-compatible services
	})

	return &s3Storage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Upload stores a file in S3
func (s *s3Storage) Upload(ctx context.Context, file *domain.File) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(file.Key),
		Body:        file.Content,
		ContentType: aws.String(file.ContentType),
		Metadata:    file.Metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

// Download retrieves a file from S3
func (s *s3Storage) Download(ctx context.Context, key string) (*domain.File, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	var size int64
	if result.ContentLength != nil {
		size = *result.ContentLength
	}

	return &domain.File{
		Key:         key,
		Content:     result.Body,
		Size:        size,
		ContentType: aws.ToString(result.ContentType),
		Metadata:    result.Metadata,
		UploadedAt:  aws.ToTime(result.LastModified),
	}, nil
}

// Delete removes a file from S3
func (s *s3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetSignedURL generates a pre-signed URL for direct upload/download
func (s *s3Storage) GetSignedURL(ctx context.Context, key string, operation domain.SignedURLOperation, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

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
		return "", fmt.Errorf("unsupported signed URL operation: %s", operation)
	}

	return presignedURL, err
}
