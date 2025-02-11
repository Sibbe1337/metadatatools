package middleware

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"metadatatool/internal/pkg/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSessionStore is a mock implementation of pkgdomain.SessionStore
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Get(ctx context.Context, id string) (*pkgdomain.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.Session), args.Error(1)
}

func (m *MockSessionStore) Create(ctx context.Context, session *pkgdomain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionStore) Touch(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*pkgdomain.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*pkgdomain.Session), args.Error(1)
}

func (m *MockSessionStore) Update(ctx context.Context, session *pkgdomain.Session) error {
	args := m.Called(ctx, session)
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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func setupTestContext() (*gin.Context, *testResponseWriter) {
	gin.SetMode(gin.TestMode)
	w := newTestResponseWriter()
	c, _ := gin.CreateTestContext(w)
	c.Writer = w
	return c, w
}

func TestSession_Middleware(t *testing.T) {
	store := &MockSessionStore{}
	config := domain.SessionConfig{
		CookieName:     "session_id",
		CookiePath:     "/",
		CookieDomain:   "",
		CookieSecure:   true,
		CookieHTTPOnly: true,
	}

	tests := []struct {
		name           string
		setupMocks     func()
		setupRequest   func(*http.Request)
		expectedStatus int
		checkContext   func(*testing.T, *gin.Context)
	}{
		{
			name: "valid session",
			setupMocks: func() {
				session := &domain.Session{
					ID:        "test-session",
					UserID:    "test-user",
					Role:      domain.RoleUser,
					ExpiresAt: time.Now().Add(time.Hour),
				}
				store.On("Get", mock.Anything, "test-session").Return(session, nil)
				store.On("Touch", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  config.CookieName,
					Value: "test-session",
				})
			},
			expectedStatus: http.StatusOK,
			checkContext: func(t *testing.T, c *gin.Context) {
				session, exists := c.Get("session")
				assert.True(t, exists)
				assert.NotNil(t, session)
				s := session.(*domain.Session)
				assert.Equal(t, "test-user", s.UserID)
			},
		},
		{
			name: "session store error",
			setupMocks: func() {
				store.On("Get", mock.Anything, "test-session").Return(nil, errors.New("store error"))
				store.On("Delete", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  config.CookieName,
					Value: "test-session",
				})
			},
			expectedStatus: http.StatusInternalServerError,
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("session")
				assert.False(t, exists)
			},
		},
		{
			name: "expired session",
			setupMocks: func() {
				session := &domain.Session{
					ID:        "test-session",
					UserID:    "test-user",
					Role:      domain.RoleUser,
					ExpiresAt: time.Now().Add(-time.Hour), // Expired
				}
				store.On("Get", mock.Anything, "test-session").Return(session, nil)
				store.On("Delete", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  config.CookieName,
					Value: "test-session",
				})
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("session")
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			store.ExpectedCalls = nil

			// Setup router with middleware
			router := gin.New()
			router.Use(Session(store, config))
			router.GET("/test", func(c *gin.Context) {
				tt.checkContext(t, c)
				c.Status(tt.expectedStatus)
			})

			// Setup request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)

			// Setup mocks
			tt.setupMocks()

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify all mock expectations were met
			store.AssertExpectations(t)
		})
	}
}

