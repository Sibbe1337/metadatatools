package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"metadatatool/internal/pkg/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockUserRepository is a mock implementation of domain.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*domain.User), args.Error(1)
}

// MockAuthService is a mock implementation of domain.AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GenerateTokens(user *domain.User) (*domain.TokenPair, error) {
	args := m.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *MockAuthService) ValidateToken(token string) (*domain.Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Claims), args.Error(1)
}

func (m *MockAuthService) RefreshToken(refreshToken string) (*domain.TokenPair, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenPair), args.Error(1)
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyPassword(hash, password string) error {
	args := m.Called(hash, password)
	return args.Error(0)
}

func (m *MockAuthService) HasPermission(role domain.Role, permission domain.Permission) bool {
	args := m.Called(role, permission)
	return args.Bool(0)
}

func (m *MockAuthService) GetPermissions(role domain.Role) []domain.Permission {
	args := m.Called(role)
	return args.Get(0).([]domain.Permission)
}

func (m *MockAuthService) GenerateAPIKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

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

func setupAuthHandler() (*AuthHandler, *MockAuthService, *MockUserRepository, *MockSessionStore, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	authService := new(MockAuthService)
	userRepo := new(MockUserRepository)
	sessionStore := new(MockSessionStore)
	handler := NewAuthHandler(authService, userRepo, sessionStore)

	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/refresh", handler.RefreshToken)
	router.POST("/auth/logout", handler.Logout)
	router.POST("/auth/api-key", handler.GenerateAPIKey)

	return handler, authService, userRepo, sessionStore, router
}

func TestAuthHandler_Register(t *testing.T) {
	_, authService, userRepo, _, router := setupAuthHandler()

	t.Run("successful registration", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
			"name":     "Test User",
		}
		body, _ := json.Marshal(reqBody)

		hashedPassword := "hashed-password"
		authService.On("HashPassword", "password123").Return(hashedPassword, nil)
		userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, nil)
		userRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.Email == "test@example.com" && u.Password == hashedPassword
		})).Return(nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		authService.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "existing@example.com",
			"password": "password123",
			"name":     "Test User",
		}
		body, _ := json.Marshal(reqBody)

		existingUser := &domain.User{
			Email: "existing@example.com",
		}
		userRepo.On("GetByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		userRepo.AssertExpectations(t)
	})

	t.Run("invalid request body", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email": "invalid-email",
		}
		body, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAuthHandler_Login(t *testing.T) {
	_, authService, userRepo, _, router := setupAuthHandler()

	t.Run("successful login", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(reqBody)

		user := &domain.User{
			ID:       "user-id",
			Email:    "test@example.com",
			Password: "hashed-password",
			Role:     domain.RoleUser,
		}

		tokens := &domain.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}

		userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(user, nil)
		authService.On("VerifyPassword", user.Password, "password123").Return(nil)
		authService.On("GenerateTokens", user).Return(tokens, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, tokens.AccessToken, response["access_token"])
		assert.Equal(t, tokens.RefreshToken, response["refresh_token"])

		authService.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "test@example.com",
			"password": "wrong-password",
		}
		body, _ := json.Marshal(reqBody)

		user := &domain.User{
			Email:    "test@example.com",
			Password: "hashed-password",
		}

		userRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(user, nil)
		authService.On("VerifyPassword", user.Password, "wrong-password").Return(domain.ErrInvalidCredentials)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		authService.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"email":    "nonexistent@example.com",
			"password": "password123",
		}
		body, _ := json.Marshal(reqBody)

		userRepo.On("GetByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		userRepo.AssertExpectations(t)
	})
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	_, authService, _, _, router := setupAuthHandler()

	t.Run("successful token refresh", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"refresh_token": "valid-refresh-token",
		}
		body, _ := json.Marshal(reqBody)

		newTokens := &domain.TokenPair{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
		}

		authService.On("RefreshToken", "valid-refresh-token").Return(newTokens, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, newTokens.AccessToken, response["access_token"])
		assert.Equal(t, newTokens.RefreshToken, response["refresh_token"])

		authService.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"refresh_token": "invalid-refresh-token",
		}
		body, _ := json.Marshal(reqBody)

		authService.On("RefreshToken", "invalid-refresh-token").Return(nil, domain.ErrInvalidToken)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/refresh", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		authService.AssertExpectations(t)
	})
}

func TestAuthHandler_GenerateAPIKey(t *testing.T) {
	_, authService, userRepo, _, router := setupAuthHandler()

	t.Run("successful API key generation", func(t *testing.T) {
		apiKey := "new-api-key"
		user := &domain.User{
			ID:    "user-id",
			Email: "test@example.com",
			Role:  domain.RoleUser,
		}

		authService.On("GenerateAPIKey").Return(apiKey, nil)
		userRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
			return u.ID == user.ID && u.APIKey == apiKey
		})).Return(nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/api-key", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set user in context
		ctx := req.Context()
		ctx = domain.WithUser(ctx, user)
		req = req.WithContext(ctx)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, apiKey, response["api_key"])

		authService.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("error generating API key", func(t *testing.T) {
		user := &domain.User{
			ID:    "user-id",
			Email: "test@example.com",
			Role:  domain.RoleUser,
		}

		authService.On("GenerateAPIKey").Return("", domain.ErrInternal)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/api-key", nil)
		req.Header.Set("Content-Type", "application/json")

		// Set user in context
		ctx := req.Context()
		ctx = domain.WithUser(ctx, user)
		req = req.WithContext(ctx)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		authService.AssertExpectations(t)
	})
}
