package domain

import "errors"

var (
	// ErrInvalidCredentials is returned when login credentials are incorrect
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidToken is returned when a token is invalid or expired
	ErrInvalidToken = errors.New("invalid token")

	// ErrInternal is returned when an internal error occurs
	ErrInternal = errors.New("internal error")

	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrEmailExists is returned when trying to register with an existing email
	ErrEmailExists = errors.New("email already exists")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when a user is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when a user does not have permission
	ErrForbidden = errors.New("forbidden")
)