func TestCreateSession_Middleware(t *testing.T) {
	store := &MockSessionStore{}
	cfg := pkgdomain.SessionConfig{
		CookieName:         "session_id",
		CookiePath:         "/",
		CookieDomain:       "localhost",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
		MaxSessionsPerUser: 5,
		SessionDuration:    24 * time.Hour,
	}

	t.Run("creates_session_for_authenticated_user", func(t *testing.T) {
		store.ExpectedCalls = nil

		claims := &domain.Claims{
			UserID:      "test-user",
			Role:        domain.RoleUser,
			Permissions: []domain.Permission{domain.PermissionReadTrack},
		}

		// Set up mock expectations
		store.On("GetUserSessions", mock.Anything, "test-user").Return([]*domain.Session{}, nil).Once()
		store.On("Create", mock.Anything, mock.MatchedBy(func(s *domain.Session) bool {
			return s.UserID == claims.UserID &&
				s.Role == claims.Role &&
				len(s.Permissions) == len(claims.Permissions) &&
				s.Permissions[0] == claims.Permissions[0] &&
				!s.ExpiresAt.IsZero() &&
				!s.CreatedAt.IsZero() &&
				!s.LastSeenAt.IsZero()
		})).Return(nil).Once()

		router := setupTestRouter()

		// First middleware to set claims
		router.Use(func(c *gin.Context) {
			c.Set("claims", claims)
			c.Next()
		})

		// Then the CreateSession middleware
		router.Use(CreateSession(store, cfg))

		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)

		// Verify session cookie was set
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == cfg.CookieName {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie)
		assert.NotEmpty(t, sessionCookie.Value)
		assert.Equal(t, cfg.CookiePath, sessionCookie.Path)
		assert.Equal(t, cfg.CookieDomain, sessionCookie.Domain)
		assert.Equal(t, cfg.CookieSecure, sessionCookie.Secure)
		assert.Equal(t, cfg.CookieHTTPOnly, sessionCookie.HttpOnly)
	})

	t.Run("no_session_created_for_unauthenticated_user", func(t *testing.T) {
		store.ExpectedCalls = nil

		router := setupTestRouter()
		router.Use(CreateSession(store, cfg))

		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		store.AssertNotCalled(t, "Create")
		store.AssertNotCalled(t, "GetUserSessions")
	})

	t.Run("session_exists", func(t *testing.T) {
		store.ExpectedCalls = nil

		claims := &domain.Claims{
			UserID:      "test-user",
			Role:        domain.RoleUser,
			Permissions: []domain.Permission{domain.PermissionReadTrack},
		}

		existingSession := &domain.Session{
			ID:          "existing-session",
			UserID:      claims.UserID,
			Role:        claims.Role,
			Permissions: claims.Permissions,
			ExpiresAt:   time.Now().Add(time.Hour),
			CreatedAt:   time.Now(),
			LastSeenAt:  time.Now(),
		}

		router := setupTestRouter()

		// First middleware to set claims
		router.Use(func(c *gin.Context) {
			c.Set("claims", claims)
			c.Next()
		})

		// Then the CreateSession middleware
		router.Use(CreateSession(store, cfg))

		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:  cfg.CookieName,
			Value: existingSession.ID,
		})

		store.On("Get", mock.Anything, existingSession.ID).Return(existingSession, nil).Once()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)

		// Verify no new session cookie was set
		cookies := w.Result().Cookies()
		for _, cookie := range cookies {
			assert.NotEqual(t, cfg.CookieName, cookie.Name, "Should not set a new session cookie")
		}
	})

	t.Run("handles_session_limit", func(t *testing.T) {
		store.ExpectedCalls = nil

		claims := &domain.Claims{
			UserID:      "test-user",
			Role:        domain.RoleUser,
			Permissions: []domain.Permission{domain.PermissionReadTrack},
		}

		// Create existing sessions
		existingSessions := make([]*domain.Session, cfg.MaxSessionsPerUser)
		now := time.Now()
		for i := 0; i < cfg.MaxSessionsPerUser; i++ {
			existingSessions[i] = &domain.Session{
				ID:         fmt.Sprintf("session-%d", i),
				UserID:     claims.UserID,
				CreatedAt:  now.Add(time.Duration(i) * time.Hour), // Different creation times
				LastSeenAt: now,
				ExpiresAt:  now.Add(24 * time.Hour),
			}
		}

		// Set up mock expectations
		store.On("GetUserSessions", mock.Anything, claims.UserID).Return(existingSessions, nil).Once()
		store.On("Delete", mock.Anything, existingSessions[0].ID).Return(nil).Once() // Should delete oldest session
		store.On("Create", mock.Anything, mock.MatchedBy(func(s *domain.Session) bool {
			return s.UserID == claims.UserID &&
				s.Role == claims.Role &&
				len(s.Permissions) == len(claims.Permissions)
		})).Return(nil).Once()

		router := setupTestRouter()

		// First middleware to set claims
		router.Use(func(c *gin.Context) {
			c.Set("claims", claims)
			c.Next()
		})

		// Then the CreateSession middleware
		router.Use(CreateSession(store, cfg))

		router.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		store.AssertExpectations(t)

		// Verify new session cookie was set
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == cfg.CookieName {
				sessionCookie = cookie
				break
			}
		}
		assert.NotNil(t, sessionCookie)
		assert.NotEmpty(t, sessionCookie.Value)
		assert.NotEqual(t, existingSessions[0].ID, sessionCookie.Value, "New session ID should be different from deleted session")
	})
}

