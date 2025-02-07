package usecase

import (
	"context"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func setupAuthService() (*authService, *MockUserRepository) {
	userRepo := new(MockUserRepository)
	cfg := &config.AuthConfig{
		JWTSecret:          "test-secret",
		AccessTokenExpiry:  time.Hour,
		RefreshTokenExpiry: 24 * time.Hour,
	}
	service := NewAuthService(cfg, userRepo).(*authService)
	return service, userRepo
}

func createTestUser() *domain.User {
	return &domain.User{
		ID:       "test-id",
		Email:    "test@example.com",
		Password: "hashed-password",
		Name:     "Test User",
		Role:     domain.RoleUser,
	}
}

func TestAuthService_GenerateTokens(t *testing.T) {
	service, _ := setupAuthService()

	t.Run("successful token generation", func(t *testing.T) {
		user := createTestUser()
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)

		// Verify access token
		claims, err := service.ValidateToken(tokens.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.NotEmpty(t, claims.Permissions)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	service, _ := setupAuthService()
	user := createTestUser()

	t.Run("valid token", func(t *testing.T) {
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)

		claims, err := service.ValidateToken(tokens.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create token that's already expired
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID,
			"email":   user.Email,
			"role":    user.Role,
			"exp":     time.Now().Add(-time.Hour).Unix(),
		})

		tokenString, err := token.SignedString([]byte(service.cfg.JWTSecret))
		require.NoError(t, err)

		claims, err := service.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("invalid token", func(t *testing.T) {
		claims, err := service.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	service, userRepo := setupAuthService()
	user := createTestUser()

	t.Run("successful refresh", func(t *testing.T) {
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)

		userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil)

		newTokens, err := service.RefreshToken(tokens.RefreshToken)
		assert.NoError(t, err)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEmpty(t, newTokens.RefreshToken)
		assert.NotEqual(t, tokens.AccessToken, newTokens.AccessToken)
		assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken)

		userRepo.AssertExpectations(t)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		newTokens, err := service.RefreshToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, newTokens)
	})

	t.Run("user not found", func(t *testing.T) {
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)

		userRepo.On("GetByID", mock.Anything, user.ID).Return(nil, nil)

		newTokens, err := service.RefreshToken(tokens.RefreshToken)
		assert.Error(t, err)
		assert.Nil(t, newTokens)

		userRepo.AssertExpectations(t)
	})
}

func TestAuthService_HashPassword(t *testing.T) {
	service, _ := setupAuthService()

	t.Run("successful password hashing", func(t *testing.T) {
		password := "test-password"
		hash, err := service.HashPassword(password)
		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash)

		// Verify password matches hash
		err = service.VerifyPassword(hash, password)
		assert.NoError(t, err)
	})

	t.Run("empty password", func(t *testing.T) {
		hash, err := service.HashPassword("")
		assert.Error(t, err)
		assert.Empty(t, hash)
	})
}

func TestAuthService_VerifyPassword(t *testing.T) {
	service, _ := setupAuthService()

	t.Run("correct password", func(t *testing.T) {
		password := "test-password"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, password)
		assert.NoError(t, err)
	})

	t.Run("incorrect password", func(t *testing.T) {
		password := "test-password"
		hash, err := service.HashPassword(password)
		require.NoError(t, err)

		err = service.VerifyPassword(hash, "wrong-password")
		assert.Error(t, err)
	})

	t.Run("invalid hash", func(t *testing.T) {
		err := service.VerifyPassword("invalid-hash", "test-password")
		assert.Error(t, err)
	})
}

func TestAuthService_HasPermission(t *testing.T) {
	service, _ := setupAuthService()

	testCases := []struct {
		name       string
		role       domain.Role
		permission domain.Permission
		hasAccess  bool
	}{
		{
			name:       "admin has all permissions",
			role:       domain.RoleAdmin,
			permission: domain.PermissionManageUsers,
			hasAccess:  true,
		},
		{
			name:       "user has basic permissions",
			role:       domain.RoleUser,
			permission: domain.PermissionReadTrack,
			hasAccess:  true,
		},
		{
			name:       "user cannot manage users",
			role:       domain.RoleUser,
			permission: domain.PermissionManageUsers,
			hasAccess:  false,
		},
		{
			name:       "guest can only read",
			role:       domain.RoleGuest,
			permission: domain.PermissionReadTrack,
			hasAccess:  true,
		},
		{
			name:       "guest cannot create",
			role:       domain.RoleGuest,
			permission: domain.PermissionCreateTrack,
			hasAccess:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasAccess := service.HasPermission(tc.role, tc.permission)
			assert.Equal(t, tc.hasAccess, hasAccess)
		})
	}
}

func TestAuthService_GetPermissions(t *testing.T) {
	service, _ := setupAuthService()

	testCases := []struct {
		name             string
		role             domain.Role
		expectedContains []domain.Permission
		notExpected      []domain.Permission
	}{
		{
			name: "admin permissions",
			role: domain.RoleAdmin,
			expectedContains: []domain.Permission{
				domain.PermissionManageUsers,
				domain.PermissionCreateTrack,
				domain.PermissionDeleteTrack,
			},
			notExpected: nil,
		},
		{
			name: "user permissions",
			role: domain.RoleUser,
			expectedContains: []domain.Permission{
				domain.PermissionCreateTrack,
				domain.PermissionReadTrack,
			},
			notExpected: []domain.Permission{
				domain.PermissionManageUsers,
				domain.PermissionManageRoles,
			},
		},
		{
			name: "guest permissions",
			role: domain.RoleGuest,
			expectedContains: []domain.Permission{
				domain.PermissionReadTrack,
			},
			notExpected: []domain.Permission{
				domain.PermissionCreateTrack,
				domain.PermissionManageUsers,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			permissions := service.GetPermissions(tc.role)

			// Check expected permissions are present
			for _, expected := range tc.expectedContains {
				assert.Contains(t, permissions, expected)
			}

			// Check unexpected permissions are not present
			for _, notExpected := range tc.notExpected {
				assert.NotContains(t, permissions, notExpected)
			}
		})
	}
}

func TestAuthService_GenerateAPIKey(t *testing.T) {
	service, _ := setupAuthService()

	t.Run("generate unique keys", func(t *testing.T) {
		key1, err := service.GenerateAPIKey()
		assert.NoError(t, err)
		assert.NotEmpty(t, key1)

		key2, err := service.GenerateAPIKey()
		assert.NoError(t, err)
		assert.NotEmpty(t, key2)

		assert.NotEqual(t, key1, key2)
	})
}
