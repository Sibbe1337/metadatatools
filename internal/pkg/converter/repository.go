package converter

import (
	"context"
	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
)

// UserRepositoryWrapper adapts domain.UserRepository to pkg/domain.UserRepository and vice versa
type UserRepositoryWrapper struct {
	internal domain.UserRepository
	pkg      pkgdomain.UserRepository
}

func NewUserRepositoryWrapper(internal domain.UserRepository, pkg pkgdomain.UserRepository) *UserRepositoryWrapper {
	return &UserRepositoryWrapper{
		internal: internal,
		pkg:      pkg,
	}
}

// Internal returns the internal domain repository
func (w *UserRepositoryWrapper) Internal() domain.UserRepository {
	return w.internal
}

// Pkg returns the pkg/domain repository
func (w *UserRepositoryWrapper) Pkg() pkgdomain.UserRepository {
	return w.pkg
}

// Create implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) Create(ctx context.Context, user *pkgdomain.User) error {
	internalUser := ToInternalUser(user)
	return w.internal.Create(ctx, internalUser)
}

// GetByID implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) GetByID(ctx context.Context, id string) (*pkgdomain.User, error) {
	user, err := w.internal.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToPkgUser(user), nil
}

// GetByEmail implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) GetByEmail(ctx context.Context, email string) (*pkgdomain.User, error) {
	user, err := w.internal.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return ToPkgUser(user), nil
}

// GetByAPIKey implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) GetByAPIKey(ctx context.Context, apiKey string) (*pkgdomain.User, error) {
	user, err := w.internal.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	return ToPkgUser(user), nil
}

// Update implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) Update(ctx context.Context, user *pkgdomain.User) error {
	internalUser := ToInternalUser(user)
	return w.internal.Update(ctx, internalUser)
}

// Delete implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) Delete(ctx context.Context, id string) error {
	return w.internal.Delete(ctx, id)
}

// UpdateAPIKey implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) UpdateAPIKey(ctx context.Context, userID string, apiKey string) error {
	return w.internal.UpdateAPIKey(ctx, userID, apiKey)
}

// List implements pkg/domain.UserRepository
func (w *UserRepositoryWrapper) List(ctx context.Context, offset, limit int) ([]*pkgdomain.User, error) {
	users, err := w.internal.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	pkgUsers := make([]*pkgdomain.User, len(users))
	for i, user := range users {
		pkgUsers[i] = ToPkgUser(user)
	}
	return pkgUsers, nil
}

// SessionStoreWrapper adapts domain.SessionStore to pkg/domain.SessionStore and vice versa
type SessionStoreWrapper struct {
	internal domain.SessionStore
	pkg      pkgdomain.SessionStore
}

func NewSessionStoreWrapper(internal domain.SessionStore, pkg pkgdomain.SessionStore) *SessionStoreWrapper {
	return &SessionStoreWrapper{
		internal: internal,
		pkg:      pkg,
	}
}

// Internal returns the internal domain store
func (w *SessionStoreWrapper) Internal() domain.SessionStore {
	return w.internal
}

// Pkg returns the pkg/domain store
func (w *SessionStoreWrapper) Pkg() pkgdomain.SessionStore {
	return w.pkg
}

// Create implements pkg/domain.SessionStore
func (w *SessionStoreWrapper) Create(ctx context.Context, session *pkgdomain.Session) error {
	internalSession := ToInternalSession(session)
	return w.internal.Create(ctx, internalSession)
}

// Get implements pkg/domain.SessionStore
func (w *SessionStoreWrapper) Get(ctx context.Context, id string) (*pkgdomain.Session, error) {
	session, err := w.internal.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return ToPkgSession(session), nil
}

// Delete implements pkg/domain.SessionStore
func (w *SessionStoreWrapper) Delete(ctx context.Context, id string) error {
	return w.internal.Delete(ctx, id)
}

// Touch implements pkg/domain.SessionStore
func (w *SessionStoreWrapper) Touch(ctx context.Context, id string) error {
	return w.internal.Touch(ctx, id)
}

// GetUserSessions implements pkg/domain.SessionStore
func (w *SessionStoreWrapper) GetUserSessions(ctx context.Context, userID string) ([]*pkgdomain.Session, error) {
	sessions, err := w.internal.GetUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	pkgSessions := make([]*pkgdomain.Session, len(sessions))
	for i, session := range sessions {
		pkgSessions[i] = ToPkgSession(session)
	}
	return pkgSessions, nil
}

// DeleteUserSessions implements pkg/domain.SessionStore
func (w *SessionStoreWrapper) DeleteUserSessions(ctx context.Context, userID string) error {
	return w.internal.DeleteUserSessions(ctx, userID)
}

// AIServiceWrapper adapts domain.AIService to pkg/domain.AIService and vice versa
type AIServiceWrapper struct {
	internal domain.AIService
}

func NewAIServiceWrapper(internal domain.AIService) *AIServiceWrapper {
	return &AIServiceWrapper{
		internal: internal,
	}
}

// EnrichMetadata implements pkg/domain.AIService interface
func (w *AIServiceWrapper) EnrichMetadata(ctx context.Context, track *pkgdomain.Track) error {
	metadata, err := w.internal.EnrichMetadata(ctx, track.FilePath)
	if err != nil {
		return err
	}

	// Update track metadata with AI results
	if track.Metadata.AI == nil {
		track.Metadata.AI = &pkgdomain.TrackAIMetadata{}
	}
	track.Metadata.AI.Tags = metadata.Tags
	track.Metadata.AI.Confidence = metadata.Confidence
	track.Metadata.AI.Version = "1.0"                          // TODO: Get from config
	track.Metadata.AI.NeedsReview = metadata.Confidence < 0.85 // TODO: Get threshold from config
	return nil
}

