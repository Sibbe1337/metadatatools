package middleware

import (
	"context"
	"metadatatool/internal/pkg/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSessionStore is a mock implementation of domain.SessionStore
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Create(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*domain.Session), args.Error(1)
}

func (m *MockSessionStore) Update(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionStore) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockSessionStore) Touch(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestSession_Middleware(t *testing.T) {
	sessionStore := new(MockSessionStore)
	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}

	t.Run("no session cookie", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(Session(sessionStore, cfg))

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
		sessionStore.AssertNotCalled(t, "Get")
	})

	t.Run("valid session", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(Session(sessionStore, cfg))

		session := &domain.Session{
			ID:         "test-session",
			UserID:     "test-user",
			ExpiresAt:  time.Now().Add(time.Hour),
			LastSeenAt: time.Now(),
		}

		sessionStore.On("Get", mock.Anything, "test-session").Return(session, nil)
		sessionStore.On("Touch", mock.Anything, "test-session").Return(nil)

		var gotSession *domain.Session
		router.GET("/test", func(c *gin.Context) {
			s, exists := c.Get("session")
			assert.True(t, exists)
			gotSession = s.(*domain.Session)
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, session.ID, gotSession.ID)
		sessionStore.AssertExpectations(t)
	})

	t.Run("invalid session", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(Session(sessionStore, cfg))

		sessionStore.On("Get", mock.Anything, "invalid-session").Return(nil, nil)

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "invalid-session"})
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
		sessionStore.AssertExpectations(t)

		// Check that cookie was removed
		var found bool
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == "session_id" {
				assert.Less(t, cookie.MaxAge, 0)
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("session store error", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(Session(sessionStore, cfg))

		sessionStore.On("Get", mock.Anything, "test-session").Return(nil, assert.AnError)

		router.GET("/test", func(c *gin.Context) {
			t.Error("handler should not be called")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		sessionStore.AssertExpectations(t)
	})
}

func TestCreateSession_Middleware(t *testing.T) {
	sessionStore := new(MockSessionStore)
	cfg := &domain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
	}

	t.Run("create session for authenticated user", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(CreateSession(sessionStore, cfg))

		claims := &domain.Claims{
			UserID: "test-user",
			Email:  "test@example.com",
		}

		sessionStore.On("Create", mock.Anything, mock.MatchedBy(func(s *domain.Session) bool {
			return s.UserID == claims.UserID
		})).Return(nil)

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Set("user", claims)
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
		sessionStore.AssertExpectations(t)

		// Check that session cookie was set
		var found bool
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == "session_id" {
				assert.NotEmpty(t, cookie.Value)
				assert.True(t, cookie.Secure)
				assert.True(t, cookie.HttpOnly)
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("no session created for unauthenticated user", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(CreateSession(sessionStore, cfg))

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
		sessionStore.AssertNotCalled(t, "Create")
	})
}

func TestClearSession_Middleware(t *testing.T) {
	sessionStore := new(MockSessionStore)

	t.Run("clear existing session", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(ClearSession(sessionStore))

		sessionStore.On("Delete", mock.Anything, "test-session").Return(nil)

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "test-session"})
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
		sessionStore.AssertExpectations(t)

		// Check that cookie was removed
		var found bool
		for _, cookie := range w.Result().Cookies() {
			if cookie.Name == "session_id" {
				assert.Less(t, cookie.MaxAge, 0)
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("no session to clear", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(ClearSession(sessionStore))

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
		sessionStore.AssertNotCalled(t, "Delete")
	})
}

func TestRequireSession_Middleware(t *testing.T) {
	t.Run("session exists", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(RequireSession())

		session := &domain.Session{
			ID:     "test-session",
			UserID: "test-user",
		}

		var called bool
		router.GET("/test", func(c *gin.Context) {
			called = true
			c.Set("session", session)
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("no session", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(RequireSession())

		router.GET("/test", func(c *gin.Context) {
			t.Error("handler should not be called")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
