package domain

import (
	"context"
	"time"
)

// Claims represents JWT claims
type Claims struct {
	UserID      string       `json:"user_id"`
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
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

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return s.ExpiresAt.Before(time.Now())
}

// SessionStore defines the interface for session storage
type SessionStore interface {
	Get(ctx context.Context, id string) (*Session, error)
	Create(ctx context.Context, session *Session) error
	Delete(ctx context.Context, id string) error
	Touch(ctx context.Context, id string) error
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)
	DeleteUserSessions(ctx context.Context, userID string) error
}

// SessionConfig holds configuration for session management
type SessionConfig struct {
	CookieName         string
	CookiePath         string
	CookieDomain       string
	CookieSecure       bool
	CookieHTTPOnly     bool
	CookieSameSite     string
	MaxSessionsPerUser int
	SessionDuration    time.Duration
	CleanupInterval    time.Duration
}
