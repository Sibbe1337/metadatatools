package base

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"metadatatool/internal/domain"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// InMemoryAuthService implements domain.AuthService for testing
type InMemoryAuthService struct {
	tokens    map[string]string // TokenID -> UserID
	secretKey []byte
	mu        sync.RWMutex
}

// NewInMemoryAuthService creates a new in-memory auth service
func NewInMemoryAuthService() domain.AuthService {
	return &InMemoryAuthService{
		tokens:    make(map[string]string),
		secretKey: []byte("test-secret-key"),
	}
}

// GenerateToken generates a new token for a user
func (s *InMemoryAuthService) GenerateToken(ctx context.Context, user *domain.User) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenID := uuid.New().String()
	s.tokens[tokenID] = user.ID
	return tokenID, nil
}

// ValidateToken validates a token and returns the claims
func (s *InMemoryAuthService) ValidateToken(ctx context.Context, token string) (*domain.TokenClaims, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	claims := &domain.TokenClaims{
		Claims: domain.Claims{
			UserID: userID,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return claims, nil
}

// GenerateTokens creates a new pair of access and refresh tokens
func (s *InMemoryAuthService) GenerateTokens(user *domain.User) (*domain.Tokens, error) {
	accessToken, err := s.GenerateToken(context.Background(), user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.GenerateToken(context.Background(), user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &domain.Tokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshToken validates a refresh token and generates new token pair
func (s *InMemoryAuthService) RefreshToken(refreshToken string) (*domain.Tokens, error) {
	claims, err := s.ValidateToken(context.Background(), refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Create a temporary user object to generate new tokens
	user := &domain.User{
		ID: claims.UserID,
	}

	return s.GenerateTokens(user)
}

// HashPassword creates a bcrypt hash of the password
func (s *InMemoryAuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the hash
func (s *InMemoryAuthService) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateAPIKey creates a new API key
func (s *InMemoryAuthService) GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// rolePermissions maps roles to their allowed permissions
var rolePermissions = map[domain.Role][]domain.Permission{
	domain.RoleAdmin: {
		domain.PermissionReadTrack,
		domain.PermissionWriteTrack,
		domain.PermissionDeleteTrack,
		domain.PermissionReadLabel,
		domain.PermissionWriteLabel,
		domain.PermissionDeleteLabel,
		domain.PermissionManageAPIKeys,
	},
	domain.RoleUser: {
		domain.PermissionReadTrack,
		domain.PermissionWriteTrack,
		domain.PermissionReadLabel,
	},
}

// HasPermission checks if a role has a specific permission
func (s *InMemoryAuthService) HasPermission(role domain.Role, permission domain.Permission) bool {
	permissions := rolePermissions[role]
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func (s *InMemoryAuthService) GetPermissions(role domain.Role) []domain.Permission {
	return rolePermissions[role]
}
