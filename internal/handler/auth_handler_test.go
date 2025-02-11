package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"metadatatool/internal/domain"
	"metadatatool/internal/pkg/converter"
	pkgdomain "metadatatool/internal/pkg/domain"
	"metadatatool/internal/usecase"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// RequestKey is a custom type for request keys to avoid SA1029
type RequestKey string

// MockAuthService is a mock implementation of domain.AuthService
type MockAuthService struct {
	mock.Mock
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

func (m *MockAuthService) ValidateToken(ctx context.Context, token string) (*domain.TokenClaims, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TokenClaims), args.Error(1)
}

func (m *MockAuthService) GenerateAPIKey() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

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

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	args := m.Called(ctx, userID, apiKey)
	return args.Error(0)
}

// MockSessionStore is a mock implementation of domain.SessionStore
type MockSessionStore struct {
	mock.Mock
}

func (m *MockSessionStore) Create(ctx context.Context, session *pkgdomain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionStore) Get(ctx context.Context, id string) (*pkgdomain.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.Session), args.Error(1)
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

func (m *MockSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// MockInternalUserRepository is a mock implementation of domain.UserRepository
type MockInternalUserRepository struct {
	mock.Mock
}

func (m *MockInternalUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockInternalUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockInternalUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockInternalUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockInternalUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockInternalUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockInternalUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockInternalUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	args := m.Called(ctx, userID, apiKey)
	return args.Error(0)
}

func (m *MockInternalUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockPkgUserRepository is a mock implementation of pkg/domain.UserRepository
type MockPkgUserRepository struct {
	mock.Mock
}

func (m *MockPkgUserRepository) Create(ctx context.Context, user *pkgdomain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockPkgUserRepository) GetByID(ctx context.Context, id string) (*pkgdomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.User), args.Error(1)
}

func (m *MockPkgUserRepository) GetByEmail(ctx context.Context, email string) (*pkgdomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.User), args.Error(1)
}

func (m *MockPkgUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*pkgdomain.User, error) {
	args := m.Called(ctx, apiKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.User), args.Error(1)
}

func (m *MockPkgUserRepository) Update(ctx context.Context, user *pkgdomain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockPkgUserRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPkgUserRepository) List(ctx context.Context, offset, limit int) ([]*pkgdomain.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*pkgdomain.User), args.Error(1)
}

func (m *MockPkgUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	args := m.Called(ctx, userID, apiKey)
	return args.Error(0)
}

// MockInternalSessionStore is a mock implementation of domain.SessionStore
type MockInternalSessionStore struct {
	mock.Mock
}

func (m *MockInternalSessionStore) Create(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockInternalSessionStore) Get(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockInternalSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*domain.Session), args.Error(1)
}

func (m *MockInternalSessionStore) Update(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockInternalSessionStore) Delete(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockInternalSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockInternalSessionStore) DeleteExpired(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockInternalSessionStore) Touch(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

// MockPkgSessionStore is a mock implementation of pkg/domain.SessionStore
type MockPkgSessionStore struct {
	mock.Mock
}

func (m *MockPkgSessionStore) Create(ctx context.Context, session *pkgdomain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockPkgSessionStore) Get(ctx context.Context, id string) (*pkgdomain.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgdomain.Session), args.Error(1)
}

func (m *MockPkgSessionStore) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPkgSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockPkgSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*pkgdomain.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*pkgdomain.Session), args.Error(1)
}

func (m *MockPkgSessionStore) Touch(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockPkgSessionStore) Reset() {
	m.ExpectedCalls = nil
}

// UserRepositoryAdapter adapts domain.UserRepository to pkg/domain.UserRepository
type UserRepositoryAdapter struct {
	internal *MockUserRepository
}

func NewUserRepositoryAdapter(internal *MockUserRepository) pkgdomain.UserRepository {
	return &UserRepositoryAdapter{internal: internal}
}

