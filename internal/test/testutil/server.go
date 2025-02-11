package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"metadatatool/internal/domain"
	"metadatatool/internal/handler"
	"metadatatool/internal/pkg/converter"
	"metadatatool/internal/repository/base"
	"metadatatool/internal/usecase"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestServer represents a test server instance
type TestServer struct {
	Router       *gin.Engine
	AuthHandler  *handler.AuthHandler
	SessionStore domain.SessionStore
	AuthService  domain.AuthService
	UserRepo     domain.UserRepository
	cleanup      func()
}

// NewTestServer creates a new test server with all necessary dependencies
func NewTestServer(t *testing.T) *TestServer {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Initialize repositories and wrappers
	pkgUserRepo := NewInMemoryPkgUserRepository()
	pkgSessionStore := NewInMemoryPkgSessionStore()

	userRepoWrapper := converter.NewUserRepositoryWrapper(base.NewInMemoryUserRepository(), pkgUserRepo)
	sessionStoreWrapper := converter.NewSessionStoreWrapper(base.NewInMemorySessionRepository(), pkgSessionStore)

	// Initialize services and handlers
	authService := base.NewInMemoryAuthService()
	userUseCase := usecase.NewUserUseCase(userRepoWrapper)
	authUseCase := usecase.NewAuthUseCase(userRepoWrapper.Internal(), sessionStoreWrapper.Internal(), authService)
	authHandler := handler.NewAuthHandler(authUseCase, userUseCase, sessionStoreWrapper.Internal())

	// Setup routes
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)
	router.POST("/auth/logout", authHandler.Logout)
	router.POST("/auth/token/refresh", authHandler.RefreshToken)
	router.POST("/auth/api-key", authHandler.GenerateAPIKey)

	return &TestServer{
		Router:       router,
		AuthHandler:  authHandler,
		SessionStore: sessionStoreWrapper.Internal(),
		AuthService:  authService,
		UserRepo:     userRepoWrapper.Internal(),
		cleanup:      func() {},
	}
}

// Close cleans up resources
func (ts *TestServer) Close() {
	if ts.cleanup != nil {
		ts.cleanup()
	}
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

// CreateTestUser creates a test user and returns the user object
func (ts *TestServer) CreateTestUser(t *testing.T, email, password string, role domain.Role) *domain.User {
	// Hash password
	hashedPassword, err := ts.AuthService.HashPassword(password)
	require.NoError(t, err)

	// Create user
	user := &domain.User{
		Email:    email,
		Password: hashedPassword,
		Role:     role,
		Name:     "Test User",
	}

	err = ts.UserRepo.Create(context.Background(), user)
	require.NoError(t, err)

	return user
}

// GetAuthToken performs login and returns the auth token
func (ts *TestServer) GetAuthToken(t *testing.T, email, password string) string {
	// Create user if it doesn't exist
	user := ts.CreateTestUser(t, email, password, domain.RoleUser)

	// Generate token
	token, err := ts.AuthService.GenerateToken(context.Background(), user)
	require.NoError(t, err)

	return token
}

// GetAuthHeader returns the Authorization header with the token
func (ts *TestServer) GetAuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token),
	}
}

// RequireStatus asserts that the response has the expected status code
func (ts *TestServer) RequireStatus(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	require.Equal(t, expectedStatus, w.Code, "unexpected status code: got %d, want %d", w.Code, expectedStatus)
}

// RequireOK asserts that the response has status 200 OK
func (ts *TestServer) RequireOK(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusOK)
}

// RequireCreated asserts that the response has status 201 Created
func (ts *TestServer) RequireCreated(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusCreated)
}

// RequireNoContent asserts that the response has status 204 No Content
func (ts *TestServer) RequireNoContent(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusNoContent)
}

// RequireBadRequest asserts that the response has status 400 Bad Request
func (ts *TestServer) RequireBadRequest(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusBadRequest)
}

// RequireUnauthorized asserts that the response has status 401 Unauthorized
func (ts *TestServer) RequireUnauthorized(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusUnauthorized)
}

// RequireForbidden asserts that the response has status 403 Forbidden
func (ts *TestServer) RequireForbidden(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusForbidden)
}

// RequireNotFound asserts that the response has status 404 Not Found
func (ts *TestServer) RequireNotFound(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusNotFound)
}

// RequireConflict asserts that the response has status 409 Conflict
func (ts *TestServer) RequireConflict(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusConflict)
}

// RequireInternalServerError asserts that the response has status 500 Internal Server Error
func (ts *TestServer) RequireInternalServerError(t *testing.T, w *httptest.ResponseRecorder) {
	ts.RequireStatus(t, w, http.StatusInternalServerError)
}
