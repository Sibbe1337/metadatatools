package middleware

import (
	"context"
	pkgdomain "metadatatool/internal/pkg/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of pkgdomain.AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GenerateTokens(user *pkgdomain.User) (*pkgdomain.TokenPair, error) {
	args := m.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.TokenPair), args.Error(1)
}

func (m *MockAuthService) ValidateToken(token string) (*pkgdomain.Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.Claims), args.Error(1)
}

func (m *MockAuthService) RefreshToken(refreshToken string) (*pkgdomain.TokenPair, error) {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.TokenPair), args.Error(1)
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyPassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

func (m *MockAuthService) GenerateAPIKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) HasPermission(role pkgdomain.Role, permission pkgdomain.Permission) bool {
	args := m.Called(role, permission)
	return args.Bool(0)
}

func (m *MockAuthService) GetPermissions(role pkgdomain.Role) []pkgdomain.Permission {
	args := m.Called(role)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]pkgdomain.Permission)
}

// MockUserRepository is a mock implementation of pkgdomain.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *pkgdomain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*pkgdomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*pkgdomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.User), args.Error(1)
}

func (m *MockUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*pkgdomain.User, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *pkgdomain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*pkgdomain.User, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*pkgdomain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	args := m.Called(ctx, userID, apiKey)
	return args.Error(0)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestAuth_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authService := new(MockAuthService)

	tests := []struct {
		name           string
		setupAuth      func()
		token          string
		expectedStatus int
	}{
		{
			name: "valid token",
			setupAuth: func() {
				claims := &pkgdomain.Claims{
					UserID:      "test-user",
					Role:        pkgdomain.RoleUser,
					Permissions: []pkgdomain.Permission{pkgdomain.PermissionReadTrack},
				}
				authService.On("ValidateToken", "valid-token").Return(claims, nil)
			},
			token:          "valid-token",
			expectedStatus: http.StatusOK,
		},
		{
			name: "missing token",
			setupAuth: func() {
				// No setup needed
			},
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid token",
			setupAuth: func() {
				authService.On("ValidateToken", "invalid-token").Return(nil, pkgdomain.ErrInvalidToken)
			},
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			authService.ExpectedCalls = nil

			// Setup auth mock
			if tt.setupAuth != nil {
				tt.setupAuth()
			}

			// Create test router
			router := gin.New()
			router.Use(Auth(authService))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Create request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify mock expectations
			authService.AssertExpectations(t)
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		requiredRole   pkgdomain.Role
		userRole       pkgdomain.Role
		expectedStatus int
	}{
		{
			name:           "matching_role",
			requiredRole:   pkgdomain.RoleUser,
			userRole:       pkgdomain.RoleUser,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "admin_can_access_any_role",
			requiredRole:   pkgdomain.RoleUser,
			userRole:       pkgdomain.RoleAdmin,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "insufficient_role",
			requiredRole:   pkgdomain.RoleAdmin,
			userRole:       pkgdomain.RoleUser,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing_role",
			requiredRole:   pkgdomain.RoleUser,
			userRole:       pkgdomain.Role(""),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.userRole != "" {
					c.Set("role", tt.userRole)
				}
				c.Next()
			})
			router.Use(RequireRole(tt.requiredRole))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		permission     pkgdomain.Permission
		userClaims     *pkgdomain.Claims
		expectedStatus int
	}{
		{
			name:       "has_permission",
			permission: pkgdomain.PermissionReadTrack,
			userClaims: &pkgdomain.Claims{
				UserID:      "test-user",
				Role:        pkgdomain.RoleUser,
				Permissions: []pkgdomain.Permission{pkgdomain.PermissionReadTrack},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "missing_permission",
			permission: pkgdomain.PermissionReadTrack,
			userClaims: &pkgdomain.Claims{
				UserID:      "test-user",
				Role:        pkgdomain.RoleUser,
				Permissions: []pkgdomain.Permission{pkgdomain.PermissionCreateTrack},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no_permissions",
			permission:     pkgdomain.PermissionReadTrack,
			userClaims:     nil,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tt.userClaims != nil {
					c.Set("claims", tt.userClaims)
				}
				c.Next()
			})
			router.Use(RequirePermission(tt.permission))
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestAPIKeyAuth(t *testing.T) {
	userRepo := &MockUserRepository{}

	tests := []struct {
		name           string
		apiKey         string
		setupMocks     func()
		expectedStatus int
		checkContext   func(*testing.T, *gin.Context)
	}{
		{
			name:   "valid_api_key",
			apiKey: "valid-key",
			setupMocks: func() {
				user := &pkgdomain.User{
					ID:     "test-user",
					Role:   pkgdomain.RoleUser,
					APIKey: "valid-key",
				}
				userRepo.On("GetByAPIKey", mock.Anything, "valid-key").Return(user, nil)
			},
			expectedStatus: http.StatusOK,
			checkContext: func(t *testing.T, c *gin.Context) {
				user, exists := c.Get("user")
				assert.True(t, exists)
				u, ok := user.(*pkgdomain.User)
				assert.True(t, ok)
				assert.Equal(t, "test-user", u.ID)
				assert.Equal(t, pkgdomain.RoleUser, u.Role)
			},
		},
		{
			name:   "invalid_api_key",
			apiKey: "invalid-key",
			setupMocks: func() {
				userRepo.On("GetByAPIKey", mock.Anything, "invalid-key").Return(nil, nil)
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("user")
				assert.False(t, exists)
			},
		},
		{
			name:           "no_api_key",
			apiKey:         "",
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
			checkContext: func(t *testing.T, c *gin.Context) {
				_, exists := c.Get("user")
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			userRepo.ExpectedCalls = nil

			// Setup router with middleware
			router := gin.New()
			router.Use(APIKeyAuth(userRepo))
			router.GET("/test", func(c *gin.Context) {
				tt.checkContext(t, c)
				c.Status(http.StatusOK)
			})

			// Setup request
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}

			// Setup mocks
			tt.setupMocks()

			// Perform request
			router.ServeHTTP(w, req)

			// Assert response
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify all mock expectations were met
			userRepo.AssertExpectations(t)
		})
	}
}
