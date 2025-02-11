package ai

import (
	"context"
	"fmt"
	"io"
	pkgdomain "metadatatool/internal/pkg/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockQwen2Client is a mock implementation of the Qwen2ClientInterface
type mockQwen2Client struct {
	mock.Mock
}

func (m *mockQwen2Client) AnalyzeAudio(ctx context.Context, audioData io.Reader, format pkgdomain.AudioFormat) (*Qwen2Response, error) {
	args := m.Called(ctx, audioData, format)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Qwen2Response), args.Error(1)
}

func (m *mockQwen2Client) ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (float64, error) {
	args := m.Called(ctx, track)
	return args.Get(0).(float64), args.Error(1)
}

func TestNewQwen2Service(t *testing.T) {
	tests := []struct {
		name    string
		config  *pkgdomain.Qwen2Config
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing API key",
			config: &pkgdomain.Qwen2Config{
				Endpoint: "https://api.qwen2.ai",
			},
			wantErr: true,
		},
		{
			name: "missing endpoint",
			config: &pkgdomain.Qwen2Config{
				APIKey: "test-key",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &pkgdomain.Qwen2Config{
				APIKey:                "test-key",
				Endpoint:              "https://api.qwen2.ai",
				TimeoutSeconds:        30,
				MinConfidence:         0.85,
				MaxConcurrentRequests: 10,
				RetryAttempts:         3,
				RetryBackoffSeconds:   2,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewQwen2Service(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, service)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
			}
		})
	}
}

func TestQwen2Service_EnrichMetadata(t *testing.T) {
	mockClient := &mockQwen2Client{}
	config := &pkgdomain.Qwen2Config{
		APIKey:                "test-key",
		Endpoint:              "https://api.qwen2.ai",
		TimeoutSeconds:        30,
		MinConfidence:         0.85,
		MaxConcurrentRequests: 10,
		RetryAttempts:         3,
		RetryBackoffSeconds:   1,
	}

	service, err := NewQwen2ServiceWithClient(config, mockClient)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	tests := []struct {
		name    string
		track   *pkgdomain.Track
		mockFn  func()
		wantErr bool
	}{
		{
			name:    "nil track",
			track:   nil,
			mockFn:  func() {},
			wantErr: true,
		},
		{
			name: "successful enrichment",
			track: &pkgdomain.Track{
				ID:        "track-1",
				AudioData: []byte("test audio data"),
			},
			mockFn: func() {
				mockClient.On("AnalyzeAudio", mock.Anything, mock.Anything, mock.Anything).
					Return(&Qwen2Response{
						Metadata: struct {
							Genre      string   `json:"genre"`
							Mood       string   `json:"mood"`
							BPM        float64  `json:"bpm"`
							Key        string   `json:"key"`
							Tags       []string `json:"tags"`
							Confidence float64  `json:"confidence"`
						}{
							Genre:      "rock",
							Mood:       "energetic",
							BPM:        120.5,
							Key:        "C",
							Tags:       []string{"guitar", "drums"},
							Confidence: 0.95,
						},
					}, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "low confidence result",
			track: &pkgdomain.Track{
				ID:        "track-2",
				AudioData: []byte("test audio data"),
			},
			mockFn: func() {
				mockClient.On("AnalyzeAudio", mock.Anything, mock.Anything, mock.Anything).
					Return(&Qwen2Response{
						Metadata: struct {
							Genre      string   `json:"genre"`
							Mood       string   `json:"mood"`
							BPM        float64  `json:"bpm"`
							Key        string   `json:"key"`
							Tags       []string `json:"tags"`
							Confidence float64  `json:"confidence"`
						}{
							Genre:      "unknown",
							Mood:       "neutral",
							BPM:        0,
							Key:        "",
							Tags:       []string{},
							Confidence: 0.3,
						},
					}, nil).Once()
			},
			wantErr: false,
		},
		{
			name: "API error with retry",
			track: &pkgdomain.Track{
				ID:        "track-3",
				AudioData: []byte("test audio data"),
			},
			mockFn: func() {
				mockClient.On("AnalyzeAudio", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("API error")).Times(config.RetryAttempts + 1)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := service.EnrichMetadata(context.Background(), tt.track)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.track != nil {
					assert.NotNil(t, tt.track.Metadata.AI)
				}
			}
		})
	}

	mockClient.AssertExpectations(t)
}

func TestQwen2Service_ValidateMetadata(t *testing.T) {
	mockClient := &mockQwen2Client{}
	config := &pkgdomain.Qwen2Config{
		APIKey:                "test-key",
		Endpoint:              "https://api.qwen2.ai",
		TimeoutSeconds:        30,
		MinConfidence:         0.85,
		MaxConcurrentRequests: 10,
		RetryAttempts:         3,
		RetryBackoffSeconds:   1,
	}

	service, err := NewQwen2ServiceWithClient(config, mockClient)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	tests := []struct {
		name           string
		track          *pkgdomain.Track
		mockFn         func()
		wantConfidence float64
		wantErr        bool
	}{
		{
			name:           "nil track",
			track:          nil,
			mockFn:         func() {},
			wantConfidence: 0,
			wantErr:        true,
		},
		{
			name: "high confidence",
			track: &pkgdomain.Track{
				ID: "track-1",
			},
			mockFn: func() {
				mockClient.On("ValidateMetadata", mock.Anything, mock.Anything).
					Return(0.95, nil).Once()
			},
			wantConfidence: 0.95,
			wantErr:        false,
		},
		{
			name: "low confidence with retries",
			track: &pkgdomain.Track{
				ID: "track-2",
			},
			mockFn: func() {
				// First attempt - low confidence
				mockClient.On("ValidateMetadata", mock.Anything, mock.Anything).
					Return(0.5, nil).Once()
				// Second attempt - still low confidence
				mockClient.On("ValidateMetadata", mock.Anything, mock.Anything).
					Return(0.6, nil).Once()
				// Third attempt - still low confidence
				mockClient.On("ValidateMetadata", mock.Anything, mock.Anything).
					Return(0.7, nil).Once()
				// Fourth attempt (last retry) - still low confidence
				mockClient.On("ValidateMetadata", mock.Anything, mock.Anything).
					Return(0.8, nil).Once()
			},
			wantConfidence: 0,
			wantErr:        true,
		},
		{
			name: "error with retries",
			track: &pkgdomain.Track{
				ID: "track-3",
			},
			mockFn: func() {
				for i := 0; i <= config.RetryAttempts; i++ {
					mockClient.On("ValidateMetadata", mock.Anything, mock.Anything).
						Return(0.0, fmt.Errorf("API error")).Once()
				}
			},
			wantConfidence: 0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			confidence, err := service.ValidateMetadata(context.Background(), tt.track)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantConfidence, confidence)
			}
		})
	}

	mockClient.AssertExpectations(t)
}

