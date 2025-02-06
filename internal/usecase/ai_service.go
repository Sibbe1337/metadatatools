// Package usecase implements the business logic layer
package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIService struct {
	client *openai.Client
	cfg    *config.AIConfig
}

// NewOpenAIService creates a new OpenAI service
func NewOpenAIService(cfg *config.AIConfig) (domain.AIService, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(cfg.APIKey)

	return &OpenAIService{
		client: client,
		cfg:    cfg,
	}, nil
}

// EnrichMetadata enriches a track with AI-generated metadata
func (s *OpenAIService) EnrichMetadata(ctx context.Context, track *domain.Track) error {
	// TODO: Implement metadata enrichment using OpenAI
	// This would involve:
	// 1. Creating a prompt describing the track
	// 2. Calling OpenAI API
	// 3. Parsing the response into AIMetadata
	// 4. Updating the track with the new metadata
	return fmt.Errorf("not implemented")
}

// ValidateMetadata validates track metadata using AI
func (s *OpenAIService) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	// TODO: Implement metadata validation using OpenAI
	// This would:
	// 1. Compare existing metadata with AI predictions
	// 2. Return a confidence score
	return 0, fmt.Errorf("not implemented")
}

// BatchProcess processes multiple tracks in batch
func (s *OpenAIService) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
	for i := 0; i < len(tracks); i += s.cfg.BatchSize {
		end := i + s.cfg.BatchSize
		if end > len(tracks) {
			end = len(tracks)
		}

		batch := tracks[i:end]
		for _, track := range batch {
			if err := s.EnrichMetadata(ctx, track); err != nil {
				return fmt.Errorf("failed to process track %s: %w", track.ID, err)
			}
		}
	}
	return nil
}

// calculateConfidence calculates a confidence score based on the AI response
func (s *OpenAIService) calculateConfidence(resp openai.ChatCompletionResponse) float64 {
	// Base confidence on response properties
	var confidence float64 = 1.0

	// Reduce confidence based on various factors
	if len(resp.Choices) > 1 {
		confidence *= 0.9 // Multiple choices indicate uncertainty
	}

	// Check for low-confidence indicators in the response
	content := resp.Choices[0].Message.Content
	if len(content) < 50 {
		confidence *= 0.8 // Very short responses might indicate low quality
	}

	// Ensure confidence is within bounds
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	return confidence
}
