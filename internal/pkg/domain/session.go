package domain

import (
	"context"
	"time"
)

// SessionConfig holds configuration for session management
type SessionConfig struct {
	// Cookie settings
	CookieName     string `json:"cookie_name"`
	CookieDomain   string `json:"cookie_domain"`
	CookiePath     string `json:"cookie_path"`
	CookieSecure   bool   `json:"cookie_secure"`
	CookieHTTPOnly bool   `json:"cookie_http_only"`
	CookieSameSite string `json:"cookie_same_site"`

	// Session settings
	SessionDuration    time.Duration `json:"session_duration"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user"`
}

// Session represents a user session
type Session struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
	UserAgent   string       `json:"user_agent"`
	IP          string       `json:"ip"`
	ExpiresAt   time.Time    `json:"expires_at"`
	CreatedAt   time.Time    `json:"created_at"`
	LastSeenAt  time.Time    `json:"last_seen_at"`
}

// HasPermission checks if the session has the given permission
func (s *Session) HasPermission(p Permission) bool {
	for _, perm := range s.Permissions {
		if perm == p {
			return true
		}
	}
	return false
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// SessionRepository defines the interface for session persistence
type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id string) (*Session, error)
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)
	Delete(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID string) error
}

// SessionStore defines the interface for session storage
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
	Touch(ctx context.Context, id string) error
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)
	DeleteUserSessions(ctx context.Context, userID string) error
}
