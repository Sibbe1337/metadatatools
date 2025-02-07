// Package repository implements the data access layer
package repository

import (
	"context"
	"fmt"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"strings"
	"sync"
	"time"

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
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.AIRequestDuration.WithLabelValues("enrich").Observe(duration)
	}()

	// Create prompt for OpenAI
	prompt := fmt.Sprintf(`Analyze the following track metadata and provide enriched information:
Title: %s
Artist: %s
Album: %s
Genre: %s
Year: %d

Please provide the following in a structured format:
1. Genre classification (including subgenres)
2. Mood/emotional content
3. Musical key and BPM estimation
4. Similar artists/influences
5. Cultural context and era
6. Production style characteristics
7. Confidence score (0.0-1.0) for these predictions`,
		track.Title, track.Artist, track.Album, track.Genre, track.Year)

	// Call OpenAI API
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.cfg.ModelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a music metadata expert. Analyze the track information and provide detailed metadata enrichment.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature:      float32(s.cfg.Temperature),
		MaxTokens:        s.cfg.MaxTokens,
		TopP:             0.9,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
	})
	if err != nil {
		metrics.AIErrorTotal.WithLabelValues("openai", "api_error").Inc()
		return fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Parse response and update track metadata
	content := resp.Choices[0].Message.Content
	confidence := s.calculateConfidence(resp)

	// Record confidence score
	metrics.AIConfidenceScore.WithLabelValues("openai").Observe(confidence)

	// Update track with AI metadata
	track.AIMetadata = &domain.AIMetadata{
		Provider:     domain.AIProviderOpenAI,
		ProcessedAt:  time.Now(),
		ProcessingMs: time.Since(start).Milliseconds(),
		NeedsReview:  confidence < s.cfg.MinConfidence,
		ReviewReason: s.getReviewReason(confidence),
	}

	// Parse and update specific fields
	// Note: In a production environment, you'd want more robust parsing
	if confidence >= s.cfg.MinConfidence {
		// Extract and update relevant fields
		// This is a simplified example - you'd want more robust parsing in production
		if strings.Contains(content, "BPM:") {
			if bpm, err := extractBPM(content); err == nil {
				track.BPM = bpm
			}
		}
		if strings.Contains(content, "Key:") {
			if key, err := extractKey(content); err == nil {
				track.Key = key
			}
		}
		if strings.Contains(content, "Mood:") {
			if mood, err := extractMood(content); err == nil {
				track.Mood = mood
			}
		}
		// Extract genre if confidence is high
		if strings.Contains(content, "Genre:") {
			if genre, err := extractGenre(content); err == nil && track.Genre == "" {
				track.Genre = genre
			}
		}
	}

	metrics.AIRequestTotal.WithLabelValues("openai", "success").Inc()
	return nil
}

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

	// Check for uncertainty language
	uncertaintyPhrases := []string{"might be", "could be", "possibly", "uncertain", "unclear"}
	for _, phrase := range uncertaintyPhrases {
		if strings.Contains(strings.ToLower(content), phrase) {
			confidence *= 0.95
		}
	}

	return confidence
}

func (s *OpenAIService) getReviewReason(confidence float64) string {
	if confidence < s.cfg.MinConfidence {
		return fmt.Sprintf("Low confidence score: %.2f", confidence)
	}
	return ""
}

// Helper functions for extracting specific metadata fields
func extractBPM(content string) (float64, error) {
	// Implement BPM extraction logic
	return 0, fmt.Errorf("not implemented")
}

func extractKey(content string) (string, error) {
	// Implement musical key extraction logic
	return "", fmt.Errorf("not implemented")
}

func extractMood(content string) (string, error) {
	// Implement mood extraction logic
	return "", fmt.Errorf("not implemented")
}

func extractGenre(content string) (string, error) {
	// Implement genre extraction logic
	return "", fmt.Errorf("not implemented")
}

// ValidateMetadata validates track metadata using AI
func (s *OpenAIService) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.AIRequestDuration.WithLabelValues("validate").Observe(duration)
	}()

	// Create prompt for OpenAI
	prompt := fmt.Sprintf(`Validate the following track metadata for accuracy and consistency:
Title: %s
Artist: %s
Album: %s
Genre: %s
Year: %d
BPM: %f
Key: %s
Mood: %s

Please analyze this metadata and:
1. Check for inconsistencies or errors
2. Verify genre classification accuracy
3. Validate BPM and musical key
4. Check mood classification
5. Provide a confidence score (0.0-1.0) for the metadata accuracy
6. List any specific issues found`,
		track.Title, track.Artist, track.Album, track.Genre, track.Year, track.BPM, track.Key, track.Mood)

	// Call OpenAI API
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: s.cfg.ModelName,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "You are a music metadata validation expert. Analyze the track information for accuracy and consistency.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature:      float32(s.cfg.Temperature),
		MaxTokens:        s.cfg.MaxTokens,
		TopP:             0.9,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
	})
	if err != nil {
		metrics.AIErrorTotal.WithLabelValues("openai", "api_error").Inc()
		return 0, fmt.Errorf("OpenAI API validation failed: %w", err)
	}

	// Calculate confidence from response
	confidence := s.calculateConfidence(resp)
	metrics.AIConfidenceScore.WithLabelValues("openai").Observe(confidence)

	// Update track's metadata with validation results
	if track.AIMetadata == nil {
		track.AIMetadata = &domain.AIMetadata{
			Provider: domain.AIProviderOpenAI,
		}
	}
	track.AIMetadata.ProcessedAt = time.Now()
	track.AIMetadata.ProcessingMs = time.Since(start).Milliseconds()
	track.AIMetadata.NeedsReview = confidence < s.cfg.MinConfidence
	track.AIMetadata.ReviewReason = s.getReviewReason(confidence)

	metrics.AIRequestTotal.WithLabelValues("openai", "success").Inc()
	return confidence, nil
}

// extractValidationIssues parses the AI response to extract validation issues
func extractValidationIssues(content string) []string {
	var issues []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines that indicate issues
		if strings.Contains(strings.ToLower(line), "issue") ||
			strings.Contains(strings.ToLower(line), "error") ||
			strings.Contains(strings.ToLower(line), "incorrect") ||
			strings.Contains(strings.ToLower(line), "invalid") ||
			strings.Contains(strings.ToLower(line), "inconsistent") {
			issues = append(issues, line)
		}
	}

	return issues
}

// BatchProcess processes multiple tracks in batch
func (s *OpenAIService) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	errChan := make(chan error, len(tracks))
	var wg sync.WaitGroup

	for i := range tracks {
		wg.Add(1)
		go func(track *domain.Track) {
			defer wg.Done()

			if err := s.EnrichMetadata(ctx, track); err != nil {
				errChan <- fmt.Errorf("failed to enrich track %s: %w", track.ID, err)
				return
			}

			confidence, err := s.ValidateMetadata(ctx, track)
			if err != nil {
				errChan <- fmt.Errorf("failed to validate track %s: %w", track.ID, err)
				return
			}

			track.AIMetadata.NeedsReview = confidence < s.cfg.MinConfidence
		}(tracks[i])
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString("batch processing encountered errors:\n")
		for _, err := range errors {
			errMsg.WriteString("- " + err.Error() + "\n")
		}
		return fmt.Errorf(errMsg.String())
	}

	return nil
}
