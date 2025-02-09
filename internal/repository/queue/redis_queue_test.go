package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	topicPrefix   = "test_topic_"
	messagePrefix = "test_msg_"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use a different DB for testing
	})

	// Verify connection
	_, err := client.Ping(context.Background()).Result()
	require.NoError(t, err)

	// Return client and cleanup function
	return client, func() {
		client.FlushDB(context.Background())
		client.Close()
	}
}

func setupTestQueue(t *testing.T) (*RedisQueue, func()) {
	client, cleanup := setupTestRedis(t)

	config := domain.QueueConfig{
		RetryDelays:       []int{1, 5, 15, 30, 60},
		DefaultMaxRetries: 3,
		DeadLetterTTL:     24 * time.Hour,
		ProcessingTimeout: 30 * time.Second,
		BatchSize:         10,
		PollInterval:      time.Second,
	}

	queue := NewRedisQueue(client, config)
	return queue, cleanup
}

func TestRedisQueue_PublishAndSubscribe(t *testing.T) {
	queue, cleanup := setupTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	topic := "test-topic"
	messageData := map[string]interface{}{
		"payload": "test message",
	}

	// Subscribe to topic
	msgChan := make(chan *domain.Message, 1)
	err := queue.Subscribe(ctx, topic, func(ctx context.Context, msg *domain.Message) error {
		msgChan <- msg
		return nil
	})
	require.NoError(t, err)

	// Publish message
	messageBytes, err := json.Marshal(messageData)
	require.NoError(t, err)
	err = queue.Publish(ctx, topic, messageBytes)
	require.NoError(t, err)

	// Wait for message
	select {
	case msg := <-msgChan:
		assert.Equal(t, messageData, msg.Data)
		assert.Equal(t, domain.MessageStatusProcessing, msg.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestRedisQueue_RetryAndDeadLetter(t *testing.T) {
	queue, cleanup := setupTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	topic := "test-topic"
	messageData := map[string]interface{}{
		"payload": "test message",
	}

	// Publish message
	messageBytes, err := json.Marshal(messageData)
	require.NoError(t, err)
	err = queue.Publish(ctx, topic, messageBytes)
	require.NoError(t, err)

	// Wait for retries and dead letter
	time.Sleep(2 * time.Second)

	// Check dead letter queue
	messages, err := queue.ListDeadLetters(ctx, topic, 0, 10)
	require.NoError(t, err)
	require.Len(t, messages, 1)
}

func TestRedisQueue_MessageLifecycle(t *testing.T) {
	queue, cleanup := setupTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	topic := "test-topic"
	messageData := map[string]interface{}{
		"payload": "test message",
	}

	// Publish message
	messageBytes, err := json.Marshal(messageData)
	require.NoError(t, err)
	err = queue.Publish(ctx, topic, messageBytes)
	require.NoError(t, err)

	// Get message from queue
	messages, err := queue.client.LRange(ctx, topicPrefix+topic, 0, -1).Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)

	msgID := messages[0]
	msg, err := queue.GetMessage(ctx, msgID)
	require.NoError(t, err)
	assert.Equal(t, messageData, msg.Data)

	// Acknowledge message
	err = queue.AckMessage(ctx, msgID)
	require.NoError(t, err)

	// Verify message status
	msg, err = queue.GetMessage(ctx, msgID)
	require.NoError(t, err)
	assert.Equal(t, domain.MessageStatusCompleted, msg.Status)
}

func TestRedisQueue_DeadLetterOperations(t *testing.T) {
	queue, cleanup := setupTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	topic := "test-topic"
	now := time.Now()

	// Create test messages
	for i := 0; i < 5; i++ {
		msg := &domain.Message{
			ID:   fmt.Sprintf("test-msg-%d", i),
			Type: topic,
			Data: map[string]interface{}{
				"payload": fmt.Sprintf("test message %d", i),
			},
			Status:    domain.MessageStatusDeadLetter,
			CreatedAt: now,
			UpdatedAt: now,
		}
		// Add messages to dead letter queue
		err := queue.moveToDeadLetter(ctx, msg)
		require.NoError(t, err)
	}

	// Test listing dead letters
	messages, err := queue.ListDeadLetters(ctx, topic, 0, 10)
	require.NoError(t, err)
	require.Len(t, messages, 5)
}

func TestRedisQueue_Cleanup(t *testing.T) {
	queue, cleanup := setupTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	topic := "test-topic"
	processingLockDuration := 5 * time.Minute
	now := time.Now().Add(-processingLockDuration * 2)

	// Create a test message that's expired
	msg := &domain.Message{
		ID:   "test-msg",
		Type: topic,
		Data: map[string]interface{}{
			"payload": "test message",
		},
		Status:    domain.MessageStatusProcessing,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Store the full message in Redis
	messageKey := fmt.Sprintf("message:%s", msg.ID)
	messageBytes, err := json.Marshal(msg)
	require.NoError(t, err)
	err = queue.client.Set(ctx, messageKey, messageBytes, 0).Err()
	require.NoError(t, err)

	// Add message to processing set
	err = queue.client.Set(ctx, fmt.Sprintf("%s%s", processingPrefix, msg.ID), "test", processingLockDuration).Err()
	require.NoError(t, err)

	// Run cleanup
	queue.cleanupExpired()

	// Verify message was cleaned up from processing set
	exists, err := queue.client.Exists(ctx, fmt.Sprintf("%s%s", processingPrefix, msg.ID)).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists)

	// Verify message data is still intact
	storedBytes, err := queue.client.Get(ctx, messageKey).Bytes()
	require.NoError(t, err)

	var storedMsg domain.Message
	err = json.Unmarshal(storedBytes, &storedMsg)
	require.NoError(t, err)

	assert.Equal(t, msg.ID, storedMsg.ID)
	assert.Equal(t, msg.Type, storedMsg.Type)
	assert.Equal(t, msg.Data, storedMsg.Data)
	assert.Equal(t, msg.Status, storedMsg.Status)
	assert.Equal(t, msg.CreatedAt.Unix(), storedMsg.CreatedAt.Unix())
	assert.Equal(t, msg.UpdatedAt.Unix(), storedMsg.UpdatedAt.Unix())
}

func TestRedisQueue_Concurrency(t *testing.T) {
	queue, cleanup := setupTestQueue(t)
	defer cleanup()

	ctx := context.Background()
	topic := "test-topic"
	messageCount := 100

	// Channel to track processed messages
	processed := make(chan *domain.Message, messageCount)

	// Subscribe to topic
	err := queue.Subscribe(ctx, topic, func(ctx context.Context, msg *domain.Message) error {
		processed <- msg
		return nil
	})
	require.NoError(t, err)

	// Publish messages concurrently
	for i := 0; i < messageCount; i++ {
		go func(i int) {
			data := map[string]interface{}{
				"payload": fmt.Sprintf("message-%d", i),
			}
			messageBytes, err := json.Marshal(data)
			require.NoError(t, err)
			err = queue.Publish(ctx, topic, messageBytes)
			require.NoError(t, err)
		}(i)
	}

	// Wait for all messages to be processed
	receivedMessages := make(map[string]bool)
	timeout := time.After(10 * time.Second)

	for i := 0; i < messageCount; i++ {
		select {
		case msg := <-processed:
			payload, ok := msg.Data["payload"].(string)
			require.True(t, ok)
			receivedMessages[payload] = true
		case <-timeout:
			t.Fatal("timeout waiting for messages")
		}
	}

	assert.Len(t, receivedMessages, messageCount)
}
