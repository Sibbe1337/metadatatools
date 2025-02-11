package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/metrics"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
)

// PubSubConfig holds configuration for Google Pub/Sub
type PubSubConfig struct {
	ProjectID          string        `env:"PUBSUB_PROJECT_ID,required"`
	HighPriorityTopic  string        `env:"PUBSUB_HIGH_PRIORITY_TOPIC" envDefault:"high-priority"`
	LowPriorityTopic   string        `env:"PUBSUB_LOW_PRIORITY_TOPIC" envDefault:"low-priority"`
	DeadLetterTopic    string        `env:"PUBSUB_DEAD_LETTER_TOPIC" envDefault:"dead-letter"`
	SubscriptionPrefix string        `env:"PUBSUB_SUBSCRIPTION_PREFIX" envDefault:"sub"`
	MaxRetries         int           `env:"PUBSUB_MAX_RETRIES" envDefault:"3"`
	AckDeadline        time.Duration `env:"PUBSUB_ACK_DEADLINE" envDefault:"30s"`
	RetentionDuration  time.Duration `env:"PUBSUB_RETENTION" envDefault:"168h"` // 7 days
}

// PubSubService implements domain.QueueService using Google Pub/Sub
type PubSubService struct {
	client   *pubsub.Client
	config   *PubSubConfig
	topics   map[string]*pubsub.Topic
	subs     map[string]*pubsub.Subscription
	metrics  *metrics.QueueMetrics
	handlers map[string]domain.MessageHandler
	mu       sync.RWMutex // protects handlers and topics
}

// NewPubSubService creates a new Google Pub/Sub service
func NewPubSubService(ctx context.Context, config *PubSubConfig, metrics *metrics.QueueMetrics) (*PubSubService, error) {
	if config == nil {
		return nil, fmt.Errorf("pubsub config is required")
	}

	// Create Pub/Sub client
	client, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	return &PubSubService{
		client:   client,
		config:   config,
		topics:   make(map[string]*pubsub.Topic),
		subs:     make(map[string]*pubsub.Subscription),
		metrics:  metrics,
		handlers: make(map[string]domain.MessageHandler),
	}, nil
}

// getTopic returns the appropriate topic based on priority
func (s *PubSubService) getTopic(priority domain.QueuePriority) string {
	switch priority {
	case domain.PriorityHigh:
		return s.config.HighPriorityTopic
	case domain.PriorityLow:
		return s.config.LowPriorityTopic
	default:
		return s.config.LowPriorityTopic
	}
}

// Publish publishes a message to the appropriate topic
func (s *PubSubService) Publish(ctx context.Context, topic string, message *domain.Message, priority domain.QueuePriority) error {
	// Record start time for latency tracking
	start := time.Now()

	// Marshal message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		s.metrics.PublishErrors.WithLabelValues(topic).Inc()
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Get the appropriate topic
	topicName := s.getTopic(priority)
	t, err := s.ensureTopic(ctx, topicName)
	if err != nil {
		return err
	}
	defer t.Stop()

	// Set message attributes
	attrs := map[string]string{
		"priority":   priority.String(),
		"message_id": message.ID,
		"type":       string(message.Type),
	}

	// Publish message
	result := t.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: attrs,
	})

	// Wait for publish result
	_, err = result.Get(ctx)
	if err != nil {
		s.metrics.PublishErrors.WithLabelValues(topicName).Inc()
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Record metrics
	s.metrics.PublishLatency.WithLabelValues(topicName).Observe(time.Since(start).Seconds())
	s.metrics.MessagesPublished.WithLabelValues(topicName).Inc()

	return nil
}

