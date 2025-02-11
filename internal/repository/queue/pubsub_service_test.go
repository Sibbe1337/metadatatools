package queue

import (
	"context"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

func TestPubSubService(t *testing.T) {
	ctx := context.Background()

	// Create test configuration
	config := &PubSubConfig{
		ProjectID:          "test-project",
		HighPriorityTopic:  "test-high-priority",
		LowPriorityTopic:   "test-low-priority",
		DeadLetterTopic:    "test-dead-letter",
		SubscriptionPrefix: "test-sub",
		MaxRetries:         3,
		AckDeadline:        30 * time.Second,
		RetentionDuration:  24 * time.Hour,
	}

	// Create metrics
	metrics := metrics.NewQueueMetrics()

	// Create test client with emulator
	client, err := pubsub.NewClient(ctx, "test-project", option.WithEndpoint("localhost:8085"))
	require.NoError(t, err)
	defer client.Close()

	// Create service
	service, err := NewPubSubService(ctx, config, metrics)
	require.NoError(t, err)
	defer service.Close()

	t.Run("Publish and Subscribe", func(t *testing.T) {
		// Create test message
		message := &domain.Message{
			ID:        "test-message",
			Type:      string(domain.QueueMessageTypeAIProcess),
			Priority:  domain.PriorityHigh,
			Data:      map[string]interface{}{"test": "data"},
			Status:    domain.MessageStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Create channel for received messages
		messageReceived := make(chan *domain.Message, 1)
		handler := func(ctx context.Context, msg *domain.Message) error {
			messageReceived <- msg
			return nil
		}

		// Subscribe to messages
		topic := "test-topic"
		err := service.Subscribe(ctx, topic, handler)
		require.NoError(t, err)

		// Publish message
		err = service.Publish(ctx, topic, message, domain.PriorityHigh)
		require.NoError(t, err)

		// Wait for message to be received
		select {
		case received := <-messageReceived:
			assert.Equal(t, message.ID, received.ID)
			assert.Equal(t, message.Type, received.Type)
			assert.Equal(t, message.Priority, received.Priority)
			assert.Equal(t, message.Data, received.Data)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for message")
		}
	})

	t.Run("Dead Letter Queue", func(t *testing.T) {
		// Create test message
		message := &domain.Message{
			ID:           "test-message",
			Type:         string(domain.QueueMessageTypeAIProcess),
			Priority:     domain.PriorityHigh,
			Data:         map[string]interface{}{"test": "data"},
			Status:       domain.MessageStatusFailed,
			ErrorMessage: "test error",
			RetryCount:   3,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// Handle dead letter
		err := service.HandleDeadLetter(ctx, "test-topic", message)
		require.NoError(t, err)
	})
}
