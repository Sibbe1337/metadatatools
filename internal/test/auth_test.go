package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"metadatatool/internal/domain"
	"metadatatool/internal/handler"
	"metadatatool/internal/handler/middleware"
	"metadatatool/internal/pkg/converter"
	pkgdomain "metadatatool/internal/pkg/domain"
	"metadatatool/internal/repository/base"
	"metadatatool/internal/test/testutil"
	"metadatatool/internal/usecase"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestServer represents a test server instance
type TestServer struct {
	Router       *gin.Engine
	AuthHandler  *handler.AuthHandler
	SessionStore pkgdomain.SessionStore
	AuthService  pkgdomain.AuthService
	UserRepo     pkgdomain.UserRepository
	cleanup      func()
}

// NewTestServer creates a new test server with all necessary dependencies
func NewTestServer(t *testing.T) *TestServer {
	// Setup internal repositories
	internalUserRepo := base.NewInMemoryUserRepository()
	internalSessionStore := base.NewInMemorySessionRepository()
	internalAuthService := base.NewInMemoryAuthService()

	// Setup pkg repositories
	pkgUserRepo := testutil.NewInMemoryPkgUserRepository()
	pkgSessionStore := testutil.NewInMemoryPkgSessionStore()
	pkgAuthService := testutil.NewInMemoryPkgAuthService()

	// Create wrappers
	userRepoWrapper := converter.NewUserRepositoryWrapper(internalUserRepo, pkgUserRepo)
	sessionStoreWrapper := converter.NewSessionStoreWrapper(internalSessionStore, pkgSessionStore)
	authServiceWrapper := converter.NewAuthServiceWrapper(internalAuthService, pkgAuthService)

	// Setup session config
	sessionConfig := pkgdomain.SessionConfig{
		SessionDuration:    24 * time.Hour,
		CleanupInterval:    time.Hour,
		MaxSessionsPerUser: 5,
		CookieName:         "session_id",
		CookiePath:         "/",
		CookieDomain:       "localhost",
		CookieSecure:       true,
		CookieHTTPOnly:     true,
	}

	// Initialize usecases
	authUseCase := usecase.NewAuthUseCase(userRepoWrapper.Internal(), sessionStoreWrapper.Internal(), authServiceWrapper.Internal())
	userUseCase := usecase.NewUserUseCase(userRepoWrapper.Pkg())

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUseCase, userUseCase, sessionStoreWrapper.Internal())

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Auth routes
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", middleware.CreateSession(sessionStoreWrapper.Pkg(), sessionConfig), authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)

		// Protected auth routes
		protected := auth.Group("")
		protected.Use(middleware.Session(sessionStoreWrapper.Pkg(), sessionConfig))
		protected.Use(middleware.RequireSession(sessionStoreWrapper.Pkg()))
		{
			protected.POST("/logout", middleware.ClearSession(sessionStoreWrapper.Pkg(), sessionConfig), authHandler.Logout)
			protected.POST("/apikey", authHandler.GenerateAPIKey)
			protected.GET("/sessions", authHandler.GetActiveSessions)
			protected.DELETE("/sessions/:id", authHandler.RevokeSession)
			protected.DELETE("/sessions", authHandler.RevokeAllSessions)
		}
	}

	// Protected routes for testing
	api := router.Group("/api/v1")
	{
		tracks := api.Group("/tracks")
		{
			tracks.GET("", func(c *gin.Context) { c.Status(http.StatusOK) })

			authenticated := tracks.Group("")
			authenticated.Use(middleware.Session(sessionStoreWrapper.Pkg(), sessionConfig))
			authenticated.Use(middleware.RequireSession(sessionStoreWrapper.Pkg()))
			{
				authenticated.POST("", func(c *gin.Context) { c.Status(http.StatusOK) })
				authenticated.PUT("/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
				authenticated.DELETE("/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

				admin := authenticated.Group("")
				admin.Use(middleware.RequireRole(pkgdomain.RoleAdmin))
				{
					admin.POST("/batch", func(c *gin.Context) { c.Status(http.StatusOK) })
					admin.POST("/export", func(c *gin.Context) { c.Status(http.StatusOK) })
				}
			}
		}
	}

	// Create test server
	ts := &TestServer{
		Router:       router,
		AuthHandler:  authHandler,
		SessionStore: sessionStoreWrapper.Pkg(),
		UserRepo:     userRepoWrapper.Pkg(),
		AuthService:  authServiceWrapper.Pkg(),
		cleanup:      func() {},
	}

	return ts
}

// Close cleans up resources
func (ts *TestServer) Close() {
	if ts.cleanup != nil {
		ts.cleanup()
	}
}

// Cleanup is an alias for Close to match testify's cleanup convention
func (ts *TestServer) Cleanup() {
	ts.Close()
}

// MakeRequest is a helper to make HTTP requests in tests
func (ts *TestServer) MakeRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal request body: %v", err))
		}
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	ts.Router.ServeHTTP(w, req)
	return w
}

