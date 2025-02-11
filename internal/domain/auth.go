package domain

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthService defines the interface for authentication operations
type AuthService interface {
	GenerateToken(ctx context.Context, user *User) (string, error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) error
	GenerateTokens(user *User) (*Tokens, error)
	GenerateAPIKey() (string, error)
}

// Tokens represents the access and refresh tokens
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// TokenClaims combines session Claims with JWT RegisteredClaims
type TokenClaims struct {
	Claims
	jwt.RegisteredClaims
}

// GetExpirationTime implements jwt.Claims interface
func (c *TokenClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return c.ExpiresAt, nil
}

// GetIssuedAt implements jwt.Claims interface
func (c *TokenClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return c.IssuedAt, nil
}

// GetNotBefore implements jwt.Claims interface
func (c *TokenClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return c.NotBefore, nil
}

// GetIssuer implements jwt.Claims interface
func (c *TokenClaims) GetIssuer() (string, error) {
	return c.Issuer, nil
}

// GetSubject implements jwt.Claims interface
func (c *TokenClaims) GetSubject() (string, error) {
	return c.Subject, nil
}

// GetAudience implements jwt.Claims interface
func (c *TokenClaims) GetAudience() (jwt.ClaimStrings, error) {
	return c.Audience, nil
}

// NewClaims creates a new Claims instance with standard JWT claims
func NewClaims(userID string, role Role, permissions []Permission, expiresAt time.Time) *TokenClaims {
	return &TokenClaims{
		Claims: Claims{
			UserID:      userID,
			Role:        role,
			Permissions: permissions,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
}

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrEmailTaken is returned when trying to register with an existing email
	ErrEmailTaken = errors.New("email already taken")

	// ErrInvalidToken is returned when a token is invalid or expired
	ErrInvalidToken = errors.New("invalid or expired token")

	// ErrInvalidAPIKey is returned when an API key is invalid
	ErrInvalidAPIKey = errors.New("invalid API key")
)
