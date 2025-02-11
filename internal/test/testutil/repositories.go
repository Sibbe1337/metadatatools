package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	pkgdomain "metadatatool/internal/pkg/domain"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryPkgUserRepository implements pkg/domain.UserRepository for testing
type InMemoryPkgUserRepository struct {
	users     map[string]*pkgdomain.User // ID -> User
	emailMap  map[string]string          // Email -> ID
	apiKeyMap map[string]string          // APIKey -> ID
	mu        sync.RWMutex
}

// NewInMemoryPkgUserRepository creates a new in-memory user repository for pkg/domain
func NewInMemoryPkgUserRepository() pkgdomain.UserRepository {
	return &InMemoryPkgUserRepository{
		users:     make(map[string]*pkgdomain.User),
		emailMap:  make(map[string]string),
		apiKeyMap: make(map[string]string),
	}
}

func (r *InMemoryPkgUserRepository) Create(ctx context.Context, user *pkgdomain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user.ID == "" {
		user.ID = uuid.NewString()
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

func (r *InMemoryPkgUserRepository) GetByID(ctx context.Context, id string) (*pkgdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (r *InMemoryPkgUserRepository) GetByEmail(ctx context.Context, email string) (*pkgdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.emailMap[email]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return r.users[id], nil
}

func (r *InMemoryPkgUserRepository) GetByAPIKey(ctx context.Context, apiKey string) (*pkgdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, exists := r.apiKeyMap[apiKey]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return r.users[id], nil
}

func (r *InMemoryPkgUserRepository) Update(ctx context.Context, user *pkgdomain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return fmt.Errorf("user not found")
	}

	oldUser := r.users[user.ID]
	if oldUser.Email != user.Email {
		delete(r.emailMap, oldUser.Email)
		r.emailMap[user.Email] = user.ID
	}

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

func (r *InMemoryPkgUserRepository) Delete(ctx context.Context, id string) error {
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

func (r *InMemoryPkgUserRepository) List(ctx context.Context, offset, limit int) ([]*pkgdomain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*pkgdomain.User, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	if offset >= len(users) {
		return []*pkgdomain.User{}, nil
	}

	end := offset + limit
	if end > len(users) {
		end = len(users)
	}

	return users[offset:end], nil
}

func (r *InMemoryPkgUserRepository) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[userID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	if user.APIKey != "" {
		delete(r.apiKeyMap, user.APIKey)
	}

	user.APIKey = apiKey
	if apiKey != "" {
		r.apiKeyMap[apiKey] = userID
	}

	return nil
}

// InMemoryPkgSessionRepository implements pkg/domain.SessionRepository for testing
type InMemoryPkgSessionRepository struct {
	sessions map[string]*pkgdomain.Session // ID -> Session
	mu       sync.RWMutex
}

// NewInMemoryPkgSessionRepository creates a new in-memory session repository for pkg/domain
func NewInMemoryPkgSessionRepository() pkgdomain.SessionRepository {
	return &InMemoryPkgSessionRepository{
		sessions: make(map[string]*pkgdomain.Session),
	}
}

func (r *InMemoryPkgSessionRepository) Create(ctx context.Context, session *pkgdomain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if session.ID == "" {
		session.ID = uuid.NewString()
	}

	r.sessions[session.ID] = session
	return nil
}

func (r *InMemoryPkgSessionRepository) GetByID(ctx context.Context, id string) (*pkgdomain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (r *InMemoryPkgSessionRepository) Update(ctx context.Context, session *pkgdomain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[session.ID]; !exists {
		return fmt.Errorf("session not found")
	}

	r.sessions[session.ID] = session
	return nil
}

func (r *InMemoryPkgSessionRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[id]; !exists {
		return fmt.Errorf("session not found")
	}

	delete(r.sessions, id)
	return nil
}

func (r *InMemoryPkgSessionRepository) List(ctx context.Context, offset, limit int) ([]*pkgdomain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*pkgdomain.Session, 0, len(r.sessions))
	for _, session := range r.sessions {
		sessions = append(sessions, session)
	}

	if offset >= len(sessions) {
		return []*pkgdomain.Session{}, nil
	}

	end := offset + limit
	if end > len(sessions) {
		end = len(sessions)
	}

	return sessions[offset:end], nil
}

func (r *InMemoryPkgSessionRepository) DeleteUserSessions(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, session := range r.sessions {
		if session.UserID == userID {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *InMemoryPkgSessionRepository) GetUserSessions(ctx context.Context, userID string) ([]*pkgdomain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var userSessions []*pkgdomain.Session
	for _, session := range r.sessions {
		if session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions, nil
}

// InMemoryPkgSessionStore implements pkg/domain.SessionStore for testing
type InMemoryPkgSessionStore struct {
	sessions     map[string]*pkgdomain.Session // SessionID -> Session
	userSessions map[string][]string           // UserID -> []SessionID
	mu           sync.RWMutex
}

// NewInMemoryPkgSessionStore creates a new in-memory session store for pkg/domain
func NewInMemoryPkgSessionStore() pkgdomain.SessionStore {
	return &InMemoryPkgSessionStore{
		sessions:     make(map[string]*pkgdomain.Session),
		userSessions: make(map[string][]string),
	}
}

func (s *InMemoryPkgSessionStore) Create(ctx context.Context, session *pkgdomain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session.ID == "" {
		session.ID = uuid.NewString()
	}

	s.sessions[session.ID] = session
	s.userSessions[session.UserID] = append(s.userSessions[session.UserID], session.ID)
	return nil
}

func (s *InMemoryPkgSessionStore) Get(ctx context.Context, id string) (*pkgdomain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (s *InMemoryPkgSessionStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return nil
	}

	sessions := s.userSessions[session.UserID]
	for i, sid := range sessions {
		if sid == id {
			s.userSessions[session.UserID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}

	delete(s.sessions, id)
	return nil
}

func (s *InMemoryPkgSessionStore) Touch(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[id]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.LastSeenAt = time.Now()
	return nil
}

func (s *InMemoryPkgSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*pkgdomain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionIDs := s.userSessions[userID]
	sessions := make([]*pkgdomain.Session, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		if session, exists := s.sessions[id]; exists {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (s *InMemoryPkgSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionIDs := s.userSessions[userID]
	for _, id := range sessionIDs {
		delete(s.sessions, id)
	}
	delete(s.userSessions, userID)
	return nil
}

// InMemoryPkgAuthService implements pkg/domain.AuthService for testing
type InMemoryPkgAuthService struct {
	tokens    map[string]string // TokenID -> UserID
	secretKey []byte
	mu        sync.RWMutex
}

// NewInMemoryPkgAuthService creates a new in-memory auth service for pkg/domain
func NewInMemoryPkgAuthService() pkgdomain.AuthService {
	return &InMemoryPkgAuthService{
		tokens:    make(map[string]string),
		secretKey: []byte("test-secret-key"),
	}
}

func (s *InMemoryPkgAuthService) GenerateTokens(user *pkgdomain.User) (*pkgdomain.TokenPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenID := uuid.NewString()
	s.tokens[tokenID] = user.ID

	return &pkgdomain.TokenPair{
		AccessToken:  tokenID,
		RefreshToken: uuid.NewString(),
	}, nil
}

func (s *InMemoryPkgAuthService) ValidateToken(token string) (*pkgdomain.Claims, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, exists := s.tokens[token]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}

	return &pkgdomain.Claims{
		UserID: userID,
		Role:   pkgdomain.RoleUser,
	}, nil
}

func (s *InMemoryPkgAuthService) RefreshToken(refreshToken string) (*pkgdomain.TokenPair, error) {
	return &pkgdomain.TokenPair{
		AccessToken:  uuid.NewString(),
		RefreshToken: uuid.NewString(),
	}, nil
}

func (s *InMemoryPkgAuthService) HashPassword(password string) (string, error) {
	return fmt.Sprintf("hashed-%s", password), nil
}

func (s *InMemoryPkgAuthService) VerifyPassword(hashedPassword, password string) error {
	if hashedPassword != fmt.Sprintf("hashed-%s", password) {
		return fmt.Errorf("invalid password")
	}
	return nil
}

func (s *InMemoryPkgAuthService) GenerateAPIKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return hex.EncodeToString(key), nil
}

func (s *InMemoryPkgAuthService) HasPermission(role pkgdomain.Role, permission pkgdomain.Permission) bool {
	permissions := s.GetPermissions(role)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (s *InMemoryPkgAuthService) GetPermissions(role pkgdomain.Role) []pkgdomain.Permission {
	switch role {
	case pkgdomain.RoleAdmin:
		return []pkgdomain.Permission{
			pkgdomain.PermissionCreateTrack,
			pkgdomain.PermissionReadTrack,
			pkgdomain.PermissionUpdateTrack,
			pkgdomain.PermissionDeleteTrack,
			pkgdomain.PermissionEnrichMetadata,
			pkgdomain.PermissionExportDDEX,
			pkgdomain.PermissionManageUsers,
			pkgdomain.PermissionManageRoles,
			pkgdomain.PermissionManageAPIKeys,
		}
	case pkgdomain.RoleUser:
		return []pkgdomain.Permission{
			pkgdomain.PermissionCreateTrack,
			pkgdomain.PermissionReadTrack,
			pkgdomain.PermissionUpdateTrack,
			pkgdomain.PermissionEnrichMetadata,
			pkgdomain.PermissionExportDDEX,
		}
	default:
		return []pkgdomain.Permission{
			pkgdomain.PermissionReadTrack,
		}
	}
}