func TestQwen2Service_BatchProcess(t *testing.T) {
	mockClient := &mockQwen2Client{}
	config := &pkgdomain.Qwen2Config{
		APIKey:                "test-key",
		Endpoint:              "https://api.qwen2.ai",
		TimeoutSeconds:        30,
		MinConfidence:         0.85,
		MaxConcurrentRequests: 2, // Small value to test concurrency
		RetryAttempts:         3,
		RetryBackoffSeconds:   1,
	}

	service, err := NewQwen2ServiceWithClient(config, mockClient)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	tests := []struct {
		name    string
		tracks  []*pkgdomain.Track
		mockFn  func()
		wantErr bool
	}{
		{
			name:    "empty tracks",
			tracks:  nil,
			mockFn:  func() {},
			wantErr: false,
		},
		{
			name: "successful batch",
			tracks: []*pkgdomain.Track{
				{ID: "track-1", AudioData: []byte("audio 1")},
				{ID: "track-2", AudioData: []byte("audio 2")},
				{ID: "track-3", AudioData: []byte("audio 3")},
			},
			mockFn: func() {
				response := &Qwen2Response{
					Metadata: struct {
						Genre      string   `json:"genre"`
						Mood       string   `json:"mood"`
						BPM        float64  `json:"bpm"`
						Key        string   `json:"key"`
						Tags       []string `json:"tags"`
						Confidence float64  `json:"confidence"`
					}{
						Genre:      "rock",
						Mood:       "energetic",
						BPM:        120.5,
						Key:        "C",
						Tags:       []string{"guitar", "drums"},
						Confidence: 0.95,
					},
				}

				// Each track will be processed once (no retries needed due to high confidence)
				for i := 0; i < 3; i++ {
					mockClient.On("AnalyzeAudio", mock.Anything, mock.Anything, mock.Anything).
						Return(response, nil).Once()
				}
			},
			wantErr: false,
		},
		{
			name: "partial failure",
			tracks: []*pkgdomain.Track{
				{ID: "track-4", AudioData: []byte("audio 4")},
				{ID: "track-5", AudioData: []byte("audio 5")},
			},
			mockFn: func() {
				successResponse := &Qwen2Response{
					Metadata: struct {
						Genre      string   `json:"genre"`
						Mood       string   `json:"mood"`
						BPM        float64  `json:"bpm"`
						Key        string   `json:"key"`
						Tags       []string `json:"tags"`
						Confidence float64  `json:"confidence"`
					}{
						Genre:      "rock",
						Mood:       "energetic",
						BPM:        120.5,
						Key:        "C",
						Tags:       []string{"guitar", "drums"},
						Confidence: 0.95,
					},
				}

				// First track succeeds on first try
				mockClient.On("AnalyzeAudio", mock.Anything, mock.Anything, mock.Anything).
					Return(successResponse, nil).Once()

				// Second track fails with retries
				for i := 0; i <= config.RetryAttempts; i++ {
					mockClient.On("AnalyzeAudio", mock.Anything, mock.Anything, mock.Anything).
						Return(nil, fmt.Errorf("API error")).Once()
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFn()
			err := service.BatchProcess(context.Background(), tt.tracks)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "batch processing completed with")
			} else {
				assert.NoError(t, err)
			}
		})
	}

	mockClient.AssertExpectations(t)
}

func TestQwen2Service_Metrics(t *testing.T) {
	mockClient := &mockQwen2Client{}
	config := &pkgdomain.Qwen2Config{}

	service, err := NewQwen2ServiceWithClient(config, mockClient)
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Test recordSuccess
	t.Run("recordSuccess", func(t *testing.T) {
		duration := 100 * time.Millisecond
		service.(*Qwen2Service).recordSuccess(duration)

		assert.Equal(t, int64(1), service.(*Qwen2Service).metrics.RequestCount)
		assert.Equal(t, int64(1), service.(*Qwen2Service).metrics.SuccessCount)
		assert.Equal(t, duration, service.(*Qwen2Service).metrics.AverageLatency)
		assert.NotZero(t, service.(*Qwen2Service).metrics.LastSuccess)
	})

	// Test recordFailure
	t.Run("recordFailure", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		service.(*Qwen2Service).recordFailure(testErr)

		assert.Equal(t, int64(2), service.(*Qwen2Service).metrics.RequestCount)
		assert.Equal(t, int64(1), service.(*Qwen2Service).metrics.FailureCount)
		assert.Equal(t, testErr, service.(*Qwen2Service).metrics.LastError)
	})
}
