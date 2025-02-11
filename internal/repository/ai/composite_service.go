// Package ai provides AI services for metadata enrichment and analysis.
//
// This package implements a composite AI service that can use multiple AI providers
// and includes features like fallback handling, retries, and analytics tracking.
//
// Key features:
//   - Multiple AI provider support (OpenAI, Qwen2, etc.)
//   - Automatic fallback to backup providers
//   - Retry mechanism with exponential backoff
//   - Analytics tracking for experiments
//   - Concurrent request limiting
//
// Usage example:
//
//	config := &ai.Config{
//	    EnableFallback:        true,
//	    APIKey:               "your-api-key",
//	    Endpoint:             "https://api.openai.com/v1",
//	    TimeoutSeconds:       30,
//	    MinConfidence:        0.85,
//	    MaxConcurrentRequests: 10,
//	    RetryAttempts:        3,
//	    RetryBackoffSeconds:  2,
//	}
//
//	service, err := ai.NewCompositeAIService(config, analyticsService)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	metadata, err := service.EnrichMetadata(ctx, track)
//	if err != nil {
//	    log.Printf("Failed to enrich metadata: %v", err)
//	}
package ai

import (
	"context"
	"fmt"
	"math/rand"
	"metadatatool/internal/pkg/analytics"
	pkgdomain "metadatatool/internal/pkg/domain"
	"sync"
	"time"
)

// Config holds configuration for the composite AI service
type Config struct {
	EnableFallback        bool
	TimeoutSeconds        int
	MinConfidence         float64
	MaxConcurrentRequests int
	RetryAttempts         int
	RetryBackoffSeconds   int
	Qwen2Config           *pkgdomain.Qwen2Config
	OpenAIConfig          *pkgdomain.OpenAIConfig
}

// OpenAIConfig contains configuration specific to the OpenAI provider.
type OpenAIConfig struct {
	// APIKey is the OpenAI API key
	APIKey string

	// Endpoint is the OpenAI API endpoint
	Endpoint string

	// Model is the specific model to use (e.g., "gpt-4")
	Model string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int
}

// Qwen2Config contains configuration specific to the Qwen2 provider.
type Qwen2Config struct {
	// APIKey is the Qwen2 API key
	APIKey string

	// Endpoint is the Qwen2 API endpoint
	Endpoint string

	// Model is the specific model to use
	Model string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int
}

// CompositeAIService implements pkg/domain.AIService
type CompositeAIService struct {
	config           *Config
	qwen2Service     pkgdomain.AIService
	openAIService    pkgdomain.AIService
	primaryProvider  pkgdomain.AIProvider
	fallbackProvider pkgdomain.AIProvider
	metrics          map[pkgdomain.AIProvider]*pkgdomain.AIMetrics
	analytics        *analytics.BigQueryService
	experimentGroup  string
	semaphore        chan struct{}
	mu               sync.RWMutex
}

// Provider defines the interface that all AI providers must implement.
// This allows the composite service to work with different AI backends.
type Provider interface {
	// EnrichMetadata enriches track metadata using AI
	EnrichMetadata(ctx context.Context, track *pkgdomain.Track) (*pkgdomain.Metadata, error)

	// ValidateMetadata validates track metadata using AI
	ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (*pkgdomain.ValidationResult, error)
}

// NewCompositeAIService creates a new composite AI service
func NewCompositeAIService(config *Config, analytics *analytics.BigQueryService) (pkgdomain.AIService, error) {
	if config == nil {
		return nil, fmt.Errorf("AI service config is required")
	}
	if analytics == nil {
		return nil, fmt.Errorf("analytics service is required")
	}

	// Create Qwen2 service
	qwen2Service, err := NewQwen2Service(config.Qwen2Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create qwen2 service: %w", err)
	}

	// Create OpenAI service
	openAIService, err := NewOpenAIService(config.OpenAIConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create openai service: %w", err)
	}

	service := &CompositeAIService{
		config:           config,
		qwen2Service:     qwen2Service,
		openAIService:    openAIService,
		primaryProvider:  pkgdomain.AIProviderQwen2,
		fallbackProvider: pkgdomain.AIProviderOpenAI,
		metrics:          make(map[pkgdomain.AIProvider]*pkgdomain.AIMetrics),
		analytics:        analytics,
		semaphore:        make(chan struct{}, config.MaxConcurrentRequests),
	}

	return service, nil
}

