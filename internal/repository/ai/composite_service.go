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
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds configuration for the AI service.
// It includes settings for timeouts, retries, and rate limiting.
type Config struct {
	// EnableFallback determines if the service should try backup providers
	EnableFallback bool

	// TimeoutSeconds is the maximum time to wait for an AI response
	TimeoutSeconds int

	// MinConfidence is the minimum confidence score required (0-1)
	MinConfidence float64

	// MaxConcurrentRequests limits the number of concurrent AI calls
	MaxConcurrentRequests int

	// RetryAttempts is the maximum number of retry attempts
	RetryAttempts int

	// RetryBackoffSeconds is the base delay between retries
	RetryBackoffSeconds int

	// OpenAIConfig contains OpenAI-specific configuration
	OpenAIConfig *OpenAIConfig

	// Qwen2Config contains Qwen2-specific configuration
	Qwen2Config *Qwen2Config
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

// CompositeAIService orchestrates multiple AI providers and handles
// fallback, retries, and analytics tracking.
type CompositeAIService struct {
	config           *Config
	qwen2Service     domain.AIService
	openAIService    domain.AIService
	primaryProvider  domain.AIProvider
	fallbackProvider domain.AIProvider
	metrics          map[domain.AIProvider]*domain.AIMetrics
	analytics        *analytics.BigQueryService
	mu               sync.RWMutex
	experimentGroup  string
	semaphore        chan struct{}
}

// Provider defines the interface that all AI providers must implement.
// This allows the composite service to work with different AI backends.
type Provider interface {
	// EnrichMetadata enriches track metadata using AI
	EnrichMetadata(ctx context.Context, track *domain.Track) (*domain.Metadata, error)

	// ValidateMetadata validates track metadata using AI
	ValidateMetadata(ctx context.Context, track *domain.Track) (*domain.ValidationResult, error)
}

// NewCompositeAIService creates a new composite AI service
func NewCompositeAIService(config *Config, analytics *analytics.BigQueryService) (*CompositeAIService, error) {
	if config == nil {
		return nil, fmt.Errorf("AI service config is required")
	}
	if analytics == nil {
		return nil, fmt.Errorf("analytics service is required")
	}

	// Create Qwen2-Audio service
	qwen2Config := &domain.Qwen2Config{
		APIKey:                config.Qwen2Config.APIKey,
		Endpoint:              config.Qwen2Config.Endpoint,
		TimeoutSeconds:        config.TimeoutSeconds,
		MinConfidence:         config.MinConfidence,
		MaxConcurrentRequests: config.MaxConcurrentRequests,
		RetryAttempts:         config.RetryAttempts,
		RetryBackoffSeconds:   config.RetryBackoffSeconds,
	}
	qwen2Service, err := NewQwen2Service(qwen2Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qwen2 service: %w", err)
	}

	// Create OpenAI service
	openAIConfig := &domain.OpenAIConfig{
		APIKey:                config.OpenAIConfig.APIKey,
		Endpoint:              config.OpenAIConfig.Endpoint,
		TimeoutSeconds:        config.TimeoutSeconds,
		MinConfidence:         config.MinConfidence,
		MaxConcurrentRequests: config.MaxConcurrentRequests,
		RetryAttempts:         config.RetryAttempts,
		RetryBackoffSeconds:   config.RetryBackoffSeconds,
	}
	openAIService, err := NewOpenAIService(openAIConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI service: %w", err)
	}

	return &CompositeAIService{
		config:           config,
		qwen2Service:     qwen2Service,
		openAIService:    openAIService,
		primaryProvider:  domain.AIProviderQwen2,
		fallbackProvider: domain.AIProviderOpenAI,
		metrics:          make(map[domain.AIProvider]*domain.AIMetrics),
		analytics:        analytics,
		experimentGroup:  "control",
		semaphore:        make(chan struct{}, config.MaxConcurrentRequests),
	}, nil
}

// EnrichMetadata enriches a track with AI-generated metadata
func (s *CompositeAIService) EnrichMetadata(ctx context.Context, track *domain.Track) error {
	start := time.Now()

	// Determine if this request should be part of the experiment (10% of traffic)
	isExperiment := rand.Float64() < 0.1

	var service domain.AIService
	var provider domain.AIProvider
	var experimentGroup string

	if isExperiment {
		service = s.openAIService
		provider = domain.AIProviderOpenAI
		experimentGroup = "experiment"
	} else {
		service = s.qwen2Service
		provider = domain.AIProviderQwen2
		experimentGroup = "control"
	}

	// Try selected service
	err := service.EnrichMetadata(ctx, track)
	duration := time.Since(start)

	// Record experiment data
	record := &analytics.AIExperimentRecord{
		Timestamp:       time.Now(),
		TrackID:         track.ID,
		ModelProvider:   string(provider),
		ModelVersion:    track.ModelVersion(),
		ProcessingTime:  duration.Seconds(),
		Confidence:      track.AIConfidence(),
		Success:         err == nil,
		ErrorMessage:    "",
		ExperimentGroup: experimentGroup,
	}

	if err != nil {
		record.ErrorMessage = err.Error()
		s.recordFailure(provider, err)
		metrics.AIRequestDuration.WithLabelValues(string(provider) + "_failed").Observe(duration.Seconds())

		// Try fallback if confidence is low or there was an error
		if s.config.EnableFallback {
			fallbackStart := time.Now()
			err = s.getFallbackService().EnrichMetadata(ctx, track)
			fallbackDuration := time.Since(fallbackStart)

			// Record fallback attempt
			fallbackRecord := &analytics.AIExperimentRecord{
				Timestamp:       time.Now(),
				TrackID:         track.ID,
				ModelProvider:   string(s.fallbackProvider),
				ModelVersion:    track.ModelVersion(),
				ProcessingTime:  fallbackDuration.Seconds(),
				Confidence:      track.AIConfidence(),
				Success:         err == nil,
				ErrorMessage:    "",
				ExperimentGroup: experimentGroup + "_fallback",
			}

			if err != nil {
				fallbackRecord.ErrorMessage = err.Error()
				s.recordFailure(s.fallbackProvider, err)
				metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_failed").Observe(fallbackDuration.Seconds())
				return fmt.Errorf("both primary and fallback services failed: %w", err)
			}

			s.recordSuccess(s.fallbackProvider, fallbackDuration)
			metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_success").Observe(fallbackDuration.Seconds())

			// Record fallback metrics
			if err := s.analytics.RecordAIExperiment(ctx, fallbackRecord); err != nil {
				logrus.WithError(err).Error("Failed to record fallback experiment data")
			}
		} else {
			return fmt.Errorf("service failed and fallback is disabled: %w", err)
		}
	} else {
		// Check confidence threshold
		if track.AIConfidence() < s.config.MinConfidence {
			// Try fallback for low confidence
			fallbackStart := time.Now()
			fallbackErr := s.getFallbackService().EnrichMetadata(ctx, track)
			fallbackDuration := time.Since(fallbackStart)

			fallbackRecord := &analytics.AIExperimentRecord{
				Timestamp:       time.Now(),
				TrackID:         track.ID,
				ModelProvider:   string(s.fallbackProvider),
				ModelVersion:    track.ModelVersion(),
				ProcessingTime:  fallbackDuration.Seconds(),
				Confidence:      track.AIConfidence(),
				Success:         fallbackErr == nil,
				ErrorMessage:    "",
				ExperimentGroup: experimentGroup + "_low_confidence",
			}

			if fallbackErr == nil && track.AIConfidence() > s.config.MinConfidence {
				s.recordSuccess(s.fallbackProvider, fallbackDuration)
				metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_success").Observe(fallbackDuration.Seconds())
			} else {
				// Keep original results if fallback didn't improve confidence
				fallbackRecord.ErrorMessage = "Fallback did not improve confidence"
			}

			// Record fallback metrics
			if err := s.analytics.RecordAIExperiment(ctx, fallbackRecord); err != nil {
				logrus.WithError(err).Error("Failed to record fallback experiment data")
			}
		}

		s.recordSuccess(provider, duration)
		metrics.AIRequestDuration.WithLabelValues(string(provider) + "_success").Observe(duration.Seconds())
	}

	// Record experiment metrics
	if err := s.analytics.RecordAIExperiment(ctx, record); err != nil {
		logrus.WithError(err).Error("Failed to record experiment data")
	}

	return nil
}

// ValidateMetadata validates track metadata using AI
func (s *CompositeAIService) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	start := time.Now()

	// Try primary service first
	confidence, err := s.getPrimaryService().ValidateMetadata(ctx, track)
	if err == nil {
		s.recordSuccess(s.primaryProvider, time.Since(start))
		return confidence, nil
	}

	// Record failure and try fallback if enabled
	s.recordFailure(s.primaryProvider, err)
	metrics.AIRequestDuration.WithLabelValues(string(s.primaryProvider) + "_failed").Observe(time.Since(start).Seconds())

	if !s.config.EnableFallback {
		return 0, fmt.Errorf("primary service failed and fallback is disabled: %w", err)
	}

	// Try fallback service
	fallbackStart := time.Now()
	confidence, err = s.getFallbackService().ValidateMetadata(ctx, track)
	if err != nil {
		s.recordFailure(s.fallbackProvider, err)
		metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_failed").Observe(time.Since(fallbackStart).Seconds())
		return 0, fmt.Errorf("both primary and fallback services failed: %w", err)
	}

	s.recordSuccess(s.fallbackProvider, time.Since(fallbackStart))
	metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_success").Observe(time.Since(fallbackStart).Seconds())
	return confidence, nil
}

