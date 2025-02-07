package ai

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"
)

// OpenAIService implements domain.AIService for OpenAI
type OpenAIService struct {
	config  *domain.OpenAIConfig
	client  *OpenAIClient
	metrics *domain.AIMetrics
	mu      sync.RWMutex
}

// NewOpenAIService creates a new OpenAI service
func NewOpenAIService(config *domain.OpenAIConfig) (*OpenAIService, error) {
	if config == nil {
		return nil, fmt.Errorf("OpenAI config is required")
	}

	client, err := NewOpenAIClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &OpenAIService{
		config:  config,
		client:  client,
		metrics: &domain.AIMetrics{},
	}, nil
}

// EnrichMetadata enriches a track with AI-generated metadata
func (s *OpenAIService) EnrichMetadata(ctx context.Context, track *domain.Track) error {
	start := time.Now()
	defer func() {
		metrics.AIRequestDuration.WithLabelValues("openai").Observe(time.Since(start).Seconds())
	}()

	if track == nil {
		return fmt.Errorf("track is required")
	}

	if track.AudioData == nil {
		return fmt.Errorf("track audio data is required")
	}

	result, err := s.client.AnalyzeAudio(ctx, track.AudioData, track.AudioFormat)
	if err != nil {
		s.recordFailure(err)
		return fmt.Errorf("failed to analyze audio with OpenAI: %w", err)
	}

	// Update track's AI metadata
	track.AIMetadata = &domain.AIMetadata{
		Provider:     domain.AIProviderOpenAI,
		Energy:       result.Energy,
		Danceability: result.Danceability,
		ProcessedAt:  time.Now(),
		ProcessingMs: time.Since(start).Milliseconds(),
		NeedsReview:  result.Confidence < s.config.MinConfidence,
		ReviewReason: result.ReviewReason,
	}

	s.recordSuccess(time.Since(start))
	return nil
}

// ValidateMetadata validates track metadata using AI
func (s *OpenAIService) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	start := time.Now()
	defer func() {
		metrics.AIRequestDuration.WithLabelValues("openai_validate").Observe(time.Since(start).Seconds())
	}()

	if track == nil {
		return 0, fmt.Errorf("track is required")
	}

	confidence, err := s.client.ValidateMetadata(ctx, track)
	if err != nil {
		s.recordFailure(err)
		return 0, fmt.Errorf("failed to validate metadata with OpenAI: %w", err)
	}

	s.recordSuccess(time.Since(start))
	return confidence, nil
}

// BatchProcess processes multiple tracks in batch
func (s *OpenAIService) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
	start := time.Now()
	defer func() {
		metrics.AIRequestDuration.WithLabelValues("openai_batch").Observe(time.Since(start).Seconds())
	}()

	if len(tracks) == 0 {
		return fmt.Errorf("at least one track is required")
	}

	// Process tracks in parallel with a limit on concurrent requests
	sem := make(chan struct{}, s.config.MaxConcurrentRequests)
	errChan := make(chan error, len(tracks))
	var wg sync.WaitGroup

	for _, track := range tracks {
		wg.Add(1)
		go func(t *domain.Track) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			if err := s.EnrichMetadata(ctx, t); err != nil {
				errChan <- fmt.Errorf("failed to process track %s: %w", t.ID, err)
			}
		}(track)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Collect any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch processing failed with %d errors: %v", len(errors), errors)
	}

	s.recordSuccess(time.Since(start))
	return nil
}

// Helper methods

func (s *OpenAIService) recordSuccess(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestCount++
	s.metrics.SuccessCount++
	s.metrics.LastSuccess = time.Now()
	s.metrics.AverageLatency = (s.metrics.AverageLatency*time.Duration(s.metrics.RequestCount-1) + duration) / time.Duration(s.metrics.RequestCount)
}

func (s *OpenAIService) recordFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.metrics.RequestCount++
	s.metrics.FailureCount++
	s.metrics.LastError = err
}
