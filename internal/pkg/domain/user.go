package domain

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// SubscriptionPlan represents a user's subscription level
type SubscriptionPlan string

const (
	PlanFree     SubscriptionPlan = "free"
	PlanBasic    SubscriptionPlan = "basic"
	PlanPro      SubscriptionPlan = "pro"
	PlanBusiness SubscriptionPlan = "business"
)

// PlanLimits defines the quota limits for each subscription plan
var PlanLimits = map[SubscriptionPlan]int{
	PlanFree:     100,
	PlanBasic:    1000,
	PlanPro:      10000,
	PlanBusiness: 100000,
}

// User represents a system user
type User struct {
	ID             string           `json:"id"`
	Email          string           `json:"email"`
	Password       string           `json:"-"` // Never expose password in JSON
	Name           string           `json:"name"`
	Role           Role             `json:"role"`
	Permissions    []Permission     `json:"permissions"`
	Company        string           `json:"company,omitempty"`
	APIKey         string           `json:"api_key,omitempty"`
	Plan           SubscriptionPlan `json:"plan"`
	TrackQuota     int              `json:"track_quota"`
	TracksUsed     int              `json:"tracks_used"`
	QuotaResetDate time.Time        `json:"quota_reset_date"`
	LastLoginAt    time.Time        `json:"last_login_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// NewUser creates a new user with default values
func NewUser(email, name string, role Role) *User {
	now := time.Now()
	return &User{
		Email:          email,
		Name:           name,
		Role:           role,
		Permissions:    RolePermissions[role],
		Plan:           PlanFree,
		TrackQuota:     PlanLimits[PlanFree],
		QuotaResetDate: now.AddDate(0, 1, 0), // Reset quota in 1 month
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// SetPassword safely hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword verifies if the provided password matches the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(perm Permission) bool {
	for _, p := range u.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// UpdatePlan updates the user's subscription plan and associated limits
func (u *User) UpdatePlan(plan SubscriptionPlan) {
	u.Plan = plan
	u.TrackQuota = PlanLimits[plan]
	u.UpdatedAt = time.Now()
}

// HasQuotaAvailable checks if the user has remaining quota
func (u *User) HasQuotaAvailable() bool {
	if time.Now().After(u.QuotaResetDate) {
		u.TracksUsed = 0
		u.QuotaResetDate = time.Now().AddDate(0, 1, 0)
	}
	return u.TracksUsed < u.TrackQuota
}

// IncrementTracksUsed increments the tracks used count
func (u *User) IncrementTracksUsed() {
	u.TracksUsed++
	u.UpdatedAt = time.Now()
}

// Validate performs validation on user data
func (u *User) Validate() error {
	if u.Email == "" {
		return fmt.Errorf("email is required")
	}
	if u.Name == "" {
		return fmt.Errorf("name is required")
	}
	if u.Role == "" {
		return fmt.Errorf("role is required")
	}
	if u.Plan == "" {
		return fmt.Errorf("subscription plan is required")
	}
	return nil
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
	UpdateAPIKey(ctx context.Context, userID string, apiKey string) error
	List(ctx context.Context, offset, limit int) ([]*User, error)
}
