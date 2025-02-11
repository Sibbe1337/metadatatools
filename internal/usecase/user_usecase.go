package usecase

import (
	"context"
	"fmt"
	"metadatatool/internal/pkg/domain"
	"time"

	"github.com/google/uuid"
)

// UserUseCase handles user operations
type UserUseCase struct {
	userRepo domain.UserRepository
}

// NewUserUseCase creates a new user use case
func NewUserUseCase(userRepo domain.UserRepository) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

// GenerateAPIKey generates a new API key for a user
func (uc *UserUseCase) GenerateAPIKey(ctx context.Context, userID string) (string, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("error getting user: %w", err)
	}
	if user == nil {
		return "", fmt.Errorf("user not found")
	}

	// Generate new API key
	apiKey := uuid.New().String()
	user.APIKey = apiKey
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return "", fmt.Errorf("error updating user: %w", err)
	}

	return apiKey, nil
}

// RevokeAPIKey revokes a user's API key
func (uc *UserUseCase) RevokeAPIKey(ctx context.Context, userID string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.APIKey = ""
	user.UpdatedAt = time.Now()

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	return nil
}

// GetUser retrieves a user by ID
func (uc *UserUseCase) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// ListUsers retrieves a list of users with pagination
func (uc *UserUseCase) ListUsers(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	users, err := uc.userRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("error listing users: %w", err)
	}
	return users, nil
}

// UpdateUser updates a user's information
func (uc *UserUseCase) UpdateUser(ctx context.Context, user *domain.User) error {
	existingUser, err := uc.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("error getting user: %w", err)
	}
	if existingUser == nil {
		return fmt.Errorf("user not found")
	}

	user.UpdatedAt = time.Now()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}

	return nil
}

// DeleteUser deletes a user
func (uc *UserUseCase) DeleteUser(ctx context.Context, userID string) error {
	if err := uc.userRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}
	return nil
}