// Subscribe subscribes to a topic and processes messages
func (s *PubSubService) Subscribe(ctx context.Context, topic string, handler domain.MessageHandler) error {
	if handler == nil {
		return fmt.Errorf("message handler is required")
	}

	// Register handler
	s.mu.Lock()
	s.handlers[topic] = handler
	s.mu.Unlock()

	// Get or create subscription
	sub, err := s.ensureSubscription(ctx, topic)
	if err != nil {
		return err
	}

	// Configure subscription
	sub.ReceiveSettings.MaxOutstandingMessages = 100
	sub.ReceiveSettings.NumGoroutines = 10

	// Start receiving messages
	go func() {
		err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			// Record start time for latency tracking
			start := time.Now()

			// Unmarshal message
			var message domain.Message
			if err := json.Unmarshal(msg.Data, &message); err != nil {
				s.metrics.ProcessingErrors.WithLabelValues(topic).Inc()
				msg.Nack()
				return
			}

			// Get handler
			s.mu.RLock()
			handler := s.handlers[topic]
			s.mu.RUnlock()

			// Process message
			if err := handler(ctx, &message); err != nil {
				s.metrics.ProcessingErrors.WithLabelValues(topic).Inc()
				msg.Nack()
				return
			}

			// Record metrics
			s.metrics.ProcessingLatency.WithLabelValues(topic).Observe(time.Since(start).Seconds())
			s.metrics.MessagesProcessed.WithLabelValues(topic).Inc()

			msg.Ack()
		})
		if err != nil {
			// Log error and increment metric
			s.metrics.SubscriptionErrors.WithLabelValues(topic).Inc()
		}
	}()

	return nil
}

// HandleDeadLetter processes messages from the dead letter queue
func (s *PubSubService) HandleDeadLetter(ctx context.Context, topic string, message *domain.Message) error {
	// Record dead letter handling
	s.metrics.DeadLetters.WithLabelValues(string(message.Type)).Inc()

	// Here you would implement your dead letter handling logic
	// For example:
	// - Log the failure
	// - Send notifications
	// - Store for manual review
	// - Attempt special processing

	return nil
}

// Close closes the Pub/Sub client
func (s *PubSubService) Close() error {
	return s.client.Close()
}

func (s *PubSubService) ensureTopic(ctx context.Context, name string) (*pubsub.Topic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.topics[name]; ok {
		return t, nil
	}

	t := s.client.Topic(name)
	exists, err := t.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check topic existence: %w", err)
	}

	if !exists {
		t, err = s.client.CreateTopic(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create topic: %w", err)
		}
	}

	s.topics[name] = t
	return t, nil
}

func (s *PubSubService) ensureSubscription(ctx context.Context, topic string) (*pubsub.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sub, ok := s.subs[topic]; ok {
		return sub, nil
	}

	t, err := s.ensureTopic(ctx, topic)
	if err != nil {
		return nil, err
	}

	subName := fmt.Sprintf("%s-sub", topic)
	sub := s.client.Subscription(subName)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check subscription existence: %w", err)
	}

	if !exists {
		// Create subscription with configuration
		sub, err = s.client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
			Topic:             t,
			AckDeadline:       s.config.AckDeadline,
			RetentionDuration: s.config.RetentionDuration,
			RetryPolicy: &pubsub.RetryPolicy{
				MaximumBackoff: time.Minute,
				MinimumBackoff: time.Second,
			},
			DeadLetterPolicy: &pubsub.DeadLetterPolicy{
				DeadLetterTopic:     s.client.Topic(s.config.DeadLetterTopic).String(),
				MaxDeliveryAttempts: s.config.MaxRetries,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create subscription: %w", err)
		}
	}

	s.subs[topic] = sub
	return sub, nil
}

func (s *PubSubService) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	// Not implemented for PubSub as it doesn't support direct message retrieval
	return nil, fmt.Errorf("get message not supported for PubSub")
}

func (s *PubSubService) RetryMessage(ctx context.Context, id string) error {
	// Not implemented for PubSub as it handles retries automatically
	return fmt.Errorf("retry message not supported for PubSub")
}

func (s *PubSubService) AckMessage(ctx context.Context, id string) error {
	// Not implemented for PubSub as acks are handled in the message handler
	return fmt.Errorf("ack message not supported for PubSub")
}

func (s *PubSubService) NackMessage(ctx context.Context, id string, err error) error {
	// Not implemented for PubSub as nacks are handled in the message handler
	return fmt.Errorf("nack message not supported for PubSub")
}

func (s *PubSubService) ListDeadLetters(ctx context.Context, topic string, offset, limit int) ([]*domain.Message, error) {
	// Not implemented for PubSub as it doesn't support listing dead letters
	return nil, fmt.Errorf("list dead letters not supported for PubSub")
}

func (s *PubSubService) ReplayDeadLetter(ctx context.Context, id string) error {
	// Not implemented for PubSub as it doesn't support replaying dead letters
	return fmt.Errorf("replay dead letter not supported for PubSub")
}

func (s *PubSubService) PurgeDeadLetters(ctx context.Context, topic string) error {
	// Not implemented for PubSub as it doesn't support purging dead letters
	return fmt.Errorf("purge dead letters not supported for PubSub")
}
