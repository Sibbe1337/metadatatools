package converter

import (
	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
)

// ToInternalRole converts pkg/domain.Role to domain.Role
func ToInternalRole(role pkgdomain.Role) domain.Role {
	return domain.Role(role)
}

// ToPkgRole converts domain.Role to pkg/domain.Role
func ToPkgRole(role domain.Role) pkgdomain.Role {
	return pkgdomain.Role(role)
}

// ToInternalPermission converts pkg/domain.Permission to domain.Permission
func ToInternalPermission(perm pkgdomain.Permission) domain.Permission {
	return domain.Permission(perm)
}

// ToPkgPermission converts domain.Permission to pkg/domain.Permission
func ToPkgPermission(perm domain.Permission) pkgdomain.Permission {
	return pkgdomain.Permission(perm)
}

// ToInternalPermissions converts []pkg/domain.Permission to []domain.Permission
func ToInternalPermissions(perms []pkgdomain.Permission) []domain.Permission {
	if perms == nil {
		return nil
	}
	result := make([]domain.Permission, len(perms))
	for i, p := range perms {
		result[i] = ToInternalPermission(p)
	}
	return result
}

// ToPkgPermissions converts []domain.Permission to []pkg/domain.Permission
func ToPkgPermissions(perms []domain.Permission) []pkgdomain.Permission {
	if perms == nil {
		return nil
	}
	result := make([]pkgdomain.Permission, len(perms))
	for i, p := range perms {
		result[i] = ToPkgPermission(p)
	}
	return result
}

