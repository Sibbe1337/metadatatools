package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"metadatatool/internal/pkg/domain"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

// Qwen2Error represents a specific error from the Qwen2 API
type Qwen2Error struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
}

func (e *Qwen2Error) Error() string {
	return fmt.Sprintf("Qwen2 error: %s - %s", e.Code, e.Message)
}

// Qwen2Client handles communication with the Qwen2 API
type Qwen2Client struct {
	config     *domain.Qwen2Config
	httpClient *http.Client
	breaker    *gobreaker.CircuitBreaker
}

// Qwen2Response represents the response from the Qwen2 API
type Qwen2Response struct {
	Metadata struct {
		Genre      string   `json:"genre"`
		Mood       string   `json:"mood"`
		BPM        float64  `json:"bpm"`
		Key        string   `json:"key"`
		Tags       []string `json:"tags"`
		Confidence float64  `json:"confidence"`
	} `json:"metadata"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// BatchResponse represents a batch processing response
type BatchResponse struct {
	Results     []*Qwen2Response `json:"results"`
	FailedItems []struct {
		Index int         `json:"index"`
		Error *Qwen2Error `json:"error"`
	} `json:"failed_items"`
}

// NewQwen2Client creates a new Qwen2 API client
func NewQwen2Client(config *domain.Qwen2Config) (*Qwen2Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
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
	}

	return &Qwen2Client{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		breaker: gobreaker.NewCircuitBreaker(breakerSettings),
	}, nil
}

// AnalyzeAudio sends an audio file to Qwen2 for analysis
func (c *Qwen2Client) AnalyzeAudio(ctx context.Context, audioData io.Reader, format domain.AudioFormat) (*Qwen2Response, error) {
	// Prepare request body
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, audioData); err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"/analyze", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "audio/"+string(format))
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request through circuit breaker
	resp, err := c.breaker.Execute(func() (interface{}, error) {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			var qwenErr Qwen2Error
			if err := json.Unmarshal(body, &qwenErr); err != nil {
				return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			}
			return nil, &qwenErr
		}

		var response Qwen2Response
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		return &response, nil
	})

	if err != nil {
		return nil, err
	}

	return resp.(*Qwen2Response), nil
}

// ValidateMetadata validates track metadata using Qwen2
func (c *Qwen2Client) ValidateMetadata(ctx context.Context, track *domain.Track) (float64, error) {
	// Prepare request body
	body, err := json.Marshal(map[string]interface{}{
		"metadata": track.Metadata,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"/validate", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	// Execute request through circuit breaker
	resp, err := c.breaker.Execute(func() (interface{}, error) {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0.0, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0.0, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			var qwenErr Qwen2Error
			if err := json.Unmarshal(body, &qwenErr); err != nil {
				return 0.0, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
			}
			return 0.0, &qwenErr
		}

		var response struct {
			Confidence float64 `json:"confidence"`
			Error      struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error,omitempty"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return 0.0, fmt.Errorf("failed to parse response: %w", err)
		}

		return response.Confidence, nil
	})

	if err != nil {
		return 0, err
	}

	return resp.(float64), nil
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
	url := fmt.Sprintf("%s/v1/metadata/validate/batch", c.config.Endpoint)

	// Convert tracks metadata
	metadataList := make([]*domain.Metadata, len(tracks))
	for i, track := range tracks {
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

		metadataList[i] = metadata
	}

	// Prepare request body
	body := struct {
		Tracks []*domain.Metadata `json:"tracks"`
	}{
		Tracks: metadataList,
	}

	respBody, err := c.sendRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to validate metadata batch: %w", err)
	}

	var result struct {
		Confidences []float64 `json:"confidences"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Confidences, nil
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