func (a *UserRepositoryAdapter) Create(ctx context.Context, user *pkgdomain.User) error {
	internalUser := &domain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        domain.Role(user.Role),
		Permissions: converter.ToInternalPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
	return a.internal.Create(ctx, internalUser)
}

func (a *UserRepositoryAdapter) GetByID(ctx context.Context, id string) (*pkgdomain.User, error) {
	user, err := a.internal.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return &pkgdomain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        pkgdomain.Role(user.Role),
		Permissions: converter.ToPkgPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

func (a *UserRepositoryAdapter) GetByEmail(ctx context.Context, email string) (*pkgdomain.User, error) {
	user, err := a.internal.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return &pkgdomain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        pkgdomain.Role(user.Role),
		Permissions: converter.ToPkgPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

func (a *UserRepositoryAdapter) GetByAPIKey(ctx context.Context, apiKey string) (*pkgdomain.User, error) {
	user, err := a.internal.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, nil
	}
	return &pkgdomain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        pkgdomain.Role(user.Role),
		Permissions: converter.ToPkgPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

func (a *UserRepositoryAdapter) Update(ctx context.Context, user *pkgdomain.User) error {
	internalUser := &domain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        domain.Role(user.Role),
		Permissions: converter.ToInternalPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
	return a.internal.Update(ctx, internalUser)
}

func (a *UserRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return a.internal.Delete(ctx, id)
}

func (a *UserRepositoryAdapter) List(ctx context.Context, offset, limit int) ([]*pkgdomain.User, error) {
	users, err := a.internal.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	if users == nil {
		return nil, nil
	}
	pkgUsers := make([]*pkgdomain.User, len(users))
	for i, user := range users {
		pkgUsers[i] = &pkgdomain.User{
			ID:          user.ID,
			Email:       user.Email,
			Password:    user.Password,
			Name:        user.Name,
			Role:        pkgdomain.Role(user.Role),
			Permissions: converter.ToPkgPermissions(user.Permissions),
			Company:     user.Company,
			APIKey:      user.APIKey,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		}
	}
	return pkgUsers, nil
}

func (a *UserRepositoryAdapter) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	return a.internal.UpdateAPIKey(ctx, userID, apiKey)
}

// Reset resets the mock's expectations
func (m *MockUserRepository) Reset() {
	m.ExpectedCalls = []*mock.Call{}
}

// Reset resets the mock's expectations
func (m *MockSessionStore) Reset() {
	m.ExpectedCalls = []*mock.Call{}
}

// Reset resets the mock's expectations
func (m *MockAuthService) Reset() {
	m.ExpectedCalls = []*mock.Call{}
}

// SessionStoreAdapter adapts domain.SessionStore to pkg/domain.SessionStore
type SessionStoreAdapter struct {
	internal *MockSessionStore
}

func NewSessionStoreAdapter(internal *MockSessionStore) domain.SessionStore {
	return &SessionStoreAdapter{internal: internal}
}

func (a *SessionStoreAdapter) Create(ctx context.Context, session *domain.Session) error {
	pkgSession := &pkgdomain.Session{
		ID:          session.ID,
		UserID:      session.UserID,
		Role:        pkgdomain.Role(session.Role),
		Permissions: converter.ToPkgPermissions(session.Permissions),
		UserAgent:   session.UserAgent,
		IP:          session.IP,
		ExpiresAt:   session.ExpiresAt,
		CreatedAt:   session.CreatedAt,
		LastSeenAt:  session.LastSeenAt,
	}
	return a.internal.Create(ctx, pkgSession)
}

func (a *SessionStoreAdapter) Get(ctx context.Context, id string) (*domain.Session, error) {
	pkgSession, err := a.internal.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if pkgSession == nil {
		return nil, nil
	}
	return &domain.Session{
		ID:          pkgSession.ID,
		UserID:      pkgSession.UserID,
		Role:        domain.Role(pkgSession.Role),
		Permissions: converter.ToInternalPermissions(pkgSession.Permissions),
		UserAgent:   pkgSession.UserAgent,
		IP:          pkgSession.IP,
		ExpiresAt:   pkgSession.ExpiresAt,
		CreatedAt:   pkgSession.CreatedAt,
		LastSeenAt:  pkgSession.LastSeenAt,
	}, nil
}

func (a *SessionStoreAdapter) Delete(ctx context.Context, id string) error {
	return a.internal.Delete(ctx, id)
}

func (a *SessionStoreAdapter) Touch(ctx context.Context, id string) error {
	return a.internal.Touch(ctx, id)
}

func (a *SessionStoreAdapter) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	pkgSessions, err := a.internal.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	if pkgSessions == nil {
		return nil, nil
	}
	sessions := make([]*domain.Session, len(pkgSessions))
	for i, pkgSession := range pkgSessions {
		sessions[i] = &domain.Session{
			ID:          pkgSession.ID,
			UserID:      pkgSession.UserID,
			Role:        domain.Role(pkgSession.Role),
			Permissions: converter.ToInternalPermissions(pkgSession.Permissions),
			UserAgent:   pkgSession.UserAgent,
			IP:          pkgSession.IP,
			ExpiresAt:   pkgSession.ExpiresAt,
			CreatedAt:   pkgSession.CreatedAt,
			LastSeenAt:  pkgSession.LastSeenAt,
		}
	}
	return sessions, nil
}

