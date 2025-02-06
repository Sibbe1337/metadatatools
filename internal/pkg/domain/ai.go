package domain

import (
	"context"
)

// AIService defines the interface for AI-based metadata enrichment
type AIService interface {
	// EnrichMetadata enriches a track with AI-generated metadata
	EnrichMetadata(ctx context.Context, track *Track) error

	// ValidateMetadata validates track metadata using AI
	ValidateMetadata(ctx context.Context, track *Track) (float64, error)

	// BatchProcess processes multiple tracks in batch
	BatchProcess(ctx context.Context, tracks []*Track) error
}

// AIMetadata represents AI-generated metadata
type AIMetadata struct {
	Genre        string   `json:"genre"`
	Mood         string   `json:"mood"`
	BPM          float64  `json:"bpm"`
	Key          string   `json:"key"`
	Tags         []string `json:"tags"`
	Confidence   float64  `json:"confidence"`
	ModelVersion string   `json:"model_version"`
}

// AIConfig holds configuration for AI services
type AIConfig struct {
	ModelName     string  `json:"model_name"`
	ModelVersion  string  `json:"model_version"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
	BatchSize     int     `json:"batch_size"`
	MinConfidence float64 `json:"min_confidence"`
}
