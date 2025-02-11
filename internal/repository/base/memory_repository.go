package base

import (
	"context"
	"fmt"
	"metadatatool/internal/domain"
	"sync"

	"github.com/google/uuid"
)

// InMemoryUserRepository implements domain.UserRepository for testing
type InMemoryUserRepository struct {
	users     map[string]*domain.User // ID -> User
	emailMap  map[string]string       // Email -> ID
	apiKeyMap map[string]string       // APIKey -> ID
	mu        sync.RWMutex
}

// NewInMemoryUserRepository creates a new in-memory user repository
func NewInMemoryUserRepository() domain.UserRepository {
	return &InMemoryUserRepository{
		users:     make(map[string]*domain.User),
		emailMap:  make(map[string]string),
		apiKeyMap: make(map[string]string),
	}
}

// Create stores a new user
func (r *InMemoryUserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	if _, exists := r.emailMap[user.Email]; exists {
		return fmt.Errorf("email already exists")
	}

	if user.APIKey != "" {
		if _, exists := r.apiKeyMap[user.APIKey]; exists {
			return fmt.Errorf("API key already exists")
		}
		r.apiKeyMap[user.APIKey] = user.ID
	}

	r.users[user.ID] = user
	r.emailMap[user.Email] = user.ID
	return nil
}

// GetByID retrieves a user by ID
func (r *InMemoryUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *InMemoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

// GetByAPIKey retrieves a user by API key
func (r *InMemoryUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.apiKeyMap[apiKey]
	if !exists {
		return nil, nil
	}
	return r.users[id], nil
}

// Update modifies an existing user
func (r *InMemoryUserRepository) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}

	// Update email mapping if changed
	oldUser := r.users[user.ID]
	if oldUser.Email != user.Email {
		delete(r.emailMap, oldUser.Email)
		r.emailMap[user.Email] = user.ID
	}

	// Update API key mapping if changed
	if oldUser.APIKey != user.APIKey {
		if oldUser.APIKey != "" {
			delete(r.apiKeyMap, oldUser.APIKey)
		}
		if user.APIKey != "" {
			r.apiKeyMap[user.APIKey] = user.ID
		}
	}

	r.users[user.ID] = user
	return nil
}

// Delete removes a user
func (r *InMemoryUserRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	delete(r.emailMap, user.Email)
	if user.APIKey != "" {
		delete(r.apiKeyMap, user.APIKey)
	}
	delete(r.users, id)
	return nil
}

// List retrieves users with pagination
func (r *InMemoryUserRepository) List(ctx context.Context, offset, limit int) ([]*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*domain.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	// Apply pagination
	if offset >= len(users) {
		return []*domain.User{}, nil
	}

	end := offset + limit
	if end > len(users) {
		end = len(users)
	}

	return users[offset:end], nil
}

// UpdateAPIKey updates the API key for a user
func (r *InMemoryUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[userID]
	if !exists {
		return domain.ErrUserNotFound
	}

	user.APIKey = apiKey
	r.users[userID] = user
	return nil
}

// Count returns the total number of users
func (r *InMemoryUserRepository) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return int64(len(r.users)), nil
}
