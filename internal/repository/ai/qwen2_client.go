package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// Qwen2Error represents a specific error from the Qwen2-Audio API
type Qwen2Error struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
}

func (e *Qwen2Error) Error() string {
	return fmt.Sprintf("Qwen2 error: %s - %s", e.Code, e.Message)
}

// Qwen2Client handles communication with the Qwen2-Audio API
type Qwen2Client struct {
	config     *domain.Qwen2Config
	httpClient *http.Client
	breaker    *gobreaker.CircuitBreaker
}

// Qwen2Response represents the response from Qwen2-Audio's analysis
type Qwen2Response struct {
	Energy       float64 `json:"energy"`
	Danceability float64 `json:"danceability"`
	Confidence   float64 `json:"confidence"`
	ReviewReason string  `json:"review_reason,omitempty"`
}

// BatchResponse represents a batch processing response
type BatchResponse struct {
	Results     []*Qwen2Response `json:"results"`
	FailedItems []struct {
		Index int         `json:"index"`
		Error *Qwen2Error `json:"error"`
	} `json:"failed_items"`
}

// NewQwen2Client creates a new Qwen2-Audio client
func NewQwen2Client(config *domain.Qwen2Config) (*Qwen2Client, error) {
	if config == nil {
		return nil, fmt.Errorf("Qwen2 config is required")
	}

	if config.APIKey == "" {
		return nil, fmt.Errorf("Qwen2 API key is required")
	}

	if config.Endpoint == "" {
		return nil, fmt.Errorf("Qwen2 endpoint is required")
	}

	// Configure circuit breaker
	breakerSettings := gobreaker.Settings{
		Name:        "qwen2-breaker",
		MaxRequests: 100,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 10 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			metrics.AIRequestTotal.WithLabelValues("qwen2", fmt.Sprintf("circuit_%s", to.String())).Inc()
		},
	}

	return &Qwen2Client{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		breaker: gobreaker.NewCircuitBreaker(breakerSettings),
	}, nil
}

// sendRequest handles the common logic for sending requests to Qwen2-Audio
func (c *Qwen2Client) sendRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var jsonBody []byte
	var err error

	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	// Execute request through circuit breaker
	resp, err := c.breaker.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

		// Send request with retries
		var resp *http.Response
		var lastErr error
		for attempt := 0; attempt <= c.config.RetryAttempts; attempt++ {
			if attempt > 0 {
				backoffDuration := time.Duration(c.config.RetryBackoffSeconds) * time.Second * time.Duration(1<<uint(attempt-1))
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoffDuration):
				}
			}

			resp, err = c.httpClient.Do(req)
			if err == nil {
				break
			}
			lastErr = err
		}

		if err != nil {
			return nil, fmt.Errorf("failed to send request after %d attempts: %w", c.config.RetryAttempts, lastErr)
		}

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			var qwenErr Qwen2Error
			if err := json.Unmarshal(respBody, &qwenErr); err != nil {
				return nil, fmt.Errorf("Qwen2 API error: %s (status code: %d)", string(respBody), resp.StatusCode)
			}
			return nil, &qwenErr
		}

		return respBody, nil
	})

	if err != nil {
		return nil, err
	}

	return resp.([]byte), nil
}

// AnalyzeAudio sends audio data to Qwen2-Audio for analysis
func (c *Qwen2Client) AnalyzeAudio(ctx context.Context, audioData []byte, format string) (*Qwen2Response, error) {
	url := fmt.Sprintf("%s/v1/audio/analyze", c.config.Endpoint)

	body := struct {
		AudioData []byte `json:"audio_data"`
		Format    string `json:"format"`
	}{
		AudioData: audioData,
		Format:    format,
	}

	respBody, err := c.sendRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze audio: %w", err)
	}

	var result Qwen2Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ValidateMetadata validates track metadata using Qwen2-Audio
func (c *Qwen2Client) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	url := fmt.Sprintf("%s/v1/metadata/validate", c.config.Endpoint)

	body := struct {
		Metadata domain.Metadata `json:"metadata"`
	}{
		Metadata: track.Metadata,
	}

	respBody, err := c.sendRequest(ctx, "POST", url, body)
	if err != nil {
		return 0, fmt.Errorf("failed to validate metadata: %w", err)
	}

	var result struct {
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Confidence, nil
}

// BatchAnalyzeAudio processes multiple audio files in a single request
func (c *Qwen2Client) BatchAnalyzeAudio(ctx context.Context, requests []struct {
	AudioData []byte `json:"audio_data"`
	Format    string `json:"format"`
}) (*BatchResponse, error) {
	url := fmt.Sprintf("%s/v1/audio/batch/analyze", c.config.Endpoint)

	respBody, err := c.sendRequest(ctx, "POST", url, requests)
	if err != nil {
		return nil, fmt.Errorf("failed to batch analyze audio: %w", err)
	}

	var result BatchResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	return &result, nil
}

// BatchValidateMetadata validates multiple tracks' metadata in a single request
func (c *Qwen2Client) BatchValidateMetadata(ctx context.Context, tracks []*domain.Track) ([]float64, error) {
	url := fmt.Sprintf("%s/v1/metadata/batch/validate", c.config.Endpoint)

	body := struct {
		Tracks []domain.Metadata `json:"tracks"`
	}{
		Tracks: make([]domain.Metadata, len(tracks)),
	}

	for i, track := range tracks {
		body.Tracks[i] = track.Metadata
	}

	respBody, err := c.sendRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to batch validate metadata: %w", err)
	}

	var result struct {
		Confidences []float64 `json:"confidences"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse batch validation response: %w", err)
	}

	return result.Confidences, nil
}
