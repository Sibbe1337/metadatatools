// Package usecase implements the business logic layer
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"strings"
	"sync"

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
	// Create a prompt describing the track
	prompt := fmt.Sprintf(`Analyze this track and provide detailed music metadata:
Title: %s
Artist: %s
Album: %s
Duration: %.2f seconds

Please provide the following metadata in a structured format:
1. Genre (specific subgenre if possible)
2. Mood/Emotion
3. Musical key
4. Estimated BPM range
5. 3-5 descriptive tags
6. Similar artists/influences

Respond in a structured JSON format.`, track.Title, track.Artist, track.Album, track.Duration)

	// Call OpenAI API
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.cfg.ModelName,
		Temperature: float32(s.cfg.Temperature),
		MaxTokens:   s.cfg.MaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a music metadata expert. Analyze the track and provide detailed metadata in JSON format.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Calculate confidence score
	confidence := s.calculateConfidence(resp)

	// Parse the response into structured metadata
	var result struct {
		Genre         string   `json:"genre"`
		Mood          string   `json:"mood"`
		Key           string   `json:"key"`
		BPM           float64  `json:"bpm"`
		Tags          []string `json:"tags"`
		SimilarArtist []string `json:"similar_artists"`
	}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Update track with AI-generated metadata
	track.Genre = result.Genre
	track.Mood = result.Mood
	track.Key = result.Key
	track.BPM = result.BPM
	track.AITags = result.Tags
	track.AIConfidence = confidence
	track.ModelVersion = s.cfg.ModelVersion
	track.NeedsReview = confidence < s.cfg.MinConfidence

	// Update metadata struct
	track.Metadata.Labels = []string{result.Genre} // Store genre as a label
	track.Metadata.Mood = result.Mood
	track.Metadata.Key = result.Key
	track.Metadata.BPM = result.BPM
	track.Metadata.AITags = result.Tags
	track.Metadata.Confidence = confidence
	track.Metadata.ModelVersion = s.cfg.ModelVersion

	// Store similar artists in custom fields
	track.Metadata.CustomFields = map[string]string{
		"similar_artists": strings.Join(result.SimilarArtist, ","),
	}

	return nil
}

// ValidateMetadata validates track metadata using AI
func (s *OpenAIService) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	// Create a prompt for validation
	prompt := fmt.Sprintf(`Validate the following music metadata for accuracy and consistency:

Track Information:
- Title: %s
- Artist: %s
- Album: %s
- Genre: %s
- Mood: %s
- Key: %s
- BPM: %.2f
- Tags: %s

Please analyze the metadata and:
1. Check if the genre matches the artist's typical style
2. Verify if the mood aligns with the genre
3. Validate if the musical key is in a standard format
4. Confirm if the BPM is within a reasonable range for the genre
5. Assess if the tags are relevant and accurate

Respond with a JSON object containing:
1. A validation score (0-1) for each field
2. An overall confidence score
3. Any inconsistencies or potential errors found`,
		track.Title, track.Artist, track.Album, track.Genre, track.Mood,
		track.Key, track.BPM, strings.Join(track.AITags, ", "))

	// Call OpenAI API
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.cfg.ModelName,
		Temperature: float32(s.cfg.Temperature),
		MaxTokens:   s.cfg.MaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a music metadata validation expert. Analyze the metadata for accuracy and consistency.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Parse the validation response
	var result struct {
		Scores struct {
			Genre   float64 `json:"genre"`
			Mood    float64 `json:"mood"`
			Key     float64 `json:"key"`
			BPM     float64 `json:"bpm"`
			Tags    float64 `json:"tags"`
			Overall float64 `json:"overall"`
		} `json:"scores"`
		Issues []string `json:"issues"`
	}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return 0, fmt.Errorf("failed to parse validation response: %w", err)
	}

	// Calculate final confidence score
	confidence := result.Scores.Overall

	// Store validation issues in custom fields if any
	if len(result.Issues) > 0 {
		if track.Metadata.CustomFields == nil {
			track.Metadata.CustomFields = make(map[string]string)
		}
		track.Metadata.CustomFields["validation_issues"] = strings.Join(result.Issues, "; ")
	}

	// Update track's review status
	track.NeedsReview = confidence < s.cfg.MinConfidence || len(result.Issues) > 0

	return confidence, nil
}

// BatchProcess processes multiple tracks in batch
func (s *OpenAIService) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
	// Create error channel to collect errors from goroutines
	errChan := make(chan error, len(tracks))

	// Create semaphore to limit concurrent API calls
	sem := make(chan struct{}, s.cfg.BatchSize)

	// Create wait group to wait for all goroutines
	var wg sync.WaitGroup

	// Process tracks concurrently
	for i := range tracks {
		wg.Add(1)
		go func(track *domain.Track) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check context cancellation
			if ctx.Err() != nil {
				errChan <- ctx.Err()
				return
			}

			// Enrich metadata
			if err := s.EnrichMetadata(ctx, track); err != nil {
				errChan <- fmt.Errorf("failed to enrich track %s: %w", track.ID, err)
				return
			}

			// Validate metadata
			confidence, err := s.ValidateMetadata(ctx, track)
			if err != nil {
				errChan <- fmt.Errorf("failed to validate track %s: %w", track.ID, err)
				return
			}

			// Mark for review if confidence is low
			track.NeedsReview = confidence < s.cfg.MinConfidence

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

	// Return combined error if any occurred
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