// ValidateMetadata implements pkg/domain.AIService interface
func (w *AIServiceWrapper) ValidateMetadata(ctx context.Context, track *pkgdomain.Track) (float64, error) {
	metadata := &domain.AIMetadata{
		Title:         track.Metadata.Title,
		Artist:        track.Metadata.Artist,
		Album:         track.Metadata.Album,
		Genre:         []string{track.Metadata.Musical.Genre},
		Year:          track.Metadata.Year,
		Confidence:    track.Metadata.AI.Confidence,
		Language:      track.Metadata.Additional.CustomFields["language"],
		Mood:          []string{track.Metadata.Musical.Mood},
		Tempo:         track.Metadata.Musical.BPM,
		Key:           track.Metadata.Musical.Key,
		TimeSignature: track.Metadata.Technical.Format.String(),
		Duration:      track.Metadata.Duration,
		Tags:          track.Metadata.AI.Tags,
	}
	valid, err := w.internal.ValidateMetadata(ctx, metadata)
	if err != nil {
		return 0, err
	}
	if valid {
		return 1.0, nil
	}
	return 0.5, nil
}

// BatchProcess implements pkg/domain.AIService interface
func (w *AIServiceWrapper) BatchProcess(ctx context.Context, tracks []*pkgdomain.Track) error {
	for _, track := range tracks {
		if err := w.EnrichMetadata(ctx, track); err != nil {
			return err
		}
	}
	return nil
}

// Internal returns the internal domain service
func (w *AIServiceWrapper) Internal() domain.AIService {
	return w.internal
}

// Pkg returns this wrapper as it implements pkg/domain.AIService
func (w *AIServiceWrapper) Pkg() pkgdomain.AIService {
	return w
}

// TrackRepositoryWrapper adapts domain.TrackRepository to pkg/domain.TrackRepository and vice versa
type TrackRepositoryWrapper struct {
	internal domain.TrackRepository
	pkg      pkgdomain.TrackRepository
}

func NewTrackRepositoryWrapper(internal domain.TrackRepository, pkg pkgdomain.TrackRepository) *TrackRepositoryWrapper {
	return &TrackRepositoryWrapper{
		internal: internal,
		pkg:      pkg,
	}
}

// Internal returns the internal domain repository
func (w *TrackRepositoryWrapper) Internal() domain.TrackRepository {
	return w.internal
}

// Pkg returns the pkg/domain repository
func (w *TrackRepositoryWrapper) Pkg() pkgdomain.TrackRepository {
	return w.pkg
}

// AuthServiceWrapper adapts domain.AuthService to pkg/domain.AuthService and vice versa
type AuthServiceWrapper struct {
	internal domain.AuthService
	pkg      pkgdomain.AuthService
}

func NewAuthServiceWrapper(internal domain.AuthService, pkg pkgdomain.AuthService) *AuthServiceWrapper {
	return &AuthServiceWrapper{
		internal: internal,
		pkg:      pkg,
	}
}

// Internal returns the internal domain service
func (w *AuthServiceWrapper) Internal() domain.AuthService {
	return w.internal
}

// Pkg returns the pkg/domain service
func (w *AuthServiceWrapper) Pkg() pkgdomain.AuthService {
	return w.pkg
}

// GenerateTokens implements pkg/domain.AuthService
func (w *AuthServiceWrapper) GenerateTokens(user *pkgdomain.User) (*pkgdomain.TokenPair, error) {
	internalUser := ToInternalUser(user)
	tokens, err := w.internal.GenerateTokens(internalUser)
	if err != nil {
		return nil, err
	}
	return &pkgdomain.TokenPair{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

// ValidateToken implements pkg/domain.AuthService
func (w *AuthServiceWrapper) ValidateToken(token string) (*pkgdomain.Claims, error) {
	claims, err := w.internal.ValidateToken(context.Background(), token)
	if err != nil {
		return nil, err
	}
	return ToPkgClaims(&claims.Claims), nil
}

// RefreshToken implements pkg/domain.AuthService
func (w *AuthServiceWrapper) RefreshToken(refreshToken string) (*pkgdomain.TokenPair, error) {
	return w.pkg.RefreshToken(refreshToken)
}

// HashPassword implements pkg/domain.AuthService
func (w *AuthServiceWrapper) HashPassword(password string) (string, error) {
	return w.internal.HashPassword(password)
}

// VerifyPassword implements pkg/domain.AuthService
func (w *AuthServiceWrapper) VerifyPassword(hashedPassword, password string) error {
	return w.internal.VerifyPassword(hashedPassword, password)
}

// GenerateAPIKey implements pkg/domain.AuthService
func (w *AuthServiceWrapper) GenerateAPIKey() (string, error) {
	return w.internal.GenerateAPIKey()
}

// HasPermission implements pkg/domain.AuthService
func (w *AuthServiceWrapper) HasPermission(role pkgdomain.Role, permission pkgdomain.Permission) bool {
	return w.pkg.HasPermission(role, permission)
}

// GetPermissions implements pkg/domain.AuthService
func (w *AuthServiceWrapper) GetPermissions(role pkgdomain.Role) []pkgdomain.Permission {
	return w.pkg.GetPermissions(role)
}
