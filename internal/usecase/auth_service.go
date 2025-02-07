package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/config"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	cfg      *config.AuthConfig
	userRepo domain.UserRepository
}

// NewAuthService creates a new authentication service
func NewAuthService(cfg *config.AuthConfig, userRepo domain.UserRepository) domain.AuthService {
	return &authService{
		cfg:      cfg,
		userRepo: userRepo,
	}
}

// GenerateTokens creates a new pair of access and refresh tokens
func (s *authService) GenerateTokens(user *domain.User) (*domain.TokenPair, error) {
	// Get permissions for the user's role
	permissions := s.GetPermissions(user.Role)

	// Create access token with permissions
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     user.ID,
		"email":       user.Email,
		"role":        user.Role,
		"permissions": permissions,
		"exp":         time.Now().Add(s.cfg.AccessTokenExpiry).Unix(),
	})

	accessTokenString, err := accessToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Create refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(s.cfg.RefreshTokenExpiry).Unix(),
	})

	refreshTokenString, err := refreshToken.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
	}, nil
}

// ValidateToken validates and parses a JWT token
func (s *authService) ValidateToken(tokenString string) (*domain.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Extract permissions from the token
		var permissions []domain.Permission
		if perms, ok := claims["permissions"].([]interface{}); ok {
			for _, p := range perms {
				if pStr, ok := p.(string); ok {
					permissions = append(permissions, domain.Permission(pStr))
				}
			}
		}

		return &domain.Claims{
			UserID:      claims["user_id"].(string),
			Email:       claims["email"].(string),
			Role:        domain.Role(claims["role"].(string)),
			Permissions: permissions,
		}, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// RefreshToken validates a refresh token and generates new token pair
func (s *authService) RefreshToken(refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	ctx := context.Background()
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.GenerateTokens(user)
}

// HashPassword creates a bcrypt hash of the password
func (s *authService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword checks if the provided password matches the hash
func (s *authService) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateAPIKey creates a new API key
func (s *authService) GenerateAPIKey() (string, error) {
	return uuid.NewString(), nil
}

// HasPermission checks if a role has a specific permission
func (s *authService) HasPermission(role domain.Role, permission domain.Permission) bool {
	permissions := domain.RolePermissions[role]
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissions returns all permissions for a role
func (s *authService) GetPermissions(role domain.Role) []domain.Permission {
	return domain.RolePermissions[role]
}