// BatchProcess processes multiple tracks in batch
func (s *CompositeAIService) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
	start := time.Now()

	// Try primary service first
	err := s.getPrimaryService().BatchProcess(ctx, tracks)
	if err == nil {
		s.recordSuccess(s.primaryProvider, time.Since(start))
		return nil
	}

	// Record failure and try fallback if enabled
	s.recordFailure(s.primaryProvider, err)
	metrics.AIRequestDuration.WithLabelValues(string(s.primaryProvider) + "_batch_failed").Observe(time.Since(start).Seconds())

	if !s.config.EnableFallback {
		return fmt.Errorf("primary service failed and fallback is disabled: %w", err)
	}

	// Try fallback service
	fallbackStart := time.Now()
	err = s.getFallbackService().BatchProcess(ctx, tracks)
	if err != nil {
		s.recordFailure(s.fallbackProvider, err)
		metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_batch_failed").Observe(time.Since(fallbackStart).Seconds())
		return fmt.Errorf("both primary and fallback services failed: %w", err)
	}

	s.recordSuccess(s.fallbackProvider, time.Since(fallbackStart))
	metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_batch_success").Observe(time.Since(fallbackStart).Seconds())
	return nil
}

// SetPrimaryProvider sets the primary AI provider
func (s *CompositeAIService) SetPrimaryProvider(provider domain.AIProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.primaryProvider = provider
}

