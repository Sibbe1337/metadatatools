# Storage Service Architecture

## Overview
The Storage Service handles file storage, retrieval, and management for audio files and related assets. It implements the `domain.StorageService` interface and primarily uses S3-compatible storage with support for quotas, cleanup, and monitoring.

## Components

### 1. Service Interface
```go
type StorageService interface {
    Upload(ctx context.Context, key string, data io.Reader) error
    Download(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    GetSignedURL(ctx context.Context, key string, operation SignedURLOperation, expiry time.Duration) (string, error)
    GetMetadata(ctx context.Context, key string) (*FileMetadata, error)
    ListFiles(ctx context.Context, prefix string) ([]*FileMetadata, error)
    GetQuotaUsage(ctx context.Context, userID string) (int64, error)
    ValidateUpload(ctx context.Context, filename string, size int64, userID string) error
    CleanupTempFiles(ctx context.Context) error
}
```

### 2. Implementation Structure
- **S3 Storage**: Primary implementation using AWS S3 or compatible services
- **Local Storage**: Development/testing implementation using local filesystem
- **Mock Storage**: Testing implementation for unit tests

### 3. Key Features

#### File Management
- Secure file upload/download
- Metadata extraction and storage
- File format validation
- Temporary file management
- Automatic cleanup

#### Quota Management
- Per-user storage quotas
- Total storage quota
- Usage tracking and alerts
- Quota enforcement

#### Security
- Pre-signed URLs
- Access control
- Encryption at rest
- Secure file transfer

### 4. Configuration

```go
type StorageConfig struct {
    // Provider settings
    Provider         string
    Region           string
    Bucket           string
    AccessKey        string
    SecretKey        string
    Endpoint         string
    UseSSL           bool
    UploadPartSize   int64
    MaxUploadRetries int

    // File restrictions
    MaxFileSize      int64
    AllowedFileTypes []string

    // Quota settings
    UserQuota       int64
    TotalQuota      int64
    QuotaWarningPct int

    // Cleanup settings
    TempFileExpiry  time.Duration
    CleanupInterval time.Duration

    // Performance settings
    UploadBufferSize int64
    DownloadTimeout  time.Duration
    UploadTimeout    time.Duration
}
```

### 5. Metrics

The service tracks the following metrics:
- Upload/download latency
- Storage operation success/failure rates
- Quota usage per user
- Temporary file count
- Storage operation errors

### 6. Error Handling

#### Error Types
- `StorageError`: Base error type for storage operations
- `QuotaExceededError`: User or total quota exceeded
- `ValidationError`: File validation failures
- `TimeoutError`: Operation timeout errors

#### Retry Strategy
- Configurable retry attempts
- Exponential backoff
- Operation-specific timeouts

### 7. Usage Examples

#### Uploading a File
```go
file := openFile("audio.mp3")
key := "uploads/user123/audio.mp3"
err := storageService.Upload(ctx, key, file)
```

#### Generating a Download URL
```go
url, err := storageService.GetSignedURL(ctx, key, domain.DownloadOperation, 1*time.Hour)
```

#### Checking Quota Usage
```go
usage, err := storageService.GetQuotaUsage(ctx, userID)
if usage > warningThreshold {
    // Send notification
}
```

## Future Improvements

1. **Enhanced Caching**
   - Implement caching layer for frequently accessed files
   - Cache metadata to reduce storage operations
   - Implement cache invalidation strategy

2. **Advanced File Processing**
   - Automatic format conversion
   - Audio quality validation
   - Metadata extraction pipeline

3. **Scalability**
   - Multi-region support
   - Cross-region replication
   - Load balancing

4. **Monitoring**
   - Enhanced usage analytics
   - Cost optimization metrics
   - Performance tracking

5. **Security**
   - Enhanced encryption options
   - Fine-grained access control
   - Audit logging 