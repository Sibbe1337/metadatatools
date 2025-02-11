package usecase

import (
	"context"
	"errors"
	"fmt"
	"metadatatool/internal/domain"
	"metadatatool/internal/pkg/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService implements domain.AuthService interface
type AuthService struct {
	config *config.AuthConfig
}

// NewAuthService creates a new auth service
func NewAuthService(config *config.AuthConfig) domain.AuthService {
	return &AuthService{config: config}
}

// GenerateToken generates a new JWT token
func (s *AuthService) GenerateToken(ctx context.Context, user *domain.User) (string, error) {
	claims := domain.NewClaims(
		user.ID,
		user.Role,
		user.Permissions,
		time.Now().Add(s.config.AccessTokenTTL),
	)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// GenerateTokens generates access and refresh tokens
func (s *AuthService) GenerateTokens(user *domain.User) (*domain.Tokens, error) {
	// Generate access token
	accessClaims := domain.NewClaims(
		user.ID,
		user.Role,
		user.Permissions,
		time.Now().Add(s.config.AccessTokenTTL),
	)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := domain.NewClaims(
		user.ID,
		user.Role,
		user.Permissions,
		time.Now().Add(s.config.RefreshTokenTTL),
	)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &domain.Tokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*domain.TokenClaims, error) {
	claims := &domain.TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domain.ErrInvalidToken
		}
		return nil, domain.ErrInvalidToken
	}

	if !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	if claims, ok := token.Claims.(*domain.TokenClaims); ok {
		return claims, nil
	}

	return nil, domain.ErrInvalidToken
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword verifies a password against its hash
func (s *AuthService) VerifyPassword(hashedPassword, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}

// GenerateAPIKey generates a new API key
func (s *AuthService) GenerateAPIKey() (string, error) {
	return uuid.NewString(), nil
}
