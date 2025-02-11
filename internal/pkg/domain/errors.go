package domain

import "errors"

var (
	// Authentication errors
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")

	// User errors
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already exists")

	// Session errors
	ErrSessionNotFound = errors.New("session not found")

	// Validation errors
	ErrInvalidInput = errors.New("invalid input")

	// System errors
	ErrInternal = errors.New("internal error")
)