func (a *SessionStoreAdapter) DeleteUserSessions(ctx context.Context, userID string) error {
	return a.internal.DeleteUserSessions(ctx, userID)
}

func (a *SessionStoreAdapter) DeleteExpired(ctx context.Context) error {
	return nil // Not implemented in mock
}

func (a *SessionStoreAdapter) Update(ctx context.Context, session *domain.Session) error {
	return nil // Not implemented in mock
}

// MockAuthUseCase is a mock implementation of AuthUseCaseInterface
type MockAuthUseCase struct {
	mock.Mock
}

func (m *MockAuthUseCase) Register(ctx context.Context, input usecase.RegisterInput) (*domain.User, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthUseCase) Login(ctx context.Context, input usecase.LoginInput) (*usecase.LoginOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.LoginOutput), args.Error(1)
}

func (m *MockAuthUseCase) Logout(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockAuthUseCase) ValidateToken(ctx context.Context, tokenString string) (*domain.User, error) {
	args := m.Called(ctx, tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockAuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (string, string, *domain.User, error) {
	args := m.Called(ctx, refreshToken)
	if args.Error(3) != nil {
		return "", "", nil, args.Error(3)
	}
	return args.String(0), args.String(1), args.Get(2).(*domain.User), nil
}

func (m *MockAuthUseCase) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Session), args.Error(1)
}

func (m *MockAuthUseCase) RevokeSession(ctx context.Context, sessionID string) error {
	args := m.Called(ctx, sessionID)
	return args.Error(0)
}

func (m *MockAuthUseCase) RevokeAllSessions(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockAuthUseCase) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	args := m.Called(ctx, sessionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockAuthUseCase) GenerateAPIKey(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthUseCase) HasPermission(user *domain.User, permission pkgdomain.Permission) bool {
	args := m.Called(user, permission)
	return args.Bool(0)
}

