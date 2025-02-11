package session

import (
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/pkg/domain"

	"github.com/redis/go-redis/v9"
)

type RedisSessionStore struct {
	client *redis.Client
	config domain.SessionConfig
}

func NewRedisSessionStore(client *redis.Client, config domain.SessionConfig) *RedisSessionStore {
	return &RedisSessionStore{
		client: client,
		config: config,
	}
}

func (s *RedisSessionStore) Create(ctx context.Context, session *domain.Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	key := fmt.Sprintf("session:%s", session.ID)
	if err := s.client.Set(ctx, key, data, s.config.SessionDuration).Err(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Add to user's sessions set
	userKey := fmt.Sprintf("user:%s:sessions", session.UserID)
	if err := s.client.SAdd(ctx, userKey, session.ID).Err(); err != nil {
		return fmt.Errorf("failed to add session to user set: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) Get(ctx context.Context, id string) (*domain.Session, error) {
	key := fmt.Sprintf("session:%s", id)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session domain.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (s *RedisSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	userKey := fmt.Sprintf("user:%s:sessions", userID)
	sessionIDs, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	sessions := make([]*domain.Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		session, err := s.Get(ctx, id)
		if err != nil {
			if err == domain.ErrSessionNotFound {
				continue
			}
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (s *RedisSessionStore) Update(ctx context.Context, session *domain.Session) error {
	return s.Create(ctx, session) // Same as create since we're using Redis SET
}

func (s *RedisSessionStore) Delete(ctx context.Context, id string) error {
	session, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", id)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	userKey := fmt.Sprintf("user:%s:sessions", session.UserID)
	if err := s.client.SRem(ctx, userKey, id).Err(); err != nil {
		return fmt.Errorf("failed to remove session from user set: %w", err)
	}

	return nil
}

func (s *RedisSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	sessions, err := s.GetUserSessions(ctx, userID)
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if err := s.Delete(ctx, session.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *RedisSessionStore) DeleteExpired(ctx context.Context) error {
	// Redis automatically handles expiration through TTL
	return nil
}

func (s *RedisSessionStore) Touch(ctx context.Context, id string) error {
	// First verify the session exists
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", id)
	if err := s.client.Expire(ctx, key, s.config.SessionDuration).Err(); err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}

	return nil
}