// ToInternalUser converts pkg/domain.User to domain.User
func ToInternalUser(user *pkgdomain.User) *domain.User {
	if user == nil {
		return nil
	}
	return &domain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        domain.Role(user.Role),
		Permissions: ToInternalPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// ToPkgUser converts an internal domain user to a pkg domain user
func ToPkgUser(user *domain.User) *pkgdomain.User {
	if user == nil {
		return nil
	}
	return &pkgdomain.User{
		ID:          user.ID,
		Email:       user.Email,
		Password:    user.Password,
		Name:        user.Name,
		Role:        pkgdomain.Role(user.Role),
		Permissions: ToPkgPermissions(user.Permissions),
		Company:     user.Company,
		APIKey:      user.APIKey,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// ToInternalSession converts pkg/domain.Session to domain.Session
func ToInternalSession(session *pkgdomain.Session) *domain.Session {
	if session == nil {
		return nil
	}
	return &domain.Session{
		ID:          session.ID,
		UserID:      session.UserID,
		Role:        ToInternalRole(session.Role),
		Permissions: ToInternalPermissions(session.Permissions),
		ExpiresAt:   session.ExpiresAt,
		CreatedAt:   session.CreatedAt,
	}
}

// ToPkgSession converts domain.Session to pkg/domain.Session
func ToPkgSession(session *domain.Session) *pkgdomain.Session {
	if session == nil {
		return nil
	}
	return &pkgdomain.Session{
		ID:          session.ID,
		UserID:      session.UserID,
		Role:        ToPkgRole(session.Role),
		Permissions: ToPkgPermissions(session.Permissions),
		UserAgent:   "", // Not present in internal domain
		IP:          "", // Not present in internal domain
		ExpiresAt:   session.ExpiresAt,
		CreatedAt:   session.CreatedAt,
		LastSeenAt:  session.CreatedAt, // Use CreatedAt as initial LastSeenAt
	}
}

// ToInternalClaims converts internal domain claims to pkg domain claims
func ToInternalClaims(claims *pkgdomain.Claims) *domain.Claims {
	if claims == nil {
		return nil
	}
	return &domain.Claims{
		UserID:      claims.UserID,
		Role:        domain.Role(claims.Role),
		Permissions: ToInternalPermissions(claims.Permissions),
	}
}

// ToPkgClaims converts internal domain claims to pkg domain claims
func ToPkgClaims(claims *domain.Claims) *pkgdomain.Claims {
	if claims == nil {
		return nil
	}
	return &pkgdomain.Claims{
		UserID:      claims.UserID,
		Role:        pkgdomain.Role(claims.Role),
		Permissions: ToPkgPermissions(claims.Permissions),
	}
}

// ToInternalSessionConfig converts pkg/domain.SessionConfig to domain.SessionConfig
func ToInternalSessionConfig(cfg pkgdomain.SessionConfig) domain.SessionConfig {
	return domain.SessionConfig{
		CookieName:         cfg.CookieName,
		CookieDomain:       cfg.CookieDomain,
		CookiePath:         cfg.CookiePath,
		CookieSecure:       cfg.CookieSecure,
		CookieHTTPOnly:     cfg.CookieHTTPOnly,
		CookieSameSite:     cfg.CookieSameSite,
		SessionDuration:    cfg.SessionDuration,
		CleanupInterval:    cfg.CleanupInterval,
		MaxSessionsPerUser: cfg.MaxSessionsPerUser,
	}
}

// ToPkgSessionConfig converts domain.SessionConfig to pkg/domain.SessionConfig
func ToPkgSessionConfig(cfg domain.SessionConfig) pkgdomain.SessionConfig {
	return pkgdomain.SessionConfig{
		CookieName:         cfg.CookieName,
		CookieDomain:       cfg.CookieDomain,
		CookiePath:         cfg.CookiePath,
		CookieSecure:       cfg.CookieSecure,
		CookieHTTPOnly:     cfg.CookieHTTPOnly,
		CookieSameSite:     cfg.CookieSameSite,
		SessionDuration:    cfg.SessionDuration,
		CleanupInterval:    cfg.CleanupInterval,
		MaxSessionsPerUser: cfg.MaxSessionsPerUser,
	}
}

// ToInternalTrack converts pkg/domain.Track to domain.Track
func ToInternalTrack(track *pkgdomain.Track) *domain.Track {
	if track == nil {
		return nil
	}
	return &domain.Track{
		ID:        track.ID,
		LabelID:   track.LabelID,
		Status:    domain.TrackStatus(track.Status),
		CreatedAt: track.CreatedAt,
		UpdatedAt: track.UpdatedAt,
		Metadata:  ToInternalAIMetadata(&track.Metadata),
	}
}

// ToPkgTrack converts domain.Track to pkg/domain.Track
func ToPkgTrack(track *domain.Track) *pkgdomain.Track {
	if track == nil {
		return nil
	}
	return &pkgdomain.Track{
		ID:        track.ID,
		LabelID:   track.LabelID,
		Status:    pkgdomain.TrackStatus(track.Status),
		CreatedAt: track.CreatedAt,
		UpdatedAt: track.UpdatedAt,
		Metadata:  ToPkgCompleteTrackMetadata(track.Metadata),
	}
}

// ToInternalAIMetadata converts pkg/domain.CompleteTrackMetadata to domain.AIMetadata
func ToInternalAIMetadata(metadata *pkgdomain.CompleteTrackMetadata) *domain.AIMetadata {
	if metadata == nil {
		return nil
	}
	return &domain.AIMetadata{
		Title:         metadata.Title,
		Artist:        metadata.Artist,
		Album:         metadata.Album,
		Genre:         []string{metadata.Musical.Genre},
		Year:          metadata.Year,
		Confidence:    metadata.AI.Confidence,
		Language:      metadata.Additional.CustomFields["language"],
		Mood:          []string{metadata.Musical.Mood},
		Tempo:         metadata.Musical.BPM,
		Key:           metadata.Musical.Key,
		TimeSignature: metadata.Technical.Format.String(),
		Duration:      metadata.Duration,
		Tags:          metadata.AI.Tags,
	}
}

// ToPkgCompleteTrackMetadata converts domain.AIMetadata to pkg/domain.CompleteTrackMetadata
func ToPkgCompleteTrackMetadata(metadata *domain.AIMetadata) pkgdomain.CompleteTrackMetadata {
	if metadata == nil {
		return pkgdomain.CompleteTrackMetadata{}
	}

	var genre, mood string
	if len(metadata.Genre) > 0 {
		genre = metadata.Genre[0]
	}
	if len(metadata.Mood) > 0 {
		mood = metadata.Mood[0]
	}

	return pkgdomain.CompleteTrackMetadata{
		BasicTrackMetadata: pkgdomain.BasicTrackMetadata{
			Title:    metadata.Title,
			Artist:   metadata.Artist,
			Album:    metadata.Album,
			Year:     metadata.Year,
			Duration: metadata.Duration,
		},
		Musical: pkgdomain.MusicalMetadata{
			Genre: genre,
			BPM:   metadata.Tempo,
			Key:   metadata.Key,
			Mood:  mood,
		},
		Technical: pkgdomain.AudioTechnicalMetadata{
			Format: pkgdomain.AudioFormat(metadata.TimeSignature),
		},
		AI: &pkgdomain.TrackAIMetadata{
			Tags:       metadata.Tags,
			Confidence: metadata.Confidence,
			Version:    "1.0", // TODO: Get from config
		},
		Additional: pkgdomain.AdditionalMetadata{
			CustomFields: map[string]string{
				"language": metadata.Language,
			},
		},
	}
}

// ToInternalMetadata converts pkg/domain.Metadata to domain.Metadata
func ToInternalMetadata(metadata *pkgdomain.Metadata) *domain.Metadata {
	if metadata == nil {
		return nil
	}
	return &domain.Metadata{
		ISRC:         metadata.ISRC,
		ISWC:         metadata.ISWC,
		BPM:          metadata.BPM,
		Key:          metadata.Key,
		Mood:         metadata.Mood,
		Labels:       metadata.Labels,
		AITags:       metadata.AITags,
		Confidence:   metadata.Confidence,
		ModelVersion: metadata.ModelVersion,
		CustomFields: metadata.CustomFields,
	}
}

// ToPkgMetadata converts domain.Metadata to pkg/domain.Metadata
func ToPkgMetadata(metadata *domain.Metadata) *pkgdomain.Metadata {
	if metadata == nil {
		return nil
	}
	return &pkgdomain.Metadata{
		ISRC:         metadata.ISRC,
		ISWC:         metadata.ISWC,
		BPM:          metadata.BPM,
		Key:          metadata.Key,
		Mood:         metadata.Mood,
		Labels:       metadata.Labels,
		AITags:       metadata.AITags,
		Confidence:   metadata.Confidence,
		ModelVersion: metadata.ModelVersion,
		CustomFields: metadata.CustomFields,
	}
}