func (m *MockAuthUseCase) CreateSession(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

// MockUserUseCaseImpl embeds usecase.UserUseCase and adds mock functionality
type MockUserUseCaseImpl struct {
	*usecase.UserUseCase
	mock.Mock
}

func NewMockUserUseCase() *MockUserUseCaseImpl {
	return &MockUserUseCaseImpl{
		UserUseCase: &usecase.UserUseCase{},
	}
}

func (m *MockUserUseCaseImpl) GetUser(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserUseCaseImpl) CreateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserUseCaseImpl) UpdateUser(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserUseCaseImpl) DeleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserUseCaseImpl) ListUsers(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	args := m.Called(ctx, offset, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserUseCaseImpl) GenerateAPIKey(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockUserUseCaseImpl) RevokeAPIKey(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func setupAuthHandler() (*gin.Engine, *AuthHandler, *MockUserRepository, *MockSessionStore, usecase.AuthUseCaseInterface) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	userRepo := &MockUserRepository{}
	sessionStore := &MockSessionStore{}
	authUseCase := &MockAuthUseCase{}
	userUseCase := usecase.NewUserUseCase(NewUserRepositoryAdapter(userRepo))

	handler := NewAuthHandler(authUseCase, userUseCase, NewSessionStoreAdapter(sessionStore))

	// Add middleware to handle session
	router.Use(func(c *gin.Context) {
		// Get session ID from context
		if sessionID, exists := c.Get("session_id"); exists {
			c.Set("session_id", sessionID)

			// Get session from context
			if session, exists := c.Get("session"); exists {
				c.Set("session", session)

				// Get user from context
				if user, exists := c.Get("user"); exists {
					c.Set("user", user)
					if u, ok := user.(*domain.User); ok {
						c.Set("user_id", u.ID)
					}
				}
			}
		}
		c.Next()
	})

	router.POST("/refresh", handler.RefreshToken)
	router.POST("/logout", handler.Logout)
	router.POST("/login", handler.Login)
	router.POST("/register", handler.Register)
	router.GET("/me", handler.GetCurrentUser)
	router.POST("/api-key", handler.GenerateAPIKey)
	router.GET("/sessions", handler.GetActiveSessions)
	router.POST("/sessions/revoke/:id", handler.RevokeSession)
	router.POST("/sessions/revoke-all", handler.RevokeAllSessions)

	return router, handler, userRepo, sessionStore, authUseCase
}

func TestAuthHandler_Register(t *testing.T) {
	router, _, _, _, authUseCase := setupAuthHandler()
	mockAuthUseCase := authUseCase.(*MockAuthUseCase)

	// Reset mock expectations
	mockAuthUseCase.ExpectedCalls = nil

	// Set up auth use case expectations
	mockAuthUseCase.On("Register", mock.Anything, usecase.RegisterInput{
		Email:    "test@example.com",
		Password: "password123",
	}).Return(&domain.User{
		ID:    "user-id",
		Email: "test@example.com",
		Role:  domain.RoleUser,
	}, nil)

	// Create test request
	reqBody := map[RequestKey]interface{}{
		RequestKey("email"):    "test@example.com",
		RequestKey("password"): "password123",
	}
	reqBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	require.Equal(t, http.StatusCreated, w.Code)
	mockAuthUseCase.AssertExpectations(t)
}

func TestAuthHandler_Login(t *testing.T) {
	router, _, _, sessionStore, authUseCase := setupAuthHandler()
	mockAuthUseCase := authUseCase.(*MockAuthUseCase)

	// Reset mock expectations
	mockAuthUseCase.ExpectedCalls = nil
	sessionStore.Reset()

	// Set up auth use case expectations
	mockAuthUseCase.On("Login", mock.Anything, usecase.LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	}).Return(&usecase.LoginOutput{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		User: &domain.User{
			ID:    "user-id",
			Email: "test@example.com",
			Role:  domain.RoleUser,
		},
		Session: &domain.Session{
			ID:     "session-id",
			UserID: "user-id",
		},
	}, nil)

	// Set up session store expectations
	sessionStore.On("Create", mock.Anything, mock.MatchedBy(func(s *pkgdomain.Session) bool {
		return s.UserID == "user-id"
	})).Return(nil)

	// Create test request
	reqBody := map[RequestKey]interface{}{
		RequestKey("email"):    "test@example.com",
		RequestKey("password"): "password123",
	}
	reqBytes, err := json.Marshal(reqBody)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	router.ServeHTTP(w, req)

	// Assert response
	require.Equal(t, http.StatusOK, w.Code)
	mockAuthUseCase.AssertExpectations(t)
	sessionStore.AssertExpectations(t)
}
