package ai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	pkgdomain "metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"
)

// Qwen2ClientInterface defines the interface for Qwen2 client operations
type Qwen2ClientInterface interface {
	AnalyzeAudio(ctx context.Context, audioData io.Reader, format pkgdomain.AudioFormat) (*Qwen2Response, error)
	ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (float64, error)
}

// Qwen2Service implements pkg/domain.AIService interface
type Qwen2Service struct {
	config  *pkgdomain.Qwen2Config
	client  Qwen2ClientInterface
	metrics *pkgdomain.AIMetrics
	mu      sync.RWMutex // Protects metrics
}

// NewQwen2Service creates a new Qwen2Service instance
func NewQwen2Service(config *pkgdomain.Qwen2Config) (pkgdomain.AIService, error) {
	if config == nil {
		return nil, fmt.Errorf("qwen2 config is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("qwen2 API key is required")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("qwen2 endpoint is required")
	}

	// Create Qwen2 client
	client, err := NewQwen2Client(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create qwen2 client: %w", err)
	}

	return &Qwen2Service{
		config: config,
		client: client,
		metrics: &pkgdomain.AIMetrics{
			RequestCount:   0,
			SuccessCount:   0,
			FailureCount:   0,
			AverageLatency: 0,
		},
	}, nil
}

// NewQwen2ServiceWithClient creates a new Qwen2Service instance with a provided client
func NewQwen2ServiceWithClient(config *pkgdomain.Qwen2Config, client Qwen2ClientInterface) (pkgdomain.AIService, error) {
	if config == nil {
		return nil, fmt.Errorf("qwen2 config is required")
	}

	if client == nil {
		return nil, fmt.Errorf("qwen2 client is required")
	}

	return &Qwen2Service{
		config: config,
		client: client,
		metrics: &pkgdomain.AIMetrics{
			RequestCount:   0,
			SuccessCount:   0,
			FailureCount:   0,
			AverageLatency: 0,
		},
	}, nil
}

// EnrichMetadata processes an audio track and enriches it with AI-generated metadata
func (s *Qwen2Service) EnrichMetadata(ctx context.Context, track *pkgdomain.Track) error {
	if track == nil {
		return fmt.Errorf("track is required")
	}

	startTime := time.Now()
	var processingErr error

	// Define retry strategy
	for attempt := 0; attempt <= s.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(attempt) * time.Second * time.Duration(s.config.RetryBackoffSeconds)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		// Convert []byte to io.Reader
		audioReader := bytes.NewReader(track.AudioData)
		format := pkgdomain.AudioFormat(track.AudioFormat())

		// Call Qwen2 API
		response, err := s.client.AnalyzeAudio(ctx, audioReader, format)
		if err != nil {
			processingErr = fmt.Errorf("attempt %d: failed to analyze audio: %w", attempt+1, err)
			continue
		}

		// Check confidence threshold
		if response.Metadata.Confidence < s.config.MinConfidence {
			track.Metadata.AI = &pkgdomain.TrackAIMetadata{
				Tags:         response.Metadata.Tags,
				Confidence:   response.Metadata.Confidence,
				Model:        "qwen2",
				Version:      "v1",
				ProcessedAt:  time.Now(),
				NeedsReview:  true,
				ReviewReason: fmt.Sprintf("Low confidence score: %.2f", response.Metadata.Confidence),
			}
		} else {
			track.Metadata.AI = &pkgdomain.TrackAIMetadata{
				Tags:        response.Metadata.Tags,
				Confidence:  response.Metadata.Confidence,
				Model:       "qwen2",
				Version:     "v1",
				ProcessedAt: time.Now(),
				NeedsReview: false,
			}
		}

		// Update track fields
		track.SetGenre(response.Metadata.Genre)
		track.SetBPM(response.Metadata.BPM)
		track.SetKey(response.Metadata.Key)
		track.SetMood(response.Metadata.Mood)

		// Update metrics
		duration := time.Since(startTime)
		s.recordSuccess(duration)
		metrics.AIConfidenceScore.WithLabelValues(string(pkgdomain.AIProviderQwen2)).Observe(response.Metadata.Confidence)

		return nil // Successfully processed
	}

	// If we get here, we've exhausted all retry attempts
	s.recordFailure(processingErr)
	return fmt.Errorf("failed to enrich metadata after %d attempts: %w", s.config.RetryAttempts, processingErr)
}

