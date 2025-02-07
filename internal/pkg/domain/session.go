package domain

import (
	"context"
	"time"
)

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	IP           string    `json:"ip"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastSeenAt   time.Time `json:"last_seen_at"`
}

// SessionStore defines the interface for session management
type SessionStore interface {
	// Create creates a new session
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, sessionID string) (*Session, error)

	// GetUserSessions retrieves all active sessions for a user
	GetUserSessions(ctx context.Context, userID string) ([]*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// Delete removes a session
	Delete(ctx context.Context, sessionID string) error

	// DeleteUserSessions removes all sessions for a user
	DeleteUserSessions(ctx context.Context, userID string) error

	// DeleteExpired removes all expired sessions
	DeleteExpired(ctx context.Context) error

	// Touch updates the last seen time of a session
	Touch(ctx context.Context, sessionID string) error
}

// SessionConfig holds configuration for session management
type SessionConfig struct {
	// SessionDuration is how long a session should last
	SessionDuration time.Duration

	// CleanupInterval is how often to run the cleanup job
	CleanupInterval time.Duration

	// MaxSessionsPerUser is the maximum number of concurrent sessions per user
	MaxSessionsPerUser int
}
