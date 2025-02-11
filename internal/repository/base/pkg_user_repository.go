package base

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PkgUserRepository implements pkg/domain.UserRepository using GORM
type PkgUserRepository struct {
	db *gorm.DB
}

// NewPkgUserRepository creates a new pkg/domain user repository
func NewPkgUserRepository(db *gorm.DB) domain.UserRepository {
	return &PkgUserRepository{db: db}
}

// Create creates a new user
func (r *PkgUserRepository) Create(ctx context.Context, user *domain.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return fmt.Errorf("failed to create user: %w", result.Error)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *PkgUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).First(&user, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *PkgUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).First(&user, "email = ?", email)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}

	return &user, nil
}

// GetByAPIKey retrieves a user by API key
func (r *PkgUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).First(&user, "api_key = ?", apiKey)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}

	return &user, nil
}

// Update updates an existing user
func (r *PkgUserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now()
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return fmt.Errorf("failed to update user: %w", result.Error)
	}

	return nil
}

// Delete deletes a user
func (r *PkgUserRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}

	return nil
}

// List retrieves users with pagination
func (r *PkgUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	var users []*domain.User
	result := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list users: %w", result.Error)
	}

	return users, nil
}

// UpdateAPIKey updates a user's API key
func (r *PkgUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	result := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("api_key", apiKey)
	if result.Error != nil {
		return fmt.Errorf("failed to update API key: %w", result.Error)
	}

	return nil
}

// Count returns the total number of users
func (r *PkgUserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).Model(&domain.User{}).Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("failed to count users: %w", result.Error)
	}

	return count, nil
}
