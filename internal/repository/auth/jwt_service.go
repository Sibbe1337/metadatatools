package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"metadatatool/internal/pkg/domain"
)

const (
	// Token durations
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 7 * 24 * time.Hour

	// Key lengths
	apiKeyLength    = 32
	SecretKeyLength = 32 // Exported for tests
)

// customClaims extends jwt.RegisteredClaims with our custom claims
type customClaims struct {
	jwt.RegisteredClaims
	UserID      string              `json:"uid"`
	Email       string              `json:"email"`
	Role        domain.Role         `json:"role"`
	Permissions []domain.Permission `json:"permissions"`
}

// JWTService implements the domain.AuthService interface
type JWTService struct {
	secretKey []byte
}

// NewJWTService creates a new JWT authentication service
func NewJWTService(secretKey []byte) (*JWTService, error) {
	if len(secretKey) == 0 {
		// Generate a random secret key if none provided
		key := make([]byte, SecretKeyLength) // Use exported constant
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("failed to generate secret key: %w", err)
		}
		secretKey = key
	}

	return &JWTService{
		secretKey: secretKey,
	}, nil
}

// GenerateTokens creates a new pair of access and refresh tokens
func (s *JWTService) GenerateTokens(user *domain.User) (*domain.TokenPair, error) {
	// Generate access token
	accessToken, err := s.createToken(user, accessTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.createToken(user, refreshTokenDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateToken validates and parses a JWT token
func (s *JWTService) ValidateToken(tokenString string) (*domain.Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("%w: token is empty", domain.ErrInvalidToken)
	}

	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: invalid signing method", domain.ErrInvalidToken)
		}
		return s.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, fmt.Errorf("%w: token is expired", domain.ErrInvalidToken)
		}
		return nil, fmt.Errorf("%w: token is invalid", domain.ErrInvalidToken)
	}

	if !token.Valid {
		return nil, fmt.Errorf("%w: token is invalid", domain.ErrInvalidToken)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return nil, fmt.Errorf("%w: invalid claims format", domain.ErrInvalidToken)
	}

	return &domain.Claims{
		UserID:      claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	}, nil
}

// RefreshToken validates a refresh token and generates new token pair
func (s *JWTService) RefreshToken(refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Create a temporary user object to generate new tokens
	user := &domain.User{
		ID:          claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	}

	return s.GenerateTokens(user)
}

// HashPassword creates a bcrypt hash of the password
func (s *JWTService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the hash
func (s *JWTService) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateAPIKey creates a new API key
func (s *JWTService) GenerateAPIKey() (string, error) {
	bytes := make([]byte, apiKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HasPermission checks if a role has a specific permission
func (s *JWTService) HasPermission(role domain.Role, permission domain.Permission) bool {
	permissions := s.GetPermissions(role)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func (s *JWTService) GetPermissions(role domain.Role) []domain.Permission {
	return domain.RolePermissions[role]
}

// Helper method to create a token
func (s *JWTService) createToken(user *domain.User, duration time.Duration) (string, error) {
	now := time.Now()
	claims := customClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: user.Permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}
