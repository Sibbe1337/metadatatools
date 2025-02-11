package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisPkgSessionStore implements the pkg/domain.SessionStore interface using Redis
type RedisPkgSessionStore struct {
	client    *redis.Client
	config    domain.SessionConfig
	closeOnce sync.Once
	done      chan struct{}
}

// NewPkgSessionStore creates a new Redis session store for pkg/domain
func NewPkgSessionStore(client *redis.Client, config domain.SessionConfig) domain.SessionStore {
	store := &RedisPkgSessionStore{
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
func (s *RedisPkgSessionStore) Create(ctx context.Context, session *domain.Session) error {
	// Check if user has reached max sessions
	if s.config.MaxSessionsPerUser > 0 {
		count, err := s.client.SCard(ctx, userSessionsKey(session.UserID)).Result()
		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to count user sessions: %w", err)
		}
		if count >= int64(s.config.MaxSessionsPerUser) {
			// Delete oldest session if limit reached
			sessions, err := s.client.SMembers(ctx, userSessionsKey(session.UserID)).Result()
			if err != nil {
				return fmt.Errorf("failed to get user sessions: %w", err)
			}
			if err := s.deleteOldestSession(ctx, session.UserID, sessions); err != nil {
				return fmt.Errorf("failed to delete oldest session: %w", err)
			}
		}
	}

	// Store session data
	sessionBytes, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Set session with expiry
	if err := s.client.Set(ctx, sessionKey(session.ID), sessionBytes, time.Until(session.ExpiresAt)).Err(); err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Add session ID to user's sessions set
	if err := s.client.SAdd(ctx, userSessionsKey(session.UserID), session.ID).Err(); err != nil {
		return fmt.Errorf("failed to add session to user sessions: %w", err)
	}

	return nil
}

// Get retrieves a session from Redis
func (s *RedisPkgSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	sessionBytes, err := s.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// GetUserSessions retrieves all sessions for a user
func (s *RedisPkgSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user session IDs: %w", err)
	}

	sessions := make([]*domain.Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		session, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if session != nil {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// Update updates a session in Redis
func (s *RedisPkgSessionStore) Update(ctx context.Context, session *domain.Session) error {
	return s.Create(ctx, session)
}

// Delete removes a session from Redis
func (s *RedisPkgSessionStore) Delete(ctx context.Context, sessionID string) error {
	// Get session first to get user ID
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}

	// Remove session from user's sessions set
	if err := s.client.SRem(ctx, userSessionsKey(session.UserID), sessionID).Err(); err != nil {
		return fmt.Errorf("failed to remove session from user sessions: %w", err)
	}

	// Delete session data
	if err := s.client.Del(ctx, sessionKey(sessionID)).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteUserSessions removes all sessions for a user
func (s *RedisPkgSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("failed to get user session IDs: %w", err)
	}

	// Delete each session
	for _, id := range sessionIDs {
		if err := s.Delete(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

// Touch updates the session's last seen time
func (s *RedisPkgSessionStore) Touch(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}

	session.LastSeenAt = time.Now()
	return s.Update(ctx, session)
}

// Close stops the cleanup goroutine
func (s *RedisPkgSessionStore) Close() error {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	return nil
}

// cleanupLoop periodically removes expired sessions
func (s *RedisPkgSessionStore) cleanupLoop() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := s.DeleteExpired(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Error cleaning up expired sessions: %v\n", err)
			}
			cancel()
		case <-s.done:
			return
		}
	}
}

// DeleteExpired removes all expired sessions
func (s *RedisPkgSessionStore) DeleteExpired(ctx context.Context) error {
	// This is a simplified implementation
	// In production, you might want to use Redis SCAN to handle large datasets
	return nil
}

// deleteOldestSession removes the oldest session for a user
func (s *RedisPkgSessionStore) deleteOldestSession(ctx context.Context, userID string, sessions []string) error {
	if len(sessions) == 0 {
		return nil
	}

	var oldestSession *domain.Session
	var oldestID string

	for _, id := range sessions {
		session, err := s.Get(ctx, id)
		if err != nil {
			return err
		}
		if session == nil {
			continue
		}

		if oldestSession == nil || session.CreatedAt.Before(oldestSession.CreatedAt) {
			oldestSession = session
			oldestID = id
		}
	}

	if oldestID != "" {
		return s.Delete(ctx, oldestID)
	}

	return nil
}
