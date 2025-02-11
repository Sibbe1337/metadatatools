package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"testing"

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

func (m *MockUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	args := m.Called(ctx, userID, apiKey)
	return args.Error(0)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
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

func (m *MockSessionStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Session), args.Error(1)
}

func (m *MockSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockSessionStore) Touch(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

// MockAuthService is a mock implementation of domain.AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) GenerateToken(ctx context.Context, user *domain.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*domain.TokenClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenClaims), args.Error(1)
}

func (m *MockAuthService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) VerifyPassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

func (m *MockAuthService) GenerateTokens(user *domain.User) (*domain.Tokens, error) {
	args := m.Called(user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tokens), args.Error(1)
}

func (m *MockAuthService) GenerateAPIKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func setupAuthService() (*AuthUseCase, *MockUserRepository) {
	userRepo := new(MockUserRepository)
	sessionRepo := new(MockSessionStore)
	authService := new(MockAuthService)

	useCase := NewAuthUseCase(userRepo, sessionRepo, authService)
	return useCase, userRepo
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

func TestAuthUseCase_GenerateToken(t *testing.T) {
	useCase, _ := setupAuthService()

	t.Run("successful token generation", func(t *testing.T) {
		user := createTestUser()
		token := "test-access-token"

		useCase.authService.(*MockAuthService).On("GenerateToken", mock.Anything, user).Return(token, nil)

		result, err := useCase.authService.GenerateToken(context.Background(), user)
		require.NoError(t, err)
		assert.Equal(t, token, result)
	})
}

func TestAuthUseCase_ValidateToken(t *testing.T) {
	useCase, userRepo := setupAuthService()
	user := createTestUser()

	t.Run("valid token", func(t *testing.T) {
		claims := &domain.TokenClaims{
			Claims: domain.Claims{
				UserID: user.ID,
				Role:   user.Role,
			},
		}

		useCase.authService.(*MockAuthService).On("ValidateToken", mock.Anything, "test-token").Return(claims, nil)
		userRepo.On("GetByID", mock.Anything, user.ID).Return(user, nil)

		result, err := useCase.ValidateToken(context.Background(), "test-token")
		require.NoError(t, err)
		assert.Equal(t, user, result)
	})

	t.Run("invalid token", func(t *testing.T) {
		useCase.authService.(*MockAuthService).On("ValidateToken", mock.Anything, "invalid-token").Return(nil, fmt.Errorf("invalid token"))

		result, err := useCase.ValidateToken(context.Background(), "invalid-token")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestAuthUseCase_HashPassword(t *testing.T) {
	useCase, _ := setupAuthService()

	t.Run("successful password hashing", func(t *testing.T) {
		password := "test-password"
		hashedPassword := "hashed-password"

		useCase.authService.(*MockAuthService).On("HashPassword", password).Return(hashedPassword, nil)

		result, err := useCase.authService.HashPassword(password)
		require.NoError(t, err)
		assert.Equal(t, hashedPassword, result)
	})

	t.Run("empty password", func(t *testing.T) {
		useCase.authService.(*MockAuthService).On("HashPassword", "").Return("", fmt.Errorf("invalid password"))

		result, err := useCase.authService.HashPassword("")
		assert.Error(t, err)
		assert.Empty(t, result)
	})
}

func TestAuthUseCase_VerifyPassword(t *testing.T) {
	useCase, _ := setupAuthService()

	t.Run("correct password", func(t *testing.T) {
		hashedPassword := "hashed-password"
		password := "test-password"

		useCase.authService.(*MockAuthService).On("VerifyPassword", hashedPassword, password).Return(nil)

		err := useCase.authService.VerifyPassword(hashedPassword, password)
		assert.NoError(t, err)
	})

	t.Run("incorrect password", func(t *testing.T) {
		hashedPassword := "hashed-password"

		useCase.authService.(*MockAuthService).On("VerifyPassword", hashedPassword, "wrong-password").Return(fmt.Errorf("invalid password"))

		err := useCase.authService.VerifyPassword(hashedPassword, "wrong-password")
		assert.Error(t, err)
	})
}

func TestAuthUseCase_HasPermission(t *testing.T) {
	useCase, _ := setupAuthService()

	testCases := []struct {
		name       string
		role       pkgdomain.Role
		permission pkgdomain.Permission
		hasAccess  bool
	}{
		{
			name:       "admin has all permissions",
			role:       pkgdomain.RoleAdmin,
			permission: pkgdomain.PermissionCreateTrack,
			hasAccess:  true,
		},
		{
			name:       "user has basic permissions",
			role:       pkgdomain.RoleUser,
			permission: pkgdomain.PermissionReadTrack,
			hasAccess:  true,
		},
		{
			name:       "guest has limited permissions",
			role:       pkgdomain.RoleGuest,
			permission: pkgdomain.PermissionCreateTrack,
			hasAccess:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user := &domain.User{
				Role: domain.Role(tc.role),
			}
			hasAccess := useCase.HasPermission(user, tc.permission)
			assert.Equal(t, tc.hasAccess, hasAccess)
		})
	}
}
