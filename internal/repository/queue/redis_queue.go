package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix        = "queue:"
	pendingPrefix    = "pending:"
	processingPrefix = "processing:"
	deadLetterPrefix = "dead_letter:"

	// Lock duration for message processing
	processingLockDuration = 5 * time.Minute
)

// RedisQueue implements domain.QueueService using Redis
type RedisQueue struct {
	client    *redis.Client
	config    domain.QueueConfig
	handlers  map[string]domain.MessageHandler
	mu        sync.RWMutex
	done      chan struct{}
	closeOnce sync.Once
}

// NewRedisQueue creates a new Redis-based queue service
func NewRedisQueue(client *redis.Client, config domain.QueueConfig) *RedisQueue {
	q := &RedisQueue{
		client:   client,
		config:   config,
		handlers: make(map[string]domain.MessageHandler),
		done:     make(chan struct{}),
	}

	// Start background workers
	go q.processMessages()
	go q.cleanupExpired()

	return q
}

// Publish publishes a message to a topic
func (q *RedisQueue) Publish(ctx context.Context, topic string, data []byte) error {
	timer := prometheus.NewTimer(metrics.MessageProcessingDuration.WithLabelValues(topic))
	defer timer.ObserveDuration()

	// Unmarshal data into map
	var dataMap map[string]interface{}
	if err := json.Unmarshal(data, &dataMap); err != nil {
		metrics.QueueOperations.WithLabelValues("publish", topic, "failure").Inc()
		return fmt.Errorf("failed to unmarshal message data: %w", err)
	}

	msg := &domain.Message{
		ID:        uuid.NewString(),
		Type:      topic,
		Data:      dataMap,
		Status:    domain.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Marshal message data
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		metrics.QueueOperations.WithLabelValues("publish", topic, "failure").Inc()
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Store in Redis
	pipe := q.client.Pipeline()
	pipe.Set(ctx, processingPrefix+msg.ID, msgBytes, 0)
	pipe.LPush(ctx, keyPrefix+msg.Type, msg.ID)
	pipe.IncrBy(ctx, keyPrefix+msg.Type+":size", 1)

	if _, err := pipe.Exec(ctx); err != nil {
		metrics.QueueOperations.WithLabelValues("publish", topic, "failure").Inc()
		return fmt.Errorf("failed to store message: %w", err)
	}

	metrics.QueueOperations.WithLabelValues("publish", topic, "success").Inc()
	metrics.QueueSize.WithLabelValues(topic).Inc()
	return nil
}

// Subscribe subscribes to a topic with a message handler
func (q *RedisQueue) Subscribe(ctx context.Context, topic string, handler domain.MessageHandler) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.handlers[topic]; exists {
		return fmt.Errorf("handler already registered for topic: %s", topic)
	}

	q.handlers[topic] = handler
	metrics.QueueOperations.WithLabelValues("subscribe", topic, "success").Inc()
	return nil
}

// Unsubscribe removes a subscription from a topic
func (q *RedisQueue) Unsubscribe(ctx context.Context, topic string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.handlers[topic]; !exists {
		return fmt.Errorf("no handler registered for topic: %s", topic)
	}

	delete(q.handlers, topic)
	metrics.QueueOperations.WithLabelValues("unsubscribe", topic, "success").Inc()
	return nil
}

// GetMessage retrieves a message by ID
func (q *RedisQueue) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	data, err := q.client.Get(ctx, processingPrefix+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	var msg domain.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// RetryMessage marks a message for retry
func (q *RedisQueue) RetryMessage(ctx context.Context, id string) error {
	msg, err := q.GetMessage(ctx, id)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", id)
	}

	if msg.RetryCount >= msg.MaxRetries {
		return q.moveToDeadLetter(ctx, msg)
	}

	msg.RetryCount++
	msg.Status = domain.MessageStatusRetrying
	msg.UpdatedAt = time.Now()
	msg.NextRetryAt = q.calculateNextRetry(msg.RetryCount)

	// Store updated message
	msgBytes, _ := json.Marshal(msg)
	pipe := q.client.Pipeline()
	pipe.Set(ctx, processingPrefix+msg.ID, msgBytes, 0)
	pipe.LPush(ctx, keyPrefix+msg.Type, msg.ID)
	pipe.IncrBy(ctx, keyPrefix+msg.Type+":size", 1)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to retry message: %w", err)
	}

	metrics.MessageRetries.WithLabelValues(msg.Type).Inc()
	return nil
}

