package redis

import (
	"context"
	"metadatatool/internal/domain"
	"metadatatool/internal/pkg/config"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
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

func createTestSession() *domain.Session {
	now := time.Now()
	return &domain.Session{
		ID:          uuid.New().String(),
		UserID:      uuid.New().String(),
		Role:        domain.RoleUser,
		Permissions: []domain.Permission{domain.PermissionReadTrack},
		UserAgent:   "test-agent",
		IP:          "127.0.0.1",
		ExpiresAt:   now.Add(24 * time.Hour),
		CreatedAt:   now,
		LastSeenAt:  now,
	}
}

func configToDomainConfig(cfg config.SessionConfig) domain.SessionConfig {
	return domain.SessionConfig{
		CookieName:         cfg.CookieName,
		CookiePath:         cfg.CookiePath,
		CookieDomain:       cfg.CookieDomain,
		CookieSecure:       cfg.CookieSecure,
		CookieHTTPOnly:     cfg.CookieHTTPOnly,
		CookieSameSite:     cfg.CookieSameSite,
		MaxSessionsPerUser: cfg.MaxSessionsPerUser,
		SessionDuration:    cfg.SessionDuration,
		CleanupInterval:    cfg.CleanupInterval,
	}
}

func TestRedisSessionStore_Create(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		CookieName:         "session",
		CookieDomain:       "",
		CookiePath:         "/",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		CookieSameSite:     "lax",
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, configToDomainConfig(cfg))

	t.Run("successful creation", func(t *testing.T) {
		session := createTestSession()
		err := store.Create(context.Background(), session)
		require.NoError(t, err)

		// Verify session was stored
		stored, err := store.Get(context.Background(), session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, stored.ID)
		assert.Equal(t, session.UserID, stored.UserID)
	})

	t.Run("max sessions limit", func(t *testing.T) {
		userID := uuid.New().String()

		// Create first session
		session1 := createTestSession()
		session1.UserID = userID
		err := store.Create(context.Background(), session1)
		require.NoError(t, err)

		// Create second session
		session2 := createTestSession()
		session2.UserID = userID
		err = store.Create(context.Background(), session2)
		require.NoError(t, err)

		// Try to create third session
		session3 := createTestSession()
		session3.UserID = userID
		err = store.Create(context.Background(), session3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum number of sessions reached")
	})
}

func TestRedisSessionStore_Get(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		CookieName:         "session",
		CookieDomain:       "",
		CookiePath:         "/",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		CookieSameSite:     "lax",
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, configToDomainConfig(cfg))

	t.Run("get existing session", func(t *testing.T) {
		session := createTestSession()
		err := store.Create(context.Background(), session)
		require.NoError(t, err)

		stored, err := store.Get(context.Background(), session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, stored.ID)
		assert.Equal(t, session.UserID, stored.UserID)
		assert.Equal(t, session.Role, stored.Role)
		assert.Equal(t, session.Permissions, stored.Permissions)
	})

	t.Run("get non-existent session", func(t *testing.T) {
		_, err := store.Get(context.Background(), "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}

func TestRedisSessionStore_GetUserSessions(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		CookieName:         "session",
		CookieDomain:       "",
		CookiePath:         "/",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		CookieSameSite:     "lax",
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}

	store := NewSessionStore(client, configToDomainConfig(cfg))

	t.Run("get multiple user sessions", func(t *testing.T) {
		userID := uuid.New().String()

		// Create two sessions for the same user
		session1 := createTestSession()
		session1.UserID = userID
		session2 := createTestSession()
		session2.UserID = userID

		require.NoError(t, store.Create(context.Background(), session1))
		require.NoError(t, store.Create(context.Background(), session2))

		// Get user sessions
		sessions, err := store.GetUserSessions(context.Background(), userID)
		require.NoError(t, err)
		assert.Len(t, sessions, 2)

		// Verify session IDs
		sessionIDs := map[string]bool{
			sessions[0].ID: true,
			sessions[1].ID: true,
		}
		assert.True(t, sessionIDs[session1.ID])
		assert.True(t, sessionIDs[session2.ID])
	})

	t.Run("get sessions for user with no sessions", func(t *testing.T) {
		sessions, err := store.GetUserSessions(context.Background(), "non-existent")
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestRedisSessionStore_Delete(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		CookieName:         "session",
		CookieDomain:       "",
		CookiePath:         "/",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		CookieSameSite:     "lax",
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, configToDomainConfig(cfg))

	t.Run("delete existing session", func(t *testing.T) {
		session := createTestSession()
		err := store.Create(context.Background(), session)
		require.NoError(t, err)

		// Delete session
		err = store.Delete(context.Background(), session.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = store.Get(context.Background(), session.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})

	t.Run("delete non-existent session", func(t *testing.T) {
		err := store.Delete(context.Background(), "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}

func TestRedisSessionStore_DeleteUserSessions(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		CookieName:         "session",
		CookieDomain:       "",
		CookiePath:         "/",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		CookieSameSite:     "lax",
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}

	store := NewSessionStore(client, configToDomainConfig(cfg))

	t.Run("delete all user sessions", func(t *testing.T) {
		userID := uuid.New().String()

		// Create multiple sessions
		session1 := createTestSession()
		session1.UserID = userID
		session2 := createTestSession()
		session2.UserID = userID

		require.NoError(t, store.Create(context.Background(), session1))
		require.NoError(t, store.Create(context.Background(), session2))

		// Delete all user sessions
		err := store.DeleteUserSessions(context.Background(), userID)
		require.NoError(t, err)

		// Verify deletion
		sessions, err := store.GetUserSessions(context.Background(), userID)
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})
}

func TestRedisSessionStore_Touch(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()

	cfg := config.SessionConfig{
		CookieName:         "session",
		CookieDomain:       "",
		CookiePath:         "/",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		CookieSameSite:     "lax",
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 2,
	}

	store := NewSessionStore(client, configToDomainConfig(cfg))

	t.Run("touch existing session", func(t *testing.T) {
		session := createTestSession()
		err := store.Create(context.Background(), session)
		require.NoError(t, err)

		// Get initial last seen time
		initialLastSeen := session.LastSeenAt

		// Wait a bit to ensure time difference
		time.Sleep(time.Millisecond * 100)

		// Touch session
		err = store.Touch(context.Background(), session.ID)
		require.NoError(t, err)

		// Verify touch
		updated, err := store.Get(context.Background(), session.ID)
		require.NoError(t, err)
		assert.True(t, updated.LastSeenAt.After(initialLastSeen))
	})

	t.Run("touch non-existent session", func(t *testing.T) {
		err := store.Touch(context.Background(), "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session not found")
	})
}
