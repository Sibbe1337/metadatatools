package ai

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"
)

// CompositeAIService implements domain.CompositeAIService
type CompositeAIService struct {
	config           *domain.AIServiceConfig
	qwen2Service     domain.AIService
	openAIService    domain.AIService
	primaryProvider  domain.AIProvider
	fallbackProvider domain.AIProvider
	metrics          map[domain.AIProvider]*domain.AIMetrics
	mu               sync.RWMutex
}

// NewCompositeAIService creates a new composite AI service
func NewCompositeAIService(config *domain.AIServiceConfig) (*CompositeAIService, error) {
	if config == nil {
		return nil, fmt.Errorf("AI service config is required")
	}

	// Create Qwen2-Audio service
	qwen2Service, err := NewQwen2Service(config.Qwen2Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qwen2 service: %w", err)
	}

	// Create OpenAI service
	openAIService, err := NewOpenAIService(config.OpenAIConfig)
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
	}, nil
}

// EnrichMetadata enriches a track with AI-generated metadata
func (s *CompositeAIService) EnrichMetadata(ctx context.Context, track *domain.Track) error {
	start := time.Now()

	// Try primary service first
	err := s.getPrimaryService().EnrichMetadata(ctx, track)
	if err == nil {
		s.recordSuccess(s.primaryProvider, time.Since(start))
		return nil
	}

	// Record failure and try fallback if enabled
	s.recordFailure(s.primaryProvider, err)
	metrics.AIRequestDuration.WithLabelValues(string(s.primaryProvider) + "_failed").Observe(time.Since(start).Seconds())

	if !s.config.EnableFallback {
		return fmt.Errorf("primary service failed and fallback is disabled: %w", err)
	}

	// Try fallback service
	fallbackStart := time.Now()
	err = s.getFallbackService().EnrichMetadata(ctx, track)
	if err != nil {
		s.recordFailure(s.fallbackProvider, err)
		metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_failed").Observe(time.Since(fallbackStart).Seconds())
		return fmt.Errorf("both primary and fallback services failed: %w", err)
	}

	s.recordSuccess(s.fallbackProvider, time.Since(fallbackStart))
	metrics.AIRequestDuration.WithLabelValues(string(s.fallbackProvider) + "_success").Observe(time.Since(fallbackStart).Seconds())
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
