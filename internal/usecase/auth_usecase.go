package usecase

import (
	"context"
	"errors"
	"fmt"
	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"time"

	"github.com/google/uuid"
)

// AuthUseCaseInterface defines the interface for authentication operations
type AuthUseCaseInterface interface {
	Register(ctx context.Context, input RegisterInput) (*domain.User, error)
	Login(ctx context.Context, input LoginInput) (*LoginOutput, error)
	Logout(ctx context.Context, sessionID string) error
	ValidateToken(ctx context.Context, tokenString string) (*domain.User, error)
	RefreshToken(ctx context.Context, refreshToken string) (string, string, *domain.User, error)
	GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeAllSessions(ctx context.Context, userID string) error
	GetSession(ctx context.Context, sessionID string) (*domain.Session, error)
	GenerateAPIKey(ctx context.Context, userID string) (string, error)
	HasPermission(user *domain.User, permission pkgdomain.Permission) bool
	CreateSession(ctx context.Context, session *domain.Session) error
}

// AuthUseCase handles authentication operations
type AuthUseCase struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionStore
	authService domain.AuthService
}

// NewAuthUseCase creates a new auth use case
func NewAuthUseCase(userRepo domain.UserRepository, sessionRepo domain.SessionStore, authService domain.AuthService) *AuthUseCase {
	return &AuthUseCase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		authService: authService,
	}
}

// RegisterInput represents registration request data
type RegisterInput struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
	Name     string      `json:"name"`
}

// Register creates a new user account
func (uc *AuthUseCase) Register(ctx context.Context, input RegisterInput) (*domain.User, error) {
	// Check if email already exists
	existingUser, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("error checking existing user: %w", err)
	}
	if existingUser != nil {
		return nil, domain.ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := uc.authService.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	// Create user
	now := time.Now()
	user := &domain.User{
		Email:     input.Email,
		Password:  hashedPassword,
		Role:      input.Role,
		Name:      input.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return user, nil
}

// LoginInput represents login request data
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginOutput contains the result of a successful login
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
	Session      *domain.Session
}

// Login authenticates a user and returns tokens
func (uc *AuthUseCase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Get user by email
	user, err := uc.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	// Verify password
	if err := uc.authService.VerifyPassword(user.Password, input.Password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := uc.authService.GenerateTokens(user)
	if err != nil {
		return nil, fmt.Errorf("error generating tokens: %w", err)
	}

	// Create session
	session := &domain.Session{
		ID:          uuid.New().String(),
		UserID:      user.ID,
		Role:        user.Role,
		Permissions: user.Permissions,
		UserAgent:   "", // This should be set by the handler
		IP:          "", // This should be set by the handler
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("error creating session: %w", err)
	}

	return &LoginOutput{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         user,
		Session:      session,
	}, nil
}

// CreateSession creates a new session
func (uc *AuthUseCase) CreateSession(ctx context.Context, session *domain.Session) error {
	return uc.sessionRepo.Create(ctx, session)
}

// Logout ends a user session
func (uc *AuthUseCase) Logout(ctx context.Context, sessionID string) error {
	return uc.sessionRepo.Delete(ctx, sessionID)
}

// ValidateToken validates a JWT token and returns the associated user
func (uc *AuthUseCase) ValidateToken(ctx context.Context, tokenString string) (*domain.User, error) {
	claims, err := uc.authService.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("error validating token: %w", err)
	}

	// Get user
	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("error getting user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

// GetUserSessions retrieves all active sessions for a user
func (uc *AuthUseCase) GetUserSessions(ctx context.Context, userID string) ([]*domain.Session, error) {
	return uc.sessionRepo.GetUserSessions(ctx, userID)
}

// RevokeSession revokes a specific session
func (uc *AuthUseCase) RevokeSession(ctx context.Context, sessionID string) error {
	return uc.sessionRepo.Delete(ctx, sessionID)
}

// RevokeAllSessions revokes all sessions for a user
func (uc *AuthUseCase) RevokeAllSessions(ctx context.Context, userID string) error {
	return uc.sessionRepo.DeleteUserSessions(ctx, userID)
}

// RefreshToken refreshes an access token using a refresh token
func (uc *AuthUseCase) RefreshToken(ctx context.Context, refreshToken string) (string, string, *domain.User, error) {
	// Validate refresh token
	claims, err := uc.authService.ValidateToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) {
			return "", "", nil, domain.ErrInvalidToken
		}
		return "", "", nil, fmt.Errorf("error validating token: %w", err)
	}

	// Get user
	user, err := uc.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", "", nil, domain.ErrUserNotFound
		}
		return "", "", nil, fmt.Errorf("error getting user: %w", err)
	}
	if user == nil {
		return "", "", nil, domain.ErrUserNotFound
	}

	// Generate new tokens
	tokens, err := uc.authService.GenerateTokens(user)
	if err != nil {
		return "", "", nil, fmt.Errorf("error generating tokens: %w", err)
	}

	return tokens.AccessToken, tokens.RefreshToken, user, nil
}

// GetSession retrieves a session by ID
func (uc *AuthUseCase) GetSession(ctx context.Context, sessionID string) (*domain.Session, error) {
	session, err := uc.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("error getting session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// GenerateAPIKey generates a new API key for a user
func (uc *AuthUseCase) GenerateAPIKey(ctx context.Context, userID string) (string, error) {
	// Generate a new API key
	apiKey, err := uc.authService.GenerateAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Update user with new API key
	if err := uc.userRepo.UpdateAPIKey(ctx, userID, apiKey); err != nil {
		return "", fmt.Errorf("failed to update API key: %w", err)
	}

	return apiKey, nil
}

// HasPermission checks if a user has a specific permission
func (uc *AuthUseCase) HasPermission(user *domain.User, permission pkgdomain.Permission) bool {
	// Admin role has all permissions
	if user.Role == domain.RoleAdmin {
		return true
	}

	// Convert domain role to pkg domain role
	pkgRole := pkgdomain.Role(user.Role)

	// Check role permissions from pkg domain
	permissions := pkgdomain.RolePermissions[pkgRole]
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
