package integration

import (
	"context"
	"metadatatool/internal/domain"
	"metadatatool/internal/handler"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

// TestServer represents a test server instance
type TestServer struct {
	Router       *gin.Engine
	AuthHandler  *handler.AuthHandler
	SessionStore domain.SessionStore
	AuthService  domain.AuthService
	UserRepo     domain.UserRepository
}

// MockAuthService is a mock implementation of domain.AuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) HashPassword(ctx context.Context, password string) (string, error) {
	args := m.Called(ctx, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ComparePasswords(ctx context.Context, hashedPassword, password string) error {
	args := m.Called(ctx, hashedPassword, password)
	return args.Error(0)
}

func (m *MockAuthService) GenerateToken(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) GenerateAPIKey(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}
