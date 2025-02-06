package postgres

import (
	"context"
	"errors"
	"metadatatool/internal/pkg/domain"
	"time"

	"gorm.io/gorm"
)

// UserModel is the GORM model for users
type UserModel struct {
	gorm.Model
	ID             string `gorm:"primaryKey"`
	Email          string `gorm:"uniqueIndex"`
	Password       string
	Name           string
	Role           string
	Company        string
	APIKey         string `gorm:"uniqueIndex"`
	Plan           string
	TrackQuota     int
	TracksUsed     int
	QuotaResetDate int64
	LastLoginAt    int64
}

// PostgresUserRepository implements domain.UserRepository
type PostgresUserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new PostgresUserRepository
func NewUserRepository(db *gorm.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create inserts a new user into the database
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	model := &UserModel{
		ID:             user.ID,
		Email:          user.Email,
		Password:       user.Password,
		Name:           user.Name,
		Role:           string(user.Role),
		Company:        user.Company,
		APIKey:         user.APIKey,
		Plan:           string(user.Plan),
		TrackQuota:     user.TrackQuota,
		TracksUsed:     user.TracksUsed,
		QuotaResetDate: user.QuotaResetDate.Unix(),
		LastLoginAt:    user.LastLoginAt.Unix(),
	}

	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// GetByID retrieves a user by their ID
func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var model UserModel
	result := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return modelToDomain(&model), nil
}

// GetByEmail retrieves a user by their email
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model UserModel
	result := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return modelToDomain(&model), nil
}

// GetByAPIKey retrieves a user by their API key
func (r *PostgresUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	var model UserModel
	result := r.db.WithContext(ctx).Where("api_key = ? AND deleted_at IS NULL", apiKey).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return modelToDomain(&model), nil
}

// Update modifies an existing user
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	model := &UserModel{
		ID:             user.ID,
		Email:          user.Email,
		Password:       user.Password,
		Name:           user.Name,
		Role:           string(user.Role),
		Company:        user.Company,
		APIKey:         user.APIKey,
		Plan:           string(user.Plan),
		TrackQuota:     user.TrackQuota,
		TracksUsed:     user.TracksUsed,
		QuotaResetDate: user.QuotaResetDate.Unix(),
		LastLoginAt:    user.LastLoginAt.Unix(),
	}

	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Delete soft-deletes a user
func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserModel{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// List retrieves a paginated list of users
func (r *PostgresUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	var models []*UserModel
	result := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&models)

	if result.Error != nil {
		return nil, result.Error
	}

	users := make([]*domain.User, len(models))
	for i, model := range models {
		users[i] = modelToDomain(model)
	}
	return users, nil
}

// Helper function to convert UserModel to domain.User
func modelToDomain(model *UserModel) *domain.User {
	return &domain.User{
		ID:             model.ID,
		Email:          model.Email,
		Password:       model.Password,
		Name:           model.Name,
		Role:           domain.Role(model.Role),
		Company:        model.Company,
		APIKey:         model.APIKey,
		Plan:           domain.SubscriptionPlan(model.Plan),
		TrackQuota:     model.TrackQuota,
		TracksUsed:     model.TracksUsed,
		QuotaResetDate: time.Unix(model.QuotaResetDate, 0),
		LastLoginAt:    time.Unix(model.LastLoginAt, 0),
	}
}