// AckMessage acknowledges a message as processed
func (q *RedisQueue) AckMessage(ctx context.Context, id string) error {
	msg, err := q.GetMessage(ctx, id)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", id)
	}

	now := time.Now()
	msg.Status = domain.MessageStatusCompleted
	msg.ProcessedAt = &now
	msg.UpdatedAt = now

	// Store updated message and clean up
	msgBytes, _ := json.Marshal(msg)
	pipe := q.client.Pipeline()
	pipe.Set(ctx, processingPrefix+msg.ID, msgBytes, 0)
	pipe.Del(ctx, processingPrefix+msg.ID)
	pipe.DecrBy(ctx, keyPrefix+msg.Type+":size", 1)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}

	metrics.QueueOperations.WithLabelValues("ack", msg.Type, "success").Inc()
	metrics.QueueSize.WithLabelValues(msg.Type).Dec()
	return nil
}

// NackMessage marks a message as failed
func (q *RedisQueue) NackMessage(ctx context.Context, id string, err error) error {
	msg, getErr := q.GetMessage(ctx, id)
	if getErr != nil {
		return getErr
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", id)
	}

	msg.Status = domain.MessageStatusFailed
	msg.UpdatedAt = time.Now()
	msg.ErrorMessage = err.Error()

	if msg.RetryCount < msg.MaxRetries {
		return q.RetryMessage(ctx, id)
	}

	return q.moveToDeadLetter(ctx, msg)
}

// ListDeadLetters retrieves messages in the dead letter queue
func (q *RedisQueue) ListDeadLetters(ctx context.Context, topic string, offset, limit int) ([]*domain.Message, error) {
	ids, err := q.client.LRange(ctx, deadLetterPrefix+topic, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list dead letters: %w", err)
	}

	messages := make([]*domain.Message, 0, len(ids))
	for _, id := range ids {
		if msg, err := q.GetMessage(ctx, id); err == nil && msg != nil {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// ReplayDeadLetter moves a message from dead letter queue back to main queue
func (q *RedisQueue) ReplayDeadLetter(ctx context.Context, id string) error {
	msg, err := q.GetMessage(ctx, id)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found: %s", id)
	}

	msg.Status = domain.MessageStatusPending
	msg.RetryCount = 0
	msg.UpdatedAt = time.Now()
	msg.NextRetryAt = nil
	msg.ProcessedAt = nil
	msg.DeadLetterAt = nil

	// Move message back to main queue
	msgBytes, _ := json.Marshal(msg)
	pipe := q.client.Pipeline()
	pipe.Set(ctx, processingPrefix+msg.ID, msgBytes, 0)
	pipe.LRem(ctx, deadLetterPrefix+msg.Type, 0, msg.ID)
	pipe.LPush(ctx, keyPrefix+msg.Type, msg.ID)
	pipe.IncrBy(ctx, keyPrefix+msg.Type+":size", 1)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to replay dead letter: %w", err)
	}

	metrics.DeadLetterMessages.WithLabelValues(msg.Type).Dec()
	metrics.QueueSize.WithLabelValues(msg.Type).Inc()
	return nil
}

// PurgeDeadLetters removes all messages from dead letter queue
func (q *RedisQueue) PurgeDeadLetters(ctx context.Context, topic string) error {
	pipe := q.client.Pipeline()
	pipe.Del(ctx, deadLetterPrefix+topic)
	pipe.Set(ctx, deadLetterPrefix+topic+":size", 0, 0)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to purge dead letters: %w", err)
	}

	metrics.DeadLetterMessages.WithLabelValues(topic).Set(0)
	return nil
}

// Close closes the queue service
func (q *RedisQueue) Close() error {
	q.closeOnce.Do(func() {
		close(q.done)
	})
	return nil
}

// Helper methods

func (q *RedisQueue) processMessages() {
	ticker := time.NewTicker(q.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.done:
			return
		case <-ticker.C:
			q.processBatch()
		}
	}
}

func (q *RedisQueue) processBatch() {
	ctx := context.Background()
	q.mu.RLock()
	topics := make([]string, 0, len(q.handlers))
	for topic := range q.handlers {
		topics = append(topics, topic)
	}
	q.mu.RUnlock()

	for _, topic := range topics {
		for i := 0; i < q.config.BatchSize; i++ {
			id, err := q.client.RPopLPush(ctx, keyPrefix+topic, processingPrefix+topic).Result()
			if err == redis.Nil {
				break
			}
			if err != nil {
				metrics.ProcessingErrors.WithLabelValues(topic, "redis_error").Inc()
				continue
			}

			go q.processMessage(ctx, id, topic)
		}
	}
}

