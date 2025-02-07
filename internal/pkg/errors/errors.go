package errors

import (
	"fmt"
	"net/http"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Error types
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeDatabase     ErrorType = "DATABASE_ERROR"
	ErrorTypeStorage      ErrorType = "STORAGE_ERROR"
	ErrorTypeAI           ErrorType = "AI_ERROR"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
)

// AppError represents an application error
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"-"`
	Err        error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Err.Error())
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details string) *AppError {
	return &AppError{
		Type:       ErrorTypeValidation,
		Message:    message,
		Details:    details,
		StatusCode: http.StatusBadRequest,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeNotFound,
		Message:    message,
		StatusCode: http.StatusNotFound,
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeDatabase,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewStorageError creates a new storage error
func NewStorageError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeStorage,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewAIError creates a new AI service error
func NewAIError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeAI,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewUnauthorizedError creates a new unauthorized error
func NewUnauthorizedError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeUnauthorized,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// NewForbiddenError creates a new forbidden error
func NewForbiddenError(message string) *AppError {
	return &AppError{
		Type:       ErrorTypeForbidden,
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Type:       ErrorTypeInternal,
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Err:        err,
	}
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*AppError); ok {
		return e.Type == ErrorTypeNotFound
	}
	return false
}

// IsValidation checks if the error is a validation error
func IsValidation(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*AppError); ok {
		return e.Type == ErrorTypeValidation
	}
	return false
}

// IsUnauthorized checks if the error is an unauthorized error
func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*AppError); ok {
		return e.Type == ErrorTypeUnauthorized
	}
	return false
}

// IsForbidden checks if the error is a forbidden error
func IsForbidden(err error) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*AppError); ok {
		return e.Type == ErrorTypeForbidden
	}
	return false
}
