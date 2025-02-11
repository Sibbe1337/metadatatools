package domain

import "context"

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

// AIService defines the interface for AI operations
type AIService interface {
	EnrichMetadata(ctx context.Context, audioPath string) (*AIMetadata, error)
	ValidateMetadata(ctx context.Context, metadata *AIMetadata) (bool, error)
}

// AIMetadata represents the metadata extracted by AI
type AIMetadata struct {
	Title         string   `json:"title"`
	Artist        string   `json:"artist"`
	Album         string   `json:"album"`
	Genre         []string `json:"genre"`
	Year          int      `json:"year"`
	Confidence    float64  `json:"confidence"`
	Language      string   `json:"language"`
	Mood          []string `json:"mood"`
	Tempo         float64  `json:"tempo"`
	Key           string   `json:"key"`
	TimeSignature string   `json:"time_signature"`
	Duration      float64  `json:"duration"`
	Tags          []string `json:"tags"`
}
