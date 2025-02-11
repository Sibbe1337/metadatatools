package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// Key prefixes
	sessionKeyPrefix    = "session:"
	userSessionsPrefix  = "user_sessions:"
	defaultCleanupBatch = 1000
)

// RedisSessionStore implements the domain.SessionStore interface using Redis
type RedisSessionStore struct {
	client    *redis.Client
	config    domain.SessionConfig
	closeOnce sync.Once
	done      chan struct{}
}

// NewSessionStore creates a new Redis session store
func NewSessionStore(client *redis.Client, config domain.SessionConfig) domain.SessionStore {
	store := &RedisSessionStore{
		client: client,
		config: config,
		done:   make(chan struct{}),
	}

	// Start cleanup goroutine if cleanup interval is set
	if config.CleanupInterval > 0 {
		go store.cleanupLoop()
	}

	return store
}

// Create stores a new session in Redis
func (s *RedisSessionStore) Create(ctx context.Context, session *domain.Session) error {
	// Check if user has reached max sessions
	if s.config.MaxSessionsPerUser > 0 {
		count, err := s.client.SCard(ctx, userSessionsKey(session.UserID)).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to count user sessions: %w", err)
		}
		if count >= int64(s.config.MaxSessionsPerUser) {
			return fmt.Errorf("maximum number of sessions reached for user")
		}
	}

	// Marshal session to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store session with expiration
	sessionKey := sessionKey(session.ID)
	pipe := s.client.Pipeline()
	pipe.Set(ctx, sessionKey, data, time.Until(session.ExpiresAt))
	pipe.SAdd(ctx, userSessionsKey(session.UserID), session.ID)
	pipe.ExpireAt(ctx, userSessionsKey(session.UserID), session.ExpiresAt)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (s *RedisSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	data, err := s.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if err == redis.Nil {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *RedisSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user session IDs: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*domain.Session{}, nil
	}

	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(sessionIDs))

	for i, id := range sessionIDs {
		cmds[i] = pipe.Get(ctx, sessionKey(id))
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	sessions := make([]*domain.Session, 0, len(sessionIDs))
	for _, cmd := range cmds {
		data, err := cmd.Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get session data: %w", err)
		}

		var session domain.Session
		if err := json.Unmarshal(data, &session); err != nil {
			return nil, fmt.Errorf("failed to unmarshal session: %w", err)
		}
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// Update updates an existing session
func (s *RedisSessionStore) Update(ctx context.Context, session *domain.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = s.client.Set(ctx, sessionKey(session.ID), data, time.Until(session.ExpiresAt)).Err()
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// Delete removes a session
func (s *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	// Get session first to get user ID
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, sessionKey(sessionID))
	pipe.SRem(ctx, userSessionsKey(session.UserID), sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteUserSessions removes all sessions for a user
func (s *RedisSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	sessions, err := s.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	for _, session := range sessions {
		pipe.Del(ctx, sessionKey(session.ID))
	}
	pipe.Del(ctx, userSessionsKey(userID))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired sessions
func (s *RedisSessionStore) DeleteExpired(ctx context.Context) error {
	// This is handled automatically by Redis TTL
	return nil
}

// Touch updates the last seen time of a session
func (s *RedisSessionStore) Touch(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}

	session.LastSeenAt = time.Now()
	return s.Update(ctx, session)
}

// Close stops the cleanup goroutine
func (s *RedisSessionStore) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	return nil
}

// cleanupLoop periodically runs cleanup of expired sessions
func (s *RedisSessionStore) cleanupLoop() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.DeleteExpired(context.Background()); err != nil {
				// Log error but continue
				fmt.Printf("Failed to cleanup expired sessions: %v\n", err)
			}
		case <-s.done:
			return
		}
	}
}

// Helper functions for Redis keys
func sessionKey(id string) string {
	return fmt.Sprintf("%s%s", sessionKeyPrefix, id)
}

func userSessionsKey(userID string) string {
	return fmt.Sprintf("%s%s", userSessionsPrefix, userID)
}

func (s *RedisSessionStore) CreateSession(ctx context.Context, userID string) (*domain.Session, error) {
	// Get current sessions for user
	userSessionsKey := fmt.Sprintf("%s%s", userSessionsPrefix, userID)
	currentSessions, err := s.client.SMembers(ctx, userSessionsKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Check if max sessions limit is reached
	if len(currentSessions) >= s.config.MaxSessionsPerUser {
		// Delete oldest session
		if err := s.deleteOldestSession(ctx, userID, currentSessions); err != nil {
			return nil, fmt.Errorf("failed to delete oldest session: %w", err)
		}
	}

	// Create new session
	sessionID := uuid.NewString()
	session := &domain.Session{
		ID:        sessionID,
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(s.config.SessionDuration),
	}

	// Marshal session data
	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store session in Redis
	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, sessionID)
	pipe := s.client.Pipeline()
	pipe.Set(ctx, sessionKey, sessionData, s.config.SessionDuration)
	pipe.SAdd(ctx, userSessionsKey, sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return session, nil
}

func (s *RedisSessionStore) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, sessionID)
	data, err := s.client.Get(ctx, sessionKey).Result()
	if err == redis.Nil {
		return nil, domain.ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *RedisSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	// Get session first to get user ID
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Delete session and remove from user sessions set
	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, sessionID)
	userSessionsKey := fmt.Sprintf("%s%s", userSessionsPrefix, session.UserID)

	pipe := s.client.Pipeline()
	pipe.Del(ctx, sessionKey)
	pipe.SRem(ctx, userSessionsKey, sessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) deleteOldestSession(ctx context.Context, userID string, sessions []string) error {
	var oldestSession *domain.Session
	var oldestSessionID string

	// Find oldest session
	for _, sessionID := range sessions {
		session, err := s.GetSession(ctx, sessionID)
		if err != nil {
			if err == domain.ErrSessionNotFound {
				continue
			}
			return fmt.Errorf("failed to get session for user %s: %w", userID, err)
		}

		// Validate that the session belongs to the correct user
		if session.UserID != userID {
			return fmt.Errorf("session %s does not belong to user %s", sessionID, userID)
		}

		if oldestSession == nil || session.CreatedAt.Before(oldestSession.CreatedAt) {
			oldestSession = session
			oldestSessionID = sessionID
		}
	}

	if oldestSessionID != "" {
		if err := s.DeleteSession(ctx, oldestSessionID); err != nil {
			return fmt.Errorf("failed to delete oldest session for user %s: %w", userID, err)
		}
	}

	return nil
}
