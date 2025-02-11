package queue

import (
	"context"
	"metadatatool/internal/pkg/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockQueueService(t *testing.T) {
	ctx := context.Background()
	service := NewMockQueueService()

	// Test message
	message := &domain.Message{
		ID:        "test-message",
		Type:      string(domain.QueueMessageTypeAIProcess),
		Priority:  domain.PriorityHigh,
		Data:      map[string]interface{}{"test": "data"},
		Status:    domain.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	t.Run("Publish and Subscribe", func(t *testing.T) {
		// Test message handler
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
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for message")
		}

		// Verify message was stored
		messages := service.GetMessages(topic)
		require.Len(t, messages, 1)
		assert.Equal(t, message.ID, messages[0].ID)
	})

	t.Run("Dead Letter Queue", func(t *testing.T) {
		// Test dead letter handling
		message.ErrorMessage = "test error"
		message.RetryCount = 3
		message.Status = domain.MessageStatusFailed

		err := service.HandleDeadLetter(ctx, "test-topic", message)
		require.NoError(t, err)

		// Verify dead letter count
		assert.Equal(t, 1, service.GetDeadLetterCount())
	})
}
