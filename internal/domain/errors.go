package domain

import "errors"

var (
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")

	// ErrMaxSessionsReached is returned when a user has reached their maximum number of sessions
	ErrMaxSessionsReached = errors.New("maximum number of sessions reached")

	// ErrSessionExpired is returned when a session has expired
	ErrSessionExpired = errors.New("session expired")
)
