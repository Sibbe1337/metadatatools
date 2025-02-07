package domain

import (
	"context"
	"time"
)

// AIService defines the interface for AI-powered metadata enrichment
type AIService interface {
	// EnrichMetadata enriches a track with AI-generated metadata
	EnrichMetadata(ctx context.Context, track *Track) error

	// ValidateMetadata validates track metadata using AI
	ValidateMetadata(ctx context.Context, track *Track) (float64, error)

	// BatchProcess processes multiple tracks in batch
	BatchProcess(ctx context.Context, tracks []*Track) error
}

// AIProvider represents the type of AI service
type AIProvider string

const (
	AIProviderQwen2  AIProvider = "qwen2"
	AIProviderOpenAI AIProvider = "openai"
)

// AIServiceConfig holds configuration for all AI services
type AIServiceConfig struct {
	EnableFallback bool
	Qwen2Config    *Qwen2Config
	OpenAIConfig   *OpenAIConfig
}

// Qwen2Config holds configuration for Qwen2-Audio service
type Qwen2Config struct {
	APIKey                string
	Endpoint              string
	TimeoutSeconds        int
	MinConfidence         float64
	MaxConcurrentRequests int
	RetryAttempts         int
	RetryBackoffSeconds   int
}

// OpenAIConfig holds configuration for OpenAI service
type OpenAIConfig struct {
	APIKey                string
	Endpoint              string
	TimeoutSeconds        int
	MinConfidence         float64
	MaxConcurrentRequests int
	RetryAttempts         int
	RetryBackoffSeconds   int
}

// AIMetadata holds AI-generated metadata for a track
type AIMetadata struct {
	Provider     AIProvider
	Energy       float64
	Danceability float64
	ProcessedAt  time.Time
	ProcessingMs int64
	NeedsReview  bool
	ReviewReason string
}

// AIMetrics holds metrics for an AI service
type AIMetrics struct {
	RequestCount   int64
	SuccessCount   int64
	FailureCount   int64
	LastSuccess    time.Time
	LastError      error
	AverageLatency time.Duration
}

// AIResult represents the result of an AI analysis
type AIResult struct {
	Metadata     *AIMetadata
	Error        error
	RetryCount   int
	Duration     time.Duration
	UsedFallback bool
}

// AIServiceFactory creates AI service instances
type AIServiceFactory interface {
	// CreateService creates an AI service based on the provider
	CreateService(provider AIProvider) (AIService, error)

	// GetDefaultService returns the default AI service
	GetDefaultService() AIService
}

// CompositeAIService defines the interface for managing multiple AI services
type CompositeAIService interface {
	AIService
	// SetPrimaryProvider sets the primary AI provider
	SetPrimaryProvider(provider AIProvider)

	// SetFallbackProvider sets the fallback AI provider
	SetFallbackProvider(provider AIProvider)

	// GetProviderMetrics returns metrics for each provider
	GetProviderMetrics() map[AIProvider]*AIMetrics
}