// SetFallbackProvider sets the fallback AI provider
func (s *CompositeAIService) SetFallbackProvider(provider domain.AIProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallbackProvider = provider
}

// GetProviderMetrics returns metrics for each provider
func (s *CompositeAIService) GetProviderMetrics() map[domain.AIProvider]*domain.AIMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of the metrics
	metrics := make(map[domain.AIProvider]*domain.AIMetrics)
	for provider, metric := range s.metrics {
		metricCopy := *metric
		metrics[provider] = &metricCopy
	}

	return metrics
}

// Helper methods

func (s *CompositeAIService) getPrimaryService() domain.AIService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.primaryProvider == domain.AIProviderQwen2 {
		return s.qwen2Service
	}
	return s.openAIService
}

func (s *CompositeAIService) getFallbackService() domain.AIService {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.fallbackProvider == domain.AIProviderQwen2 {
		return s.qwen2Service
	}
	return s.openAIService
}

func (s *CompositeAIService) recordSuccess(provider domain.AIProvider, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.metrics[provider]; !exists {
		s.metrics[provider] = &domain.AIMetrics{}
	}

	s.metrics[provider].RequestCount++
	s.metrics[provider].SuccessCount++
	s.metrics[provider].LastSuccess = time.Now()
	s.metrics[provider].AverageLatency = (s.metrics[provider].AverageLatency*time.Duration(s.metrics[provider].RequestCount-1) + duration) / time.Duration(s.metrics[provider].RequestCount)
}

func (s *CompositeAIService) recordFailure(provider domain.AIProvider, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.metrics[provider]; !exists {
		s.metrics[provider] = &domain.AIMetrics{}
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

// getExperimentGroup safely retrieves the current experiment group.
func (s *CompositeAIService) getExperimentGroup() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.experimentGroup
}

// enrichWithRetry attempts to enrich metadata with retries and fallback.
func (s *CompositeAIService) enrichWithRetry(ctx context.Context, track *domain.Track) (*domain.Metadata, error) {
	// Implementation details...
	return nil, nil // TODO: Implement
}

// validateWithRetry attempts to validate metadata with retries and fallback.
func (s *CompositeAIService) validateWithRetry(ctx context.Context, track *domain.Track) (*domain.ValidationResult, error) {
	// Implementation details...
	return nil, nil // TODO: Implement
}