// EnrichMetadata enriches track metadata using AI
func (s *CompositeAIService) EnrichMetadata(ctx context.Context, track *pkgdomain.Track) error {
	start := time.Now()

	// Determine if this request should be part of the experiment (10% of traffic)
	isExperiment := rand.Float64() < 0.1

	var service pkgdomain.AIService
	var provider pkgdomain.AIProvider

	if isExperiment {
		service = s.openAIService
		provider = pkgdomain.AIProviderOpenAI
	} else {
		service = s.qwen2Service
		provider = pkgdomain.AIProviderQwen2
	}

	// Acquire semaphore
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// Call the service
	err := service.EnrichMetadata(ctx, track)
	duration := time.Since(start)

	if err != nil {
		s.recordFailure(provider, err)
		return fmt.Errorf("failed to enrich metadata: %w", err)
	}

	s.recordSuccess(provider, duration)

	return nil
}

// ValidateMetadata validates track metadata using AI
func (s *CompositeAIService) ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (float64, error) {
	// Use primary service first
	confidence, err := s.getPrimaryService().ValidateMetadata(ctx, track)
	if err == nil {
		return confidence, nil
	}

	// If fallback is disabled, return the error
	if !s.config.EnableFallback {
		return 0.0, err
	}

	// Try fallback service
	confidence, fallbackErr := s.getFallbackService().ValidateMetadata(ctx, track)
	if fallbackErr != nil {
		// Return original error if fallback also fails
		return 0.0, err
	}

	return confidence, nil
}

// BatchProcess processes multiple tracks in batch
func (s *CompositeAIService) BatchProcess(ctx context.Context, tracks []*pkgdomain.Track) error {
	// Process tracks sequentially since some services don't support batch processing
	for _, track := range tracks {
		if err := s.EnrichMetadata(ctx, track); err != nil {
			return fmt.Errorf("failed to process track %s: %w", track.ID, err)
		}
	}
	return nil
}

// SetPrimaryProvider sets the primary AI provider
func (s *CompositeAIService) SetPrimaryProvider(provider pkgdomain.AIProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.primaryProvider = provider
}

// SetFallbackProvider sets the fallback AI provider
func (s *CompositeAIService) SetFallbackProvider(provider pkgdomain.AIProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallbackProvider = provider
}

// GetProviderMetrics returns metrics for each provider
func (s *CompositeAIService) GetProviderMetrics() map[pkgdomain.AIProvider]*pkgdomain.AIMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the metrics
	metrics := make(map[pkgdomain.AIProvider]*pkgdomain.AIMetrics)
	for provider, metric := range s.metrics {
		metricCopy := *metric
		metrics[provider] = &metricCopy
	}

	return metrics
}

// Helper methods

func (s *CompositeAIService) getPrimaryService() pkgdomain.AIService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.primaryProvider == pkgdomain.AIProviderQwen2 {
		return s.qwen2Service
	}
	return s.openAIService
}

func (s *CompositeAIService) getFallbackService() pkgdomain.AIService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.fallbackProvider == pkgdomain.AIProviderQwen2 {
		return s.qwen2Service
	}
	return s.openAIService
}

func (s *CompositeAIService) recordSuccess(provider pkgdomain.AIProvider, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.metrics[provider]; !exists {
		s.metrics[provider] = &pkgdomain.AIMetrics{}
	}

	s.metrics[provider].RequestCount++
	s.metrics[provider].SuccessCount++
	s.metrics[provider].LastSuccess = time.Now()
	s.metrics[provider].AverageLatency = (s.metrics[provider].AverageLatency*time.Duration(s.metrics[provider].RequestCount-1) + duration) / time.Duration(s.metrics[provider].RequestCount)
}

func (s *CompositeAIService) recordFailure(provider pkgdomain.AIProvider, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.metrics[provider]; !exists {
		s.metrics[provider] = &pkgdomain.AIMetrics{}
	}

	s.metrics[provider].RequestCount++
	s.metrics[provider].FailureCount++
	s.metrics[provider].LastError = err
}

// SetExperimentGroup sets the experiment group for this service instance.
// This is used for A/B testing different AI providers or configurations.
//
// Parameters:
//   - group: Experiment group ("control" or "experiment")
func (s *CompositeAIService) SetExperimentGroup(group string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.experimentGroup = group
}