func TestClearSession_Middleware(t *testing.T) {
	sessionStore := new(MockSessionStore)
	cfg := pkgdomain.SessionConfig{
		CookieName:     "session_id",
		CookiePath:     "/",
		CookieDomain:   "localhost",
		CookieSecure:   true,
		CookieHTTPOnly: true,
	}

	tests := []struct {
		name           string
		setupMocks     func()
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "clear existing session",
			setupMocks: func() {
				sessionStore.ExpectedCalls = nil
				sessionStore.On("Get", mock.Anything, "test-session").Return(&pkgdomain.Session{
					ID: "test-session",
				}, nil)
				sessionStore.On("Delete", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:     cfg.CookieName,
					Value:    "test-session",
					Path:     cfg.CookiePath,
					Domain:   cfg.CookieDomain,
					Secure:   cfg.CookieSecure,
					HttpOnly: cfg.CookieHTTPOnly,
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "no existing session",
			setupMocks: func() {
				sessionStore.ExpectedCalls = nil
			},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusOK,
		},
		{
			name: "session not found",
			setupMocks: func() {
				sessionStore.ExpectedCalls = nil
				sessionStore.On("Get", mock.Anything, "test-session").Return(nil, pkgdomain.ErrSessionNotFound)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:     cfg.CookieName,
					Value:    "test-session",
					Path:     cfg.CookiePath,
					Domain:   cfg.CookieDomain,
					Secure:   cfg.CookieSecure,
					HttpOnly: cfg.CookieHTTPOnly,
				})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "handle delete error",
			setupMocks: func() {
				sessionStore.ExpectedCalls = nil
				sessionStore.On("Get", mock.Anything, "test-session").Return(&pkgdomain.Session{
					ID: "test-session",
				}, nil)
				sessionStore.On("Delete", mock.Anything, "test-session").Return(errors.New("delete error"))
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:     cfg.CookieName,
					Value:    "test-session",
					Path:     cfg.CookiePath,
					Domain:   cfg.CookieDomain,
					Secure:   cfg.CookieSecure,
					HttpOnly: cfg.CookieHTTPOnly,
				})
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to delete session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			c, w := setupTestContext()

			// Create request
			req, _ := http.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)
			c.Request = req

			// Setup mocks
			tt.setupMocks()

			// Run middleware
			ClearSession(sessionStore, cfg)(c)

			// Run handler only if not aborted
			if !c.IsAborted() {
				c.Status(http.StatusOK)
			}

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// For error cases, check error response
			if tt.expectedStatus >= 400 {
				var response struct {
					Error string `json:"error"`
				}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			}

			// Verify cookie was cleared
			var found bool
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == cfg.CookieName {
					found = true
					assert.Equal(t, -1, cookie.MaxAge)
					assert.Equal(t, cfg.CookiePath, cookie.Path)
					assert.Equal(t, cfg.CookieDomain, cookie.Domain)
					assert.Equal(t, cfg.CookieSecure, cookie.Secure)
					assert.Equal(t, cfg.CookieHTTPOnly, cookie.HttpOnly)
				}
			}
			assert.True(t, found, "session cookie should be cleared")

			// Verify mocks
			sessionStore.AssertExpectations(t)
		})
	}
}

// testResponseWriter is a custom ResponseWriter that implements all required methods
type testResponseWriter struct {
	*httptest.ResponseRecorder
	size   int
	status int
}

func newTestResponseWriter() *testResponseWriter {
	return &testResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		size:             0,
		status:           0,
	}
}

func (w *testResponseWriter) CloseNotify() <-chan bool {
	return make(chan bool, 1)
}

func (w *testResponseWriter) Status() int {
	return w.status
}

func (w *testResponseWriter) Size() int {
	return w.size
}

func (w *testResponseWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseRecorder.WriteString(s)
	w.size += n
	return n, err
}

