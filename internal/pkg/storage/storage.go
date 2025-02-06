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

// S3Client implements the domain.StorageService interface
type S3Client struct {
	client *s3.Client
	bucket string
}

// NewStorageClient creates a new S3 storage client
func NewStorageClient(cfg *config.StorageConfig) domain.StorageService {
	// Load AWS configuration
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey,
			cfg.SecretKey,
			"",
		)),
	}

	// Add custom endpoint if specified
	if cfg.Endpoint != "" {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		})
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(customResolver))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		fmt.Printf("Unable to load SDK config: %v\n", err)
		return nil
	}

	client := s3.NewFromConfig(awsCfg)

	return &S3Client{
		client: client,
		bucket: cfg.Bucket,
	}
}

// Upload stores a file in S3
func (c *S3Client) Upload(ctx context.Context, file *domain.File) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(file.Key),
		Body:        file.Content,
		ContentType: aws.String(file.ContentType),
		Metadata:    file.Metadata,
	})
	return err
}

// Download retrieves a file from S3
func (c *S3Client) Download(ctx context.Context, key string) (*domain.File, error) {
	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return &domain.File{
		Key:         key,
		Content:     result.Body,
		Size:        aws.ToInt64(result.ContentLength),
		ContentType: aws.ToString(result.ContentType),
		Metadata:    result.Metadata,
		UploadedAt:  aws.ToTime(result.LastModified),
	}, nil
}

// Delete removes a file from S3
func (c *S3Client) Delete(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

// GetSignedURL generates a pre-signed URL for direct upload/download
func (c *S3Client) GetSignedURL(ctx context.Context, key string, operation domain.SignedURLOperation, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(c.client)

	switch operation {
	case domain.SignedURLDownload:
		request, err := presignClient.PresignGetObject(ctx,
			&s3.GetObjectInput{
				Bucket: aws.String(c.bucket),
				Key:    aws.String(key),
			},
			s3.WithPresignExpires(expiry),
		)
		if err != nil {
			return "", err
		}
		return request.URL, nil

	case domain.SignedURLUpload:
		request, err := presignClient.PresignPutObject(ctx,
			&s3.PutObjectInput{
				Bucket: aws.String(c.bucket),
				Key:    aws.String(key),
			},
			s3.WithPresignExpires(expiry),
		)
		if err != nil {
			return "", err
		}
		return request.URL, nil

	default:
		return "", fmt.Errorf("unsupported operation: %s", operation)
	}
}
