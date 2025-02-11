package domain

import (
	"context"
	"fmt"
	"time"
)

// QueuePriority represents the priority level of a queue message
type QueuePriority int

const (
	PriorityLow QueuePriority = iota
	PriorityMedium
	PriorityHigh
)

// QueueMessageType represents the type of message
type QueueMessageType string

const (
	QueueMessageTypeAIProcess QueueMessageType = "ai_process"
	QueueMessageTypeDDEX      QueueMessageType = "ddex"
	QueueMessageTypeCleanup   QueueMessageType = "cleanup"
)

// QueueMessage represents a message in the queue
type QueueMessage struct {
	ID         string           `json:"id"`
	Type       QueueMessageType `json:"type"`
	Priority   QueuePriority    `json:"priority"`
	Payload    []byte           `json:"payload"`
	PubSubID   string           `json:"pubsub_id,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	Error      string           `json:"error,omitempty"`
	RetryCount int              `json:"retry_count"`
}

// MessageStatus represents the current status of a message
type MessageStatus string

const (
	// MessageStatusPending indicates the message is waiting to be processed
	MessageStatusPending MessageStatus = "pending"
	// MessageStatusProcessing indicates the message is being processed
	MessageStatusProcessing MessageStatus = "processing"
	// MessageStatusCompleted indicates the message was processed successfully
	MessageStatusCompleted MessageStatus = "completed"
	// MessageStatusFailed indicates the message processing failed
	MessageStatusFailed MessageStatus = "failed"
	// MessageStatusRetrying indicates the message is being retried
	MessageStatusRetrying MessageStatus = "retrying"
	// MessageStatusDeadLetter indicates the message has been moved to dead letter queue
	MessageStatusDeadLetter MessageStatus = "dead_letter"
)

// Message represents a queue message
type Message struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Data         map[string]interface{} `json:"data"`
	Status       MessageStatus          `json:"status"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	NextRetryAt  *time.Time             `json:"next_retry_at,omitempty"`
	ProcessedAt  *time.Time             `json:"processed_at,omitempty"`
	DeadLetterAt *time.Time             `json:"dead_letter_at,omitempty"`
	Priority     QueuePriority          `json:"priority"`
}

// MessageHandler is a function that processes a message
type MessageHandler func(ctx context.Context, msg *Message) error

// Publisher defines the interface for publishing messages
type Publisher interface {
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, data []byte) error
	// PublishWithRetry publishes a message with retry configuration
	PublishWithRetry(ctx context.Context, topic string, data []byte, maxRetries int) error
}

// Subscriber defines the interface for subscribing to messages
type Subscriber interface {
	// Subscribe subscribes to a topic with a message handler
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	// Unsubscribe removes a subscription from a topic
	Unsubscribe(ctx context.Context, topic string) error
}

// QueueService combines Publisher and Subscriber interfaces
type QueueService interface {
	Publish(ctx context.Context, topic string, message *Message, priority QueuePriority) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	HandleDeadLetter(ctx context.Context, topic string, message *Message) error
	// GetMessage retrieves a message by ID
	GetMessage(ctx context.Context, id string) (*Message, error)
	// RetryMessage marks a message for retry
	RetryMessage(ctx context.Context, id string) error
	// AckMessage acknowledges a message as processed
	AckMessage(ctx context.Context, id string) error
	// NackMessage marks a message as failed
	NackMessage(ctx context.Context, id string, err error) error
	// ListDeadLetters retrieves messages in the dead letter queue
	ListDeadLetters(ctx context.Context, topic string, offset, limit int) ([]*Message, error)
	// ReplayDeadLetter moves a message from dead letter queue back to main queue
	ReplayDeadLetter(ctx context.Context, id string) error
	// PurgeDeadLetters removes all messages from dead letter queue
	PurgeDeadLetters(ctx context.Context, topic string) error
	// Close closes the queue service
	Close() error
}

// QueueConfig holds configuration for the queue service
type QueueConfig struct {
	// Connection settings
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`

	// Processing settings
	RetryDelays       []int         `json:"retry_delays"`
	DefaultMaxRetries int           `json:"default_max_retries"`
	DeadLetterTTL     time.Duration `json:"dead_letter_ttl"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`
	BatchSize         int           `json:"batch_size"`
	PollInterval      time.Duration `json:"poll_interval"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
}

// DefaultQueueConfig returns a default configuration
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		RetryDelays:       []int{1, 5, 15, 30, 60}, // Exponential backoff
		DefaultMaxRetries: 5,
		DeadLetterTTL:     7 * 24 * time.Hour, // 7 days
		ProcessingTimeout: 5 * time.Minute,
		BatchSize:         100,
		PollInterval:      time.Second,
		CleanupInterval:   1 * time.Hour,
	}
}

// Validate checks if the configuration is valid
func (c *QueueConfig) Validate() error {
	if len(c.RetryDelays) == 0 {
		return fmt.Errorf("retry delays must not be empty")
	}
	if c.DefaultMaxRetries <= 0 {
		return fmt.Errorf("default max retries must be positive")
	}
	if c.DeadLetterTTL <= 0 {
		return fmt.Errorf("dead letter TTL must be positive")
	}
	if c.ProcessingTimeout <= 0 {
		return fmt.Errorf("processing timeout must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be positive")
	}
	if c.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup interval must be positive")
	}
	return nil
}

type Queue interface {
	Close() error
	Publish(ctx context.Context, topic string, data []byte) error
	PublishWithRetry(ctx context.Context, topic string, data []byte, maxRetries int) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	Unsubscribe(ctx context.Context, topic string) error
	GetMessage(ctx context.Context, id string) (*Message, error)
	RetryMessage(ctx context.Context, id string) error
	AckMessage(ctx context.Context, id string) error
	NackMessage(ctx context.Context, id string, err error) error
	ListDeadLetters(ctx context.Context, topic string, offset, limit int) ([]*Message, error)
	ReplayDeadLetter(ctx context.Context, id string) error
	PurgeDeadLetters(ctx context.Context, topic string) error
}

// String returns the string representation of QueuePriority
func (p QueuePriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	default:
		return "unknown"
	}
}