func (w *testResponseWriter) Written() bool {
	return w.size != 0
}

func (w *testResponseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.WriteHeader(w.status)
	}
}

func (w *testResponseWriter) Pusher() http.Pusher {
	return nil
}

func (w *testResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

func (w *testResponseWriter) Flush() {
	w.ResponseRecorder.Flush()
}

func (w *testResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseRecorder.WriteHeader(code)
}

func TestRequireSession_Middleware(t *testing.T) {
	store := new(MockSessionStore)
	config := pkgdomain.SessionConfig{
		CookieName:     "session_id",
		CookiePath:     "/",
		CookieDomain:   "localhost",
		CookieSecure:   true,
		CookieHTTPOnly: true,
	}

	tests := []struct {
		name           string
		setupMocks     func()
		setupRequest   func(*http.Request)
		expectedStatus int
		expectedError  string
		checkContext   func(*testing.T, *gin.Context)
	}{
		{
			name: "valid_session",
			setupMocks: func() {
				store.ExpectedCalls = nil
				session := &domain.Session{
					ID:          "test-session",
					UserID:      "test-user",
					Role:        domain.RoleUser,
					Permissions: []domain.Permission{domain.PermissionReadTrack},
					ExpiresAt:   time.Now().Add(24 * time.Hour),
					CreatedAt:   time.Now(),
					LastSeenAt:  time.Now(),
				}
				store.On("Get", mock.Anything, "test-session").Return(session, nil)
				store.On("Touch", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "test-session",
				})
			},
			expectedStatus: http.StatusOK,
			checkContext: func(t *testing.T, c *gin.Context) {
				session, exists := c.Get("session")
				assert.True(t, exists)
				assert.NotNil(t, session)
			},
		},
		{
			name:           "no_session",
			setupMocks:     func() {},
			setupRequest:   func(req *http.Request) {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "No active session",
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("session")
				assert.False(t, exists)
			},
		},
		{
			name: "session_expired",
			setupMocks: func() {
				store.ExpectedCalls = nil
				session := &domain.Session{
					ID:          "test-session",
					UserID:      "test-user",
					Role:        domain.RoleUser,
					Permissions: []domain.Permission{domain.PermissionReadTrack},
					ExpiresAt:   time.Now().Add(-24 * time.Hour), // Expired
					CreatedAt:   time.Now().Add(-48 * time.Hour),
					LastSeenAt:  time.Now().Add(-24 * time.Hour),
				}
				store.On("Get", mock.Anything, "test-session").Return(session, nil)
				store.On("Delete", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "test-session",
				})
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Session expired",
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("session")
				assert.False(t, exists)
				_, exists = c.Get("session_id")
				assert.False(t, exists)
				_, exists = c.Get("user_id")
				assert.False(t, exists)
				_, exists = c.Get("role")
				assert.False(t, exists)
				_, exists = c.Get("permissions")
				assert.False(t, exists)
			},
		},
		{
			name: "session_store_error",
			setupMocks: func() {
				store.ExpectedCalls = nil
				store.On("Get", mock.Anything, "test-session").Return(nil, errors.New("store error"))
				store.On("Delete", mock.Anything, "test-session").Return(nil)
			},
			setupRequest: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: "test-session",
				})
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to retrieve session",
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("session")
				assert.False(t, exists)
				_, exists = c.Get("session_id")
				assert.False(t, exists)
				_, exists = c.Get("user_id")
				assert.False(t, exists)
				_, exists = c.Get("role")
				assert.False(t, exists)
				_, exists = c.Get("permissions")
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			c, w := setupTestContext()

			// Create request
			req, _ := http.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)
			c.Request = req

			// Setup mocks
			tt.setupMocks()

			// Run middleware
			Session(store, config)(c)
			RequireSession(store)(c)

			// Run handler only if not aborted
			if !c.IsAborted() {
				c.Status(http.StatusOK)
			}

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// For error cases, check error response
			if tt.expectedStatus >= 400 {
				var response struct {
					Error string `json:"error"`
				}
				err := json.NewDecoder(w.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedError, response.Error)
			}

			// Check context
			tt.checkContext(t, c)

			// Verify mocks
			store.AssertExpectations(t)
		})
	}
}
