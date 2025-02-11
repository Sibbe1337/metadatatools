package domain

import (
	"context"
	"time"
)

// User represents a system user
type User struct {
	ID          string       `json:"id"`
	Email       string       `json:"email"`
	Password    string       `json:"-"` // Never expose password in JSON
	Name        string       `json:"name"`
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
	Company     string       `json:"company,omitempty"`
	APIKey      string       `json:"api_key,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// UserRepository defines the interface for user persistence
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*User, error)
	UpdateAPIKey(ctx context.Context, userID string, apiKey string) error
	Count(ctx context.Context) (int64, error)
}
