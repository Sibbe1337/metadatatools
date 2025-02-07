package redis

import (
	"context"
	"metadatatool/internal/pkg/domain"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client, func() {
		client.Close()
		mr.Close()
	}
}

func createTestSession(userID string) *domain.Session {
	return &domain.Session{
		ID:           uuid.NewString(),
		UserID:       userID,
		RefreshToken: uuid.NewString(),
		UserAgent:    "test-agent",
		IP:           "127.0.0.1",
		CreatedAt:    time.Now(),
		LastSeenAt:   time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
}

func TestSessionStore_Create(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, cfg)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		session := createTestSession("user1")
		err := store.Create(ctx, session)
		assert.NoError(t, err)

		// Verify session was stored
		stored, err := store.Get(ctx, session.ID)
		assert.NoError(t, err)
		assert.Equal(t, session.ID, stored.ID)
		assert.Equal(t, session.UserID, stored.UserID)
	})

	t.Run("enforce max sessions per user", func(t *testing.T) {
		// Create max number of sessions
		for i := 0; i < cfg.MaxSessionsPerUser; i++ {
			session := createTestSession("user2")
			err := store.Create(ctx, session)
			assert.NoError(t, err)
		}

		// Try to create one more session
		session := createTestSession("user2")
		err := store.Create(ctx, session)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum number of sessions")
	})
}

func TestSessionStore_Get(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, cfg)
	ctx := context.Background()

	t.Run("get existing session", func(t *testing.T) {
		session := createTestSession("user1")
		err := store.Create(ctx, session)
		require.NoError(t, err)

		stored, err := store.Get(ctx, session.ID)
		assert.NoError(t, err)
		assert.NotNil(t, stored)
		assert.Equal(t, session.ID, stored.ID)
	})

	t.Run("get non-existent session", func(t *testing.T) {
		stored, err := store.Get(ctx, "non-existent")
		assert.NoError(t, err)
		assert.Nil(t, stored)
	})
}

func TestSessionStore_GetUserSessions(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}

	store := NewSessionStore(client, cfg)
	ctx := context.Background()

	t.Run("get all user sessions", func(t *testing.T) {
		userID := "user1"
		var createdSessions []*domain.Session

		// Create multiple sessions
		for i := 0; i < 3; i++ {
			session := createTestSession(userID)
			err := store.Create(ctx, session)
			require.NoError(t, err)
			createdSessions = append(createdSessions, session)
		}

		// Get all sessions
		sessions, err := store.GetUserSessions(ctx, userID)
		assert.NoError(t, err)
		assert.Len(t, sessions, len(createdSessions))

		// Verify each session
		sessionMap := make(map[string]*domain.Session)
		for _, s := range sessions {
			sessionMap[s.ID] = s
		}

		for _, created := range createdSessions {
			stored, exists := sessionMap[created.ID]
			assert.True(t, exists)
			assert.Equal(t, created.UserID, stored.UserID)
		}
	})

	t.Run("get sessions for user with no sessions", func(t *testing.T) {
		sessions, err := store.GetUserSessions(ctx, "non-existent")
		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestSessionStore_Delete(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, cfg)
	ctx := context.Background()

	t.Run("delete existing session", func(t *testing.T) {
		session := createTestSession("user1")
		err := store.Create(ctx, session)
		require.NoError(t, err)

		// Delete session
		err = store.Delete(ctx, session.ID)
		assert.NoError(t, err)

		// Verify session was deleted
		stored, err := store.Get(ctx, session.ID)
		assert.NoError(t, err)
		assert.Nil(t, stored)

		// Verify session was removed from user sessions
		sessions, err := store.GetUserSessions(ctx, session.UserID)
		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("delete non-existent session", func(t *testing.T) {
		err := store.Delete(ctx, "non-existent")
		assert.NoError(t, err)
	})
}

func TestSessionStore_DeleteUserSessions(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}

	store := NewSessionStore(client, cfg)
	ctx := context.Background()

	t.Run("delete all user sessions", func(t *testing.T) {
		userID := "user1"

		// Create multiple sessions
		for i := 0; i < 3; i++ {
			session := createTestSession(userID)
			err := store.Create(ctx, session)
			require.NoError(t, err)
		}

		// Delete all sessions
		err := store.DeleteUserSessions(ctx, userID)
		assert.NoError(t, err)

		// Verify all sessions were deleted
		sessions, err := store.GetUserSessions(ctx, userID)
		assert.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("delete sessions for user with no sessions", func(t *testing.T) {
		err := store.DeleteUserSessions(ctx, "non-existent")
		assert.NoError(t, err)
	})
}

func TestSessionStore_Touch(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, cfg)
	ctx := context.Background()

	t.Run("touch existing session", func(t *testing.T) {
		session := createTestSession("user1")
		originalLastSeen := session.LastSeenAt
		err := store.Create(ctx, session)
		require.NoError(t, err)

		// Wait a moment to ensure time difference
		time.Sleep(time.Millisecond * 100)

		// Touch session
		err = store.Touch(ctx, session.ID)
		assert.NoError(t, err)

		// Verify last seen time was updated
		stored, err := store.Get(ctx, session.ID)
		assert.NoError(t, err)
		assert.NotNil(t, stored)
		assert.True(t, stored.LastSeenAt.After(originalLastSeen))
	})

	t.Run("touch non-existent session", func(t *testing.T) {
		err := store.Touch(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}