func (q *RedisQueue) processMessage(ctx context.Context, id, topic string) {
	start := time.Now()
	msg, err := q.GetMessage(ctx, id)
	if err != nil {
		metrics.ProcessingErrors.WithLabelValues(topic, "get_message_error").Inc()
		return
	}

	q.mu.RLock()
	handler := q.handlers[topic]
	q.mu.RUnlock()

	if handler == nil {
		metrics.ProcessingErrors.WithLabelValues(topic, "no_handler").Inc()
		return
	}

	// Set processing status
	msg.Status = domain.MessageStatusProcessing
	msg.UpdatedAt = time.Now()
	msgBytes, _ := json.Marshal(msg)
	q.client.Set(ctx, processingPrefix+id, msgBytes, 0)

	// Create processing context with timeout
	ctx, cancel := context.WithTimeout(ctx, q.config.ProcessingTimeout)
	defer cancel()

	// Process message
	err = handler(ctx, msg)
	duration := time.Since(start)
	metrics.MessageProcessingDuration.WithLabelValues(topic).Observe(duration.Seconds())

	if err != nil {
		metrics.ProcessingErrors.WithLabelValues(topic, "handler_error").Inc()
		q.NackMessage(ctx, id, err)
	} else {
		q.AckMessage(ctx, id)
	}
}

func (q *RedisQueue) moveToDeadLetter(ctx context.Context, msg *domain.Message) error {
	now := time.Now()
	msg.Status = domain.MessageStatusDeadLetter
	msg.UpdatedAt = now
	msg.DeadLetterAt = &now

	// Move message to dead letter queue
	msgBytes, _ := json.Marshal(msg)
	pipe := q.client.Pipeline()
	pipe.Set(ctx, processingPrefix+msg.ID, msgBytes, 0)
	pipe.LPush(ctx, deadLetterPrefix+msg.Type, msg.ID)
	pipe.DecrBy(ctx, keyPrefix+msg.Type+":size", 1)
	pipe.IncrBy(ctx, deadLetterPrefix+msg.Type+":size", 1)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to move to dead letter: %w", err)
	}

	metrics.DeadLetterMessages.WithLabelValues(msg.Type).Inc()
	metrics.QueueSize.WithLabelValues(msg.Type).Dec()
	return nil
}

func (q *RedisQueue) calculateNextRetry(retryCount int) *time.Time {
	if retryCount <= 0 || retryCount > len(q.config.RetryDelays) {
		return nil
	}

	delay := time.Duration(q.config.RetryDelays[retryCount-1]) * time.Second
	next := time.Now().Add(delay)
	return &next
}

func (q *RedisQueue) cleanupExpired() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-q.done:
			return
		case <-ticker.C:
			ctx := context.Background()
			q.mu.RLock()
			topics := make([]string, 0, len(q.handlers))
			for topic := range q.handlers {
				topics = append(topics, topic)
			}
			q.mu.RUnlock()

			for _, topic := range topics {
				q.cleanupExpiredForTopic(ctx, topic)
			}
		}
	}
}

func (q *RedisQueue) cleanupExpiredForTopic(ctx context.Context, topic string) {
	// Cleanup expired processing messages
	ids, _ := q.client.LRange(ctx, processingPrefix+topic, 0, -1).Result()
	for _, id := range ids {
		msg, err := q.GetMessage(ctx, id)
		if err != nil || msg == nil {
			continue
		}

		if time.Since(msg.UpdatedAt) > processingLockDuration {
			q.RetryMessage(ctx, id)
		}
	}

	// Cleanup expired dead letter messages
	if q.config.DeadLetterTTL > 0 {
		ids, _ = q.client.LRange(ctx, deadLetterPrefix+topic, 0, -1).Result()
		for _, id := range ids {
			msg, err := q.GetMessage(ctx, id)
			if err != nil || msg == nil {
				continue
			}

			if msg.DeadLetterAt != nil && time.Since(*msg.DeadLetterAt) > q.config.DeadLetterTTL {
				pipe := q.client.Pipeline()
				pipe.Del(ctx, processingPrefix+id)
				pipe.LRem(ctx, deadLetterPrefix+topic, 0, id)
				pipe.DecrBy(ctx, deadLetterPrefix+topic+":size", 1)
				pipe.Exec(ctx)

				metrics.DeadLetterMessages.WithLabelValues(topic).Dec()
			}
		}
	}
}
