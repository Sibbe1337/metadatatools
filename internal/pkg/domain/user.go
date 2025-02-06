package domain

import (
	"context"
	"time"
)

// Role represents a user's role in the system
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleGuest  Role = "guest"
	RoleSystem Role = "system"
)

// SubscriptionPlan represents a user's subscription level
type SubscriptionPlan string

const (
	PlanFree     SubscriptionPlan = "free"
	PlanBasic    SubscriptionPlan = "basic"
	PlanPro      SubscriptionPlan = "pro"
	PlanBusiness SubscriptionPlan = "business"
)

// User represents a user in the system
type User struct {
	ID             string
	Email          string
	Password       string
	Name           string
	Role           Role
	Company        string
	APIKey         string
	Plan           SubscriptionPlan
	TrackQuota     int
	TracksUsed     int
	QuotaResetDate time.Time
	LastLoginAt    time.Time
}

// UserRepository defines the interface for user data persistence
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*User, error)
}
