// Package repository implements the data access layer
package repository

import (
	"context"
	"fmt"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"metadatatool/internal/pkg/retry"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"
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
		track.Metadata.BasicTrackMetadata.Title,
		track.Metadata.BasicTrackMetadata.Artist,
		track.Metadata.BasicTrackMetadata.Album,
		track.Metadata.Musical.Genre,
		track.Metadata.BasicTrackMetadata.Year)

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
	track.Metadata.AI = &domain.TrackAIMetadata{
		Model:        s.cfg.ModelName,
		Version:      s.cfg.ModelVersion,
		ProcessedAt:  time.Now(),
		NeedsReview:  confidence < s.cfg.MinConfidence,
		ReviewReason: s.getReviewReason(confidence),
		Confidence:   confidence,
	}

	// Update musical metadata based on content analysis
	if strings.Contains(content, "BPM:") {
		if bpm, err := s.extractBPM(content); err == nil {
			track.Metadata.Musical.BPM = bpm
		}
	}
	if strings.Contains(content, "Key:") {
		if key, err := s.extractKey(content); err == nil {
			track.Metadata.Musical.Key = key
		}
	}
	if strings.Contains(content, "Mood:") {
		if mood, err := s.extractMood(content); err == nil {
			track.Metadata.Musical.Mood = mood
		}
	}
	if strings.Contains(content, "Genre:") {
		if genre, err := s.extractGenre(content); err == nil {
			track.Metadata.Musical.Genre = genre
		}
	}

	// Store similar artists in custom fields
	if similarArtists, err := s.extractSimilarArtists(content); err == nil && len(similarArtists) > 0 {
		track.Metadata.Additional.CustomTags["similar_artists"] = strings.Join(similarArtists, ",")
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
func (s *OpenAIService) extractBPM(content string) (float64, error) {
	// Look for BPM in the content
	if strings.Contains(content, "BPM:") {
		// Find the line containing BPM
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "BPM:") {
				// Try to parse the BPM value
				var bpm float64
				_, err := fmt.Sscanf(line, "BPM: %f", &bpm)
				if err == nil && bpm > 0 {
					return bpm, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("could not extract BPM from content")
}

func (s *OpenAIService) extractKey(content string) (string, error) {
	// Look for musical key in the content
	if strings.Contains(content, "Key:") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Key:") {
				// Extract everything after "Key:"
				parts := strings.SplitN(line, "Key:", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[1])
					if key != "" {
						return key, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("could not extract musical key from content")
}

func (s *OpenAIService) extractMood(content string) (string, error) {
	// Look for mood in the content
	if strings.Contains(content, "Mood:") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Mood:") {
				// Extract everything after "Mood:"
				parts := strings.SplitN(line, "Mood:", 2)
				if len(parts) == 2 {
					mood := strings.TrimSpace(parts[1])
					if mood != "" {
						return mood, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("could not extract mood from content")
}

func (s *OpenAIService) extractGenre(content string) (string, error) {
	// Look for genre in the content
	if strings.Contains(content, "Genre:") {
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Genre:") {
				// Extract everything after "Genre:"
				parts := strings.SplitN(line, "Genre:", 2)
				if len(parts) == 2 {
					genre := strings.TrimSpace(parts[1])
					if genre != "" {
						return genre, nil
					}
				}
			}
		}
	}
	return "", fmt.Errorf("could not extract genre from content")
}

func (s *OpenAIService) extractSimilarArtists(content string) ([]string, error) {
	// Look for similar artists section in the content
	if strings.Contains(content, "Similar artists:") {
		lines := strings.Split(content, "\n")
		var artists []string
		inSimilarArtistsSection := false

		for _, line := range lines {
			if strings.Contains(line, "Similar artists:") {
				inSimilarArtistsSection = true
				continue
			}
			if inSimilarArtistsSection {
				// Stop if we hit another section
				if strings.Contains(line, ":") {
					break
				}
				// Add non-empty lines as artists
				artist := strings.TrimSpace(line)
				if artist != "" {
					artists = append(artists, artist)
				}
			}
		}

		if len(artists) > 0 {
			return artists, nil
		}
	}
	return nil, fmt.Errorf("could not extract similar artists from content")
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
		track.Metadata.BasicTrackMetadata.Title,
		track.Metadata.BasicTrackMetadata.Artist,
		track.Metadata.BasicTrackMetadata.Album,
		track.Metadata.Musical.Genre,
		track.Metadata.BasicTrackMetadata.Year,
		track.Metadata.Musical.BPM,
		track.Metadata.Musical.Key,
		track.Metadata.Musical.Mood)

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
	if track.Metadata.AI == nil {
		track.Metadata.AI = &domain.TrackAIMetadata{
			Model:   s.cfg.ModelName,
			Version: s.cfg.ModelVersion,
		}
	}
	track.Metadata.AI.ProcessedAt = time.Now()
	track.Metadata.AI.NeedsReview = confidence < s.cfg.MinConfidence
	track.Metadata.AI.ReviewReason = s.getReviewReason(confidence)
	track.Metadata.AI.Confidence = confidence

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

// BatchProcess processes multiple tracks in batch with retry logic and progress tracking
func (s *OpenAIService) BatchProcess(ctx context.Context, tracks []*domain.Track) error {
	if len(tracks) == 0 {
		return nil
	}

	// Create channels for error handling and progress tracking
	errChan := make(chan error, len(tracks))
	progressChan := make(chan struct{}, s.cfg.BatchSize)
	var wg sync.WaitGroup

	// Start timer for metrics
	timer := metrics.NewTimer(metrics.AIBatchProcessingDuration)
	defer timer.ObserveDuration()

	// Track batch metrics
	metrics.BatchProcessingTotal.Inc()
	defer func() {
		metrics.TracksProcessedTotal.Add(float64(len(tracks)))
	}()

	// Process tracks in parallel with batch size limit
	for i := range tracks {
		wg.Add(1)
		progressChan <- struct{}{} // Limit concurrent processing

		go func(track *domain.Track) {
			defer wg.Done()
			defer func() { <-progressChan }()

			// Process with retry logic
			err := retry.Do(
				func() error {
					// Enrich metadata
					if err := s.EnrichMetadata(ctx, track); err != nil {
						metrics.AIEnrichmentErrors.Inc()
						return fmt.Errorf("failed to enrich track %s: %w", track.ID, err)
					}

					// Validate metadata
					confidence, err := s.ValidateMetadata(ctx, track)
					if err != nil {
						metrics.AIValidationErrors.Inc()
						return fmt.Errorf("failed to validate track %s: %w", track.ID, err)
					}

					// Update track status
					track.Metadata.AI.NeedsReview = confidence < s.cfg.MinConfidence
					track.Metadata.AI.ProcessedAt = time.Now()

					return nil
				},
				retry.Attempts(3),
				retry.Delay(1*time.Second),
				retry.MaxDelay(5*time.Second),
				retry.OnRetry(func(n uint, err error) {
					metrics.AIRetryAttempts.Inc()
					logrus.Printf("Retry %d for track %s: %v", n, track.ID, err)
				}),
			)

			if err != nil {
				errChan <- err
			}
		}(tracks[i])
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)
	close(progressChan)

	// Collect and aggregate errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
		metrics.AIBatchErrors.Inc()
	}

	// Return combined error if any occurred
	if len(errors) > 0 {
		var errMsg strings.Builder
		errMsg.WriteString(fmt.Sprintf("batch processing encountered %d errors:\n", len(errors)))
		for _, err := range errors {
			errMsg.WriteString("- " + err.Error() + "\n")
		}
		return fmt.Errorf("batch processing errors: %s", errMsg.String())
	}

	return nil
}
