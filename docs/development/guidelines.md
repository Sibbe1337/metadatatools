# Development Guidelines

## Code Organization

### Project Structure
```plaintext
metadatatool/
├── cmd/                  # Application entry points
├── internal/            # Private application code
│   ├── domain/         # Business domain models
│   ├── usecase/        # Application business rules
│   ├── repository/     # Data access layer
│   ├── handler/        # HTTP handlers
│   ├── config/         # Configuration
│   └── pkg/            # Shared packages
├── pkg/                # Public packages
├── docs/               # Documentation
└── scripts/            # Build and maintenance scripts
```

## Coding Standards

### Go Code Style
1. Follow standard Go formatting:
   ```bash
   # Format code
   go fmt ./...
   
   # Run linter
   golangci-lint run
   ```

2. Package Organization:
   ```go
   package example

   import (
       // Standard library
       "context"
       "fmt"
       
       // Third party
       "github.com/example/pkg"
       
       // Internal packages
       "metadatatool/internal/domain"
   )
   ```

3. Error Handling:
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to process request: %w", err)
   }

   // Bad
   if err != nil {
       return err
   }
   ```

### Clean Architecture Guidelines

1. **Domain Layer**
   ```go
   // Good
   type Track struct {
       ID       string
       Title    string
       Metadata Metadata
   }

   // Bad - mixing concerns
   type Track struct {
       ID       string
       Title    string
       DBFields map[string]interface{}
   }
   ```

2. **Use Case Layer**
   ```go
   // Good
   type TrackUseCase struct {
       repo domain.TrackRepository
       ai   domain.AIService
   }

   // Bad - direct database access
   type TrackUseCase struct {
       db *sql.DB
   }
   ```

3. **Repository Layer**
   ```go
   // Good
   func (r *TrackRepository) Create(ctx context.Context, track *domain.Track) error {
       // Implementation
   }

   // Bad - exposing implementation details
   func (r *TrackRepository) CreateInPostgres(track *domain.Track) error {
       // Implementation
   }
   ```

## Testing Guidelines

### Unit Tests
```go
func TestTrackUseCase_Create(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMockTrackRepository()
    useCase := NewTrackUseCase(mockRepo)
    
    // Act
    result, err := useCase.Create(context.Background(), track)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Tests
```go
func TestTrackAPI_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    // Test implementation
}
```

### Performance Tests
```go
func BenchmarkTrackProcessing(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Benchmark implementation
    }
}
```

## Documentation Guidelines

### Code Documentation
```go
// TrackService handles track-related operations
type TrackService interface {
    // Create creates a new track with the given details
    // Returns ErrInvalidInput if the track data is invalid
    Create(ctx context.Context, track *Track) error
}
```

### API Documentation
```go
// @Summary Create track
// @Description Create a new track with metadata
// @Tags tracks
// @Accept json
// @Produce json
// @Param track body Track true "Track object"
// @Success 200 {object} Response
// @Router /tracks [post]
func (h *Handler) CreateTrack(c *gin.Context) {
    // Implementation
}
```

## Error Handling

### Error Types
```go
type ErrorCode string

const (
    ErrNotFound     ErrorCode = "NOT_FOUND"
    ErrInvalidInput ErrorCode = "INVALID_INPUT"
    ErrInternal     ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
    Code    ErrorCode
    Message string
    Err     error
}
```

### Error Handling Pattern
```go
func (s *service) Process(ctx context.Context) error {
    result, err := s.repository.Get(ctx)
    if err != nil {
        return &AppError{
            Code:    ErrInternal,
            Message: "Failed to get data",
            Err:     err,
        }
    }
    return nil
}
```

## Performance Guidelines

### Database Access
```go
// Good - Using indexes
db.Where("created_at > ?", time.Now().Add(-24*time.Hour))

// Bad - Full table scan
db.Where("EXTRACT(day FROM created_at) = ?", time.Now().Day())
```

### Caching Strategy
```go
func (s *service) GetTrack(ctx context.Context, id string) (*Track, error) {
    // Try cache first
    if cached, err := s.cache.Get(ctx, id); err == nil {
        return cached, nil
    }
    
    // Get from database
    track, err := s.repo.Get(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Update cache
    s.cache.Set(ctx, id, track)
    return track, nil
}
```

## Security Guidelines

### Input Validation
```go
func validateTrack(track *Track) error {
    if track.Title == "" {
        return ErrInvalidInput("title is required")
    }
    if len(track.Title) > 255 {
        return ErrInvalidInput("title too long")
    }
    return nil
}
```

### Authentication
```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if !validateToken(token) {
            c.AbortWithStatus(http.StatusUnauthorized)
            return
        }
        c.Next()
    }
}
```

## Monitoring Guidelines

### Metrics
```go
func (s *service) Process(ctx context.Context) error {
    start := time.Now()
    defer func() {
        metrics.ProcessingDuration.Observe(time.Since(start).Seconds())
    }()
    
    // Implementation
}
```

### Logging
```go
func (s *service) Process(ctx context.Context) error {
    logger := log.WithContext(ctx).With(
        "operation", "process",
        "user_id", ctx.Value("user_id"),
    )
    
    logger.Info("Starting processing")
    // Implementation
    logger.Info("Processing completed")
}
```

## Version Control Guidelines

### Commit Messages
```plaintext
feat: add track metadata enrichment
^--^  ^------------------------^
|     |
|     +-> Summary in present tense
|
+-------> Type: feat, fix, docs, style, refactor, test, chore
```

### Branch Naming
```plaintext
feature/track-enrichment
bugfix/metadata-validation
docs/api-documentation
```

## CI/CD Guidelines

### Pipeline Stages
```yaml
stages:
  - lint
  - test
  - build
  - deploy

lint:
  script:
    - golangci-lint run

test:
  script:
    - go test ./...

build:
  script:
    - go build ./...
```

## Dependencies Management

### Adding Dependencies
```bash
# Add direct dependency
go get -u github.com/example/pkg

# Add development dependency
go get -u -d github.com/example/pkg
```

### Updating Dependencies
```bash
# Update all dependencies
go get -u ./...

# Update specific dependency
go get -u github.com/example/pkg
```

## Review Guidelines

### Code Review Checklist
```plaintext
1. Architecture
   - Follows clean architecture
   - Proper separation of concerns
   - Clear dependencies

2. Code Quality
   - Follows coding standards
   - Proper error handling
   - Adequate testing

3. Performance
   - Efficient algorithms
   - Proper resource usage
   - Caching strategy

4. Security
   - Input validation
   - Authentication/Authorization
   - Secure communication
``` 