// ValidateMetadata validates track metadata using AI
func (s *Qwen2Service) ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (float64, error) {
	if track == nil {
		return 0, fmt.Errorf("track is required")
	}

	startTime := time.Now()
	var processingErr error

	// Define retry strategy
	for attempt := 0; attempt <= s.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(attempt) * time.Second * time.Duration(s.config.RetryBackoffSeconds)
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(backoffDuration):
			}
		}

		// Validate metadata
		confidence, err := s.client.ValidateMetadata(ctx, track)
		if err != nil {
			processingErr = fmt.Errorf("attempt %d: failed to validate metadata: %w", attempt+1, err)
			continue // Try again if we have attempts left
		}

		// Update metrics
		duration := time.Since(startTime)
		s.recordSuccess(duration)
		metrics.AIConfidenceScore.WithLabelValues(string(pkgdomain.AIProviderQwen2)).Observe(confidence)

		// Return early if confidence meets threshold
		if confidence >= s.config.MinConfidence {
			return confidence, nil
		}

		// If confidence is too low, try again if we have attempts left
		processingErr = fmt.Errorf("confidence score too low: %.2f", confidence)
	}

	// If we get here, we've exhausted all retry attempts
	s.recordFailure(processingErr)
	return 0, fmt.Errorf("failed to validate metadata after %d attempts: %w", s.config.RetryAttempts, processingErr)
}

// BatchProcess processes multiple tracks in parallel
func (s *Qwen2Service) BatchProcess(ctx context.Context, tracks []*pkgdomain.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	startTime := time.Now()

	// Create a semaphore to limit concurrent requests
	sem := make(chan struct{}, s.config.MaxConcurrentRequests)
	errChan := make(chan error, len(tracks))
	var wg sync.WaitGroup

	// Process tracks in parallel
	for i := range tracks {
		wg.Add(1)
		go func(track *pkgdomain.Track) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Process individual track
			if err := s.EnrichMetadata(ctx, track); err != nil {
				errChan <- fmt.Errorf("failed to process track %s: %w", track.ID, err)
				return
			}
		}(tracks[i])
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// Update batch metrics
	duration := time.Since(startTime)
	metrics.AIBatchSize.WithLabelValues(string(pkgdomain.AIProviderQwen2)).Observe(float64(len(tracks)))
	metrics.AIRequestDuration.WithLabelValues(string(pkgdomain.AIProviderQwen2)).Observe(duration.Seconds())

	// Return combined errors if any
	if len(errors) > 0 {
		return fmt.Errorf("batch processing completed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// recordSuccess updates metrics for successful requests
func (s *Qwen2Service) recordSuccess(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestCount++
	s.metrics.SuccessCount++
	s.metrics.LastSuccess = time.Now()

	// Update average latency
	if s.metrics.AverageLatency == 0 {
		s.metrics.AverageLatency = duration
	} else {
		s.metrics.AverageLatency = (s.metrics.AverageLatency + duration) / 2
	}

	// Update Prometheus metrics
	metrics.AIRequestTotal.WithLabelValues(string(pkgdomain.AIProviderQwen2), "success").Inc()
	metrics.AIRequestDuration.WithLabelValues(string(pkgdomain.AIProviderQwen2)).Observe(duration.Seconds())
}

// recordFailure updates metrics for failed requests
func (s *Qwen2Service) recordFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestCount++
	s.metrics.FailureCount++
	s.metrics.LastError = err

	// Update Prometheus metrics
	metrics.AIRequestTotal.WithLabelValues(string(pkgdomain.AIProviderQwen2), "failure").Inc()
	metrics.AIErrorTotal.WithLabelValues(string(pkgdomain.AIProviderQwen2), err.Error()).Inc()
}
