package domain

import (
	"time"
)

// Role represents user access levels
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleLabelUser Role = "label_user"
	RoleAPIUser   Role = "api_user"
)

// User represents a system user with role-based permissions
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // Never expose password in JSON
	Name     string `json:"name"`
	Role     Role   `json:"role"`
	Company  string `json:"company"`
	APIKey   string `json:"api_key,omitempty"`

	// Subscription and usage tracking
	Plan           string    `json:"plan"` // free, pro, enterprise
	TrackQuota     int       `json:"track_quota"`
	TracksUsed     int       `json:"tracks_used"`
	QuotaResetDate time.Time `json:"quota_reset_date"`

	// Compliance and auditing
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	LastLoginAt time.Time  `json:"last_login_at"`
}

// UserRepository defines the interface for user data persistence
type UserRepository interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByAPIKey(apiKey string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	List(offset, limit int) ([]*User, error)
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	GenerateTokens(user *User) (*TokenPair, error)
	ValidateToken(token string) (*Claims, error)
	RefreshToken(refreshToken string) (*TokenPair, error)
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) error
	GenerateAPIKey() (string, error)
}

// TokenPair represents an authentication token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Claims represents the JWT claims structure
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   Role   `json:"role"`
}
