package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// Key prefixes
	sessionKeyPrefix = "session:"
	userKeyPrefix    = "user-sessions:"
)

type sessionStore struct {
	client *redis.Client
	cfg    *domain.SessionConfig
}

// NewSessionStore creates a new Redis session store
func NewSessionStore(client *redis.Client, cfg *domain.SessionConfig) domain.SessionStore {
	store := &sessionStore{
		client: client,
		cfg:    cfg,
	}

	// Start cleanup goroutine
	go store.cleanupLoop()

	return store
}

// Create stores a new session in Redis
func (s *sessionStore) Create(ctx context.Context, session *domain.Session) error {
	// Check if user has too many sessions
	sessions, err := s.GetUserSessions(ctx, session.UserID)
	if err != nil {
		return fmt.Errorf("failed to check user sessions: %w", err)
	}

	if len(sessions) >= s.cfg.MaxSessionsPerUser {
		return fmt.Errorf("maximum number of sessions reached for user")
	}

	// Marshal session to JSON
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Store session data with expiration
	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, session.ID)
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, session.UserID)

	pipe := s.client.Pipeline()
	pipe.Set(ctx, sessionKey, sessionData, time.Until(session.ExpiresAt))
	pipe.SAdd(ctx, userKey, session.ID)
	pipe.ExpireAt(ctx, userKey, session.ExpiresAt)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	return nil
}

// Get retrieves a session by ID
func (s *sessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, sessionID)
	data, err := s.client.Get(ctx, sessionKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *sessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, userID)
	sessionIDs, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessionIDs) == 0 {
		return []*domain.Session{}, nil
	}

	var sessions []*domain.Session
	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(sessionIDs))

	for i, sessionID := range sessionIDs {
		sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, sessionID)
		cmds[i] = pipe.Get(ctx, sessionKey)
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	for _, cmd := range cmds {
		data, err := cmd.Bytes()
		if err != nil {
			if err == redis.Nil {
				continue
			}
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
func (s *sessionStore) Update(ctx context.Context, session *domain.Session) error {
	sessionData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, session.ID)
	if err := s.client.Set(ctx, sessionKey, sessionData, time.Until(session.ExpiresAt)).Err(); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// Delete removes a session
func (s *sessionStore) Delete(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil
	}

	sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, sessionID)
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, session.UserID)

	pipe := s.client.Pipeline()
	pipe.Del(ctx, sessionKey)
	pipe.SRem(ctx, userKey, sessionID)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteUserSessions removes all sessions for a user
func (s *sessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	sessions, err := s.GetUserSessions(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	if len(sessions) == 0 {
		return nil
	}

	pipe := s.client.Pipeline()
	userKey := fmt.Sprintf("%s%s", userKeyPrefix, userID)

	for _, session := range sessions {
		sessionKey := fmt.Sprintf("%s%s", sessionKeyPrefix, session.ID)
		pipe.Del(ctx, sessionKey)
	}
	pipe.Del(ctx, userKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	return nil
}

// DeleteExpired removes all expired sessions
func (s *sessionStore) DeleteExpired(ctx context.Context) error {
	// This is handled automatically by Redis TTL
	return nil
}

// Touch updates the last seen time of a session
func (s *sessionStore) Touch(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	session.LastSeenAt = time.Now()
	session.ExpiresAt = time.Now().Add(s.cfg.SessionDuration)

	return s.Update(ctx, session)
}

// cleanupLoop runs the cleanup job at regular intervals
func (s *sessionStore) cleanupLoop() {
	ticker := time.NewTicker(s.cfg.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		if err := s.DeleteExpired(ctx); err != nil {
			// Log error but continue
			fmt.Printf("Failed to cleanup expired sessions: %v\n", err)
		}
	}
}
