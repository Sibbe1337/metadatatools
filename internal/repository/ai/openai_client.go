package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"metadatatool/internal/pkg/domain"
)

// OpenAIClient handles communication with the OpenAI API
type OpenAIClient struct {
	config     *domain.OpenAIConfig
	httpClient *http.Client
}

// OpenAIResponse represents the response from OpenAI's audio analysis
type OpenAIResponse struct {
	Energy       float64 `json:"energy"`
	Danceability float64 `json:"danceability"`
	Confidence   float64 `json:"confidence"`
	ReviewReason string  `json:"review_reason,omitempty"`
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config *domain.OpenAIConfig) (*OpenAIClient, error) {
	if config == nil {
		return nil, fmt.Errorf("OpenAI config is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("OpenAI endpoint is required")
	}

	return &OpenAIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
	}, nil
}

// AnalyzeAudio sends audio data to OpenAI for analysis
func (c *OpenAIClient) AnalyzeAudio(ctx context.Context, audioData []byte, format string) (*OpenAIResponse, error) {
	url := fmt.Sprintf("%s/v1/audio/analyze", c.config.Endpoint)

	// Prepare request body
	body := struct {
		AudioData []byte `json:"audio_data"`
		Format    string `json:"format"`
	}{
		AudioData: audioData,
		Format:    format,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %s (status code: %d)", string(respBody), resp.StatusCode)
	}

	// Parse response
	var result OpenAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ValidateMetadata validates track metadata using OpenAI
func (c *OpenAIClient) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	url := fmt.Sprintf("%s/v1/metadata/validate", c.config.Endpoint)

	// Convert CompleteTrackMetadata to Metadata
	metadata := &domain.Metadata{
		ISRC:         track.ISRC(),
		ISWC:         track.ISWC(),
		BPM:          track.BPM(),
		Key:          track.Key(),
		Mood:         track.Mood(),
		Labels:       []string{},
		AITags:       track.AITags(),
		Confidence:   track.AIConfidence(),
		ModelVersion: track.ModelVersion(),
		CustomFields: track.Metadata.Additional.CustomFields,
	}

	// Convert custom tags to labels
	for tag := range track.Metadata.Additional.CustomTags {
		metadata.Labels = append(metadata.Labels, tag)
	}

	// Prepare request body
	body := struct {
		Metadata *domain.Metadata `json:"metadata"`
	}{
		Metadata: metadata,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result struct {
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Confidence, nil
}
