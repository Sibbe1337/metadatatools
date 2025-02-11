package base

import (
	"context"
	"fmt"
	"metadatatool/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemorySessionRepository implements domain.SessionStore for testing
type InMemorySessionRepository struct {
	sessions     map[string]*domain.Session // SessionID -> Session
	userSessions map[string][]string        // UserID -> []SessionID
	mu           sync.RWMutex
}

// NewInMemorySessionRepository creates a new in-memory session repository
func NewInMemorySessionRepository() domain.SessionStore {
	return &InMemorySessionRepository{
		sessions:     make(map[string]*domain.Session),
		userSessions: make(map[string][]string),
	}
}

// Create stores a new session
func (r *InMemorySessionRepository) Create(ctx context.Context, session *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session.ID == "" {
		session.ID = uuid.New().String()
	}

	// Set timestamps if not set
	now := time.Now()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = now
	}
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = now.Add(24 * time.Hour)
	}

	r.sessions[session.ID] = session

	// Add to user sessions
	r.userSessions[session.UserID] = append(r.userSessions[session.UserID], session.ID)

	return nil
}

// Get retrieves a session by ID
func (r *InMemorySessionRepository) Get(ctx context.Context, id string) (*domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// Delete removes a session
func (r *InMemorySessionRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[id]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Remove from userSessions
	sessions := r.userSessions[session.UserID]
	for i, sid := range sessions {
		if sid == id {
			r.userSessions[session.UserID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}

	delete(r.sessions, id)
	return nil
}

// Touch updates the last seen time of a session
func (r *InMemorySessionRepository) Touch(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[id]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.LastSeenAt = time.Now()
	r.sessions[id] = session
	return nil
}

// GetUserSessions retrieves all active sessions for a user
func (r *InMemorySessionRepository) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionIDs, exists := r.userSessions[userID]
	if !exists {
		return []*domain.Session{}, nil
	}

	var activeSessions []*domain.Session
	now := time.Now()
	for _, id := range sessionIDs {
		session := r.sessions[id]
		if session.ExpiresAt.After(now) {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions, nil
}

// DeleteUserSessions removes all sessions for a user
func (r *InMemorySessionRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sessionIDs, exists := r.userSessions[userID]
	if !exists {
		return nil
	}

	for _, id := range sessionIDs {
		delete(r.sessions, id)
	}

	delete(r.userSessions, userID)
	return nil
}

// DeleteExpired removes all expired sessions
func (r *InMemorySessionRepository) DeleteExpired(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for id, session := range r.sessions {
		if session.ExpiresAt.Before(now) {
			// Remove from userSessions
			sessions := r.userSessions[session.UserID]
			for i, sid := range sessions {
				if sid == id {
					r.userSessions[session.UserID] = append(sessions[:i], sessions[i+1:]...)
					break
				}
			}
			delete(r.sessions, id)
		}
	}
	return nil
}

// Update updates an existing session
func (r *InMemorySessionRepository) Update(ctx context.Context, session *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[session.ID]; !exists {
		return fmt.Errorf("session not found")
	}

	r.sessions[session.ID] = session
	return nil
}
