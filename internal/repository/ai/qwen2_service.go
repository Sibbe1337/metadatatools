package ai

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"
)

// Qwen2Service implements the AIService interface using Qwen2-Audio
type Qwen2Service struct {
	config  *domain.Qwen2Config
	client  *Qwen2Client
	metrics *domain.AIMetrics
	mu      sync.RWMutex // Protects metrics
}

// NewQwen2Service creates a new Qwen2Service instance
func NewQwen2Service(config *domain.Qwen2Config) (*Qwen2Service, error) {
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
		metrics: &domain.AIMetrics{
			RequestCount:   0,
			SuccessCount:   0,
			FailureCount:   0,
			AverageLatency: 0,
		},
	}, nil
}

// EnrichMetadata processes an audio track and enriches it with AI-generated metadata
func (s *Qwen2Service) EnrichMetadata(ctx context.Context, track *domain.Track) error {
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

		// Process the audio file
		response, err := s.client.AnalyzeAudio(ctx, track.AudioData, track.AudioFormat)
		if err != nil {
			processingErr = fmt.Errorf("attempt %d: failed to analyze audio: %w", attempt+1, err)
			continue // Try again if we have attempts left
		}

		// Check confidence threshold
		if response.Confidence < s.config.MinConfidence {
			track.NeedsReview = true
			track.AIMetadata = &domain.AIMetadata{
				Provider:     domain.AIProviderQwen2,
				Energy:       response.Energy,
				Danceability: response.Danceability,
				ProcessedAt:  time.Now(),
				ProcessingMs: time.Since(startTime).Milliseconds(),
				NeedsReview:  true,
				ReviewReason: fmt.Sprintf("Low confidence score: %.2f", response.Confidence),
			}
		} else {
			track.NeedsReview = false
			track.AIMetadata = &domain.AIMetadata{
				Provider:     domain.AIProviderQwen2,
				Energy:       response.Energy,
				Danceability: response.Danceability,
				ProcessedAt:  time.Now(),
				ProcessingMs: time.Since(startTime).Milliseconds(),
				NeedsReview:  false,
			}
		}

		// Update metrics
		duration := time.Since(startTime)
		s.recordSuccess(duration)
		metrics.AIConfidenceScore.WithLabelValues(string(domain.AIProviderQwen2)).Observe(response.Confidence)

		return nil // Successfully processed
	}

	// If we get here, we've exhausted all retry attempts
	s.recordFailure(processingErr)
	return fmt.Errorf("failed to enrich metadata after %d attempts: %w", s.config.RetryAttempts, processingErr)
}

// ValidateMetadata validates track metadata using AI
func (s *Qwen2Service) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
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
		metrics.AIConfidenceScore.WithLabelValues(string(domain.AIProviderQwen2)).Observe(confidence)

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
func (s *Qwen2Service) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
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
		go func(track *domain.Track) {
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
	metrics.AIBatchSize.WithLabelValues(string(domain.AIProviderQwen2)).Observe(float64(len(tracks)))
	metrics.AIRequestDuration.WithLabelValues(string(domain.AIProviderQwen2)).Observe(duration.Seconds())

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
	metrics.AIRequestTotal.WithLabelValues(string(domain.AIProviderQwen2), "success").Inc()
	metrics.AIRequestDuration.WithLabelValues(string(domain.AIProviderQwen2)).Observe(duration.Seconds())
}

// recordFailure updates metrics for failed requests
func (s *Qwen2Service) recordFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestCount++
	s.metrics.FailureCount++
	s.metrics.LastError = err

	// Update Prometheus metrics
	metrics.AIRequestTotal.WithLabelValues(string(domain.AIProviderQwen2), "failure").Inc()
	metrics.AIErrorTotal.WithLabelValues(string(domain.AIProviderQwen2), err.Error()).Inc()
}

// getMetrics returns a copy of current metrics
func (s *Qwen2Service) getMetrics() *domain.AIMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &domain.AIMetrics{
		RequestCount:   s.metrics.RequestCount,
		SuccessCount:   s.metrics.SuccessCount,
		FailureCount:   s.metrics.FailureCount,
		LastSuccess:    s.metrics.LastSuccess,
		LastError:      s.metrics.LastError,
		AverageLatency: s.metrics.AverageLatency,
	}
}