// GetAuthHeader returns the Authorization header with the token
func (ts *TestServer) GetAuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}

func TestAuthFlow(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Test user registration
	user := ts.CreateTestUser(t, "test@example.com", "password123", domain.RoleUser)
	require.NotEmpty(t, user.ID)

	// Test login
	loginReq := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	resp := ts.MakeRequest(http.MethodPost, "/api/v1/auth/login", loginReq, nil)
	require.Equal(t, http.StatusOK, resp.Code)

	// Extract session cookie
	cookies := resp.Result().Cookies()
	require.NotEmpty(t, cookies)

	// Test protected route access
	headers := map[string]string{
		"Cookie": cookies[0].String(),
	}
	resp = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)

	// Test logout
	resp = ts.MakeRequest(http.MethodPost, "/api/v1/auth/logout", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)

	// Verify session is cleared
	resp = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil, headers)
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestAdminRoutes(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create admin user
	adminUser := ts.CreateTestUser(t, "admin@example.com", "admin123", domain.RoleAdmin)
	require.NotEmpty(t, adminUser.ID)

	// Login as admin
	loginReq := map[string]string{
		"email":    "admin@example.com",
		"password": "admin123",
	}
	resp := ts.MakeRequest(http.MethodPost, "/api/v1/auth/login", loginReq, nil)
	require.Equal(t, http.StatusOK, resp.Code)

	// Extract session cookie
	cookies := resp.Result().Cookies()
	require.NotEmpty(t, cookies)
	headers := map[string]string{
		"Cookie": cookies[0].String(),
	}

	// Test admin-only routes
	resp = ts.MakeRequest(http.MethodPost, "/api/v1/tracks/batch", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestRegularUserRoutes(t *testing.T) {
	ts := testutil.NewTestServer(t)
	defer ts.Close()

	// Create regular user
	regularUser := ts.CreateTestUser(t, "user@example.com", "user123", domain.RoleUser)
	require.NotEmpty(t, regularUser.ID)

	// Login as regular user
	loginReq := map[string]string{
		"email":    "user@example.com",
		"password": "user123",
	}
	resp := ts.MakeRequest(http.MethodPost, "/api/v1/auth/login", loginReq, nil)
	require.Equal(t, http.StatusOK, resp.Code)

	// Extract session cookie
	cookies := resp.Result().Cookies()
	require.NotEmpty(t, cookies)
	headers := map[string]string{
		"Cookie": cookies[0].String(),
	}

	// Test regular user routes
	resp = ts.MakeRequest(http.MethodGet, "/api/v1/tracks", nil, headers)
	require.Equal(t, http.StatusOK, resp.Code)

	// Test admin-only routes (should fail)
	resp = ts.MakeRequest(http.MethodPost, "/api/v1/tracks/batch", nil, headers)
	require.Equal(t, http.StatusForbidden, resp.Code)
}
