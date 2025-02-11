package queue

import (
	"context"
	"errors"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"sync"
	"time"
)

var (
	ErrMessageNotFound = errors.New("message not found")
)

// MockQueueService implements domain.QueueService for testing
type MockQueueService struct {
	messages    map[string][]*domain.Message
	deadLetters []*domain.Message
	handlers    map[string]domain.MessageHandler
	mu          sync.RWMutex
}

// NewMockQueueService creates a new mock queue service
func NewMockQueueService() *MockQueueService {
	return &MockQueueService{
		messages: make(map[string][]*domain.Message),
		handlers: make(map[string]domain.MessageHandler),
	}
}

// Publish adds a message to the mock queue
func (s *MockQueueService) Publish(ctx context.Context, topic string, message *domain.Message, priority domain.QueuePriority) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages[topic] = append(s.messages[topic], message)

	if handler, ok := s.handlers[topic]; ok {
		go handler(ctx, message)
	}

	return nil
}

// Subscribe registers a handler for a topic
func (s *MockQueueService) Subscribe(ctx context.Context, topic string, handler domain.MessageHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[topic] = handler
	return nil
}

// HandleDeadLetter processes messages from the dead letter queue
func (s *MockQueueService) HandleDeadLetter(ctx context.Context, topic string, message *domain.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deadLetters = append(s.deadLetters, message)
	return nil
}

// GetMessage returns a message by ID
func (s *MockQueueService) GetMessage(ctx context.Context, id string) (*domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, messages := range s.messages {
		for _, msg := range messages {
			if msg.ID == id {
				return msg, nil
			}
		}
	}

	return nil, ErrMessageNotFound
}

// GetMessages returns all messages for a topic
func (s *MockQueueService) GetMessages(topic string) []*domain.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.messages[topic]
}

// GetDeadLetterCount returns the number of dead letter messages processed
func (s *MockQueueService) GetDeadLetterCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.deadLetters)
}

func (s *MockQueueService) RetryMessage(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, messages := range s.messages {
		for _, msg := range messages {
			if msg.ID == id {
				msg.RetryCount++
				msg.Status = domain.MessageStatusRetrying
				return nil
			}
		}
	}

	return ErrMessageNotFound
}

func (s *MockQueueService) AckMessage(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, messages := range s.messages {
		for _, msg := range messages {
			if msg.ID == id {
				msg.Status = domain.MessageStatusCompleted
				msg.ProcessedAt = &time.Time{}
				*msg.ProcessedAt = time.Now()
				return nil
			}
		}
	}

	return ErrMessageNotFound
}

func (s *MockQueueService) NackMessage(ctx context.Context, id string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, messages := range s.messages {
		for _, msg := range messages {
			if msg.ID == id {
				msg.Status = domain.MessageStatusFailed
				msg.ErrorMessage = err.Error()
				return nil
			}
		}
	}

	return ErrMessageNotFound
}

func (s *MockQueueService) ListDeadLetters(ctx context.Context, topic string, offset, limit int) ([]*domain.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if offset < 0 || limit < 0 {
		return nil, fmt.Errorf("invalid offset or limit")
	}

	var deadLetters []*domain.Message
	for _, msg := range s.deadLetters {
		if msg.Status == domain.MessageStatusDeadLetter {
			deadLetters = append(deadLetters, msg)
		}
	}

	if offset >= len(deadLetters) {
		return []*domain.Message{}, nil
	}

	end := offset + limit
	if end > len(deadLetters) {
		end = len(deadLetters)
	}

	return deadLetters[offset:end], nil
}

func (s *MockQueueService) ReplayDeadLetter(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, msg := range s.deadLetters {
		if msg.ID == id {
			msg.Status = domain.MessageStatusRetrying
			msg.RetryCount = 0
			msg.ErrorMessage = ""

			// Remove from dead letters
			s.deadLetters = append(s.deadLetters[:i], s.deadLetters[i+1:]...)
			return nil
		}
	}

	return ErrMessageNotFound
}

func (s *MockQueueService) PurgeDeadLetters(ctx context.Context, topic string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deadLetters = nil
	return nil
}
