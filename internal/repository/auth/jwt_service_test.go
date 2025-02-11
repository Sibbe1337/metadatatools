package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metadatatool/internal/pkg/domain"
)

func createTestUser() *domain.User {
	return &domain.User{
		ID:          uuid.New().String(),
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        domain.RoleUser,
		Permissions: domain.RolePermissions[domain.RoleUser],
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func TestNewJWTService(t *testing.T) {
	t.Run("with provided key", func(t *testing.T) {
		key := []byte("test-secret-key")
		service, err := NewJWTService(key)
		require.NoError(t, err)
		assert.Equal(t, key, service.secretKey)
	})

	t.Run("with auto-generated key", func(t *testing.T) {
		service, err := NewJWTService(nil)
		require.NoError(t, err)
		assert.Len(t, service.secretKey, SecretKeyLength)
	})
}

func TestJWTService_GenerateTokens(t *testing.T) {
	service, err := NewJWTService([]byte("test-secret-key"))
	require.NoError(t, err)

	t.Run("successful token generation", func(t *testing.T) {
		user := createTestUser()
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)

		// Validate access token
		claims, err := service.ValidateToken(tokens.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, user.Permissions, claims.Permissions)

		// Validate refresh token
		claims, err = service.ValidateToken(tokens.RefreshToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
	})
}

func TestJWTService_ValidateToken(t *testing.T) {
	service, err := NewJWTService([]byte("test-secret-key"))
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		user := createTestUser()
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)

		claims, err := service.ValidateToken(tokens.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := service.ValidateToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create a service with very short token duration for testing
		shortDurationService := &JWTService{
			secretKey: []byte("test-secret-key"),
		}

		user := createTestUser()
		token, err := shortDurationService.createToken(user, time.Millisecond)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(time.Millisecond * 2)

		_, err = shortDurationService.ValidateToken(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is expired")
	})
}

func TestJWTService_RefreshToken(t *testing.T) {
	service, err := NewJWTService([]byte("test-secret-key"))
	require.NoError(t, err)

	t.Run("successful refresh", func(t *testing.T) {
		user := createTestUser()
		tokens, err := service.GenerateTokens(user)
		require.NoError(t, err)

		newTokens, err := service.RefreshToken(tokens.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEmpty(t, newTokens.RefreshToken)
		assert.NotEqual(t, tokens.AccessToken, newTokens.AccessToken)
		assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := service.RefreshToken("invalid-token")
		assert.Error(t, err)
	})
}

func TestJWTService_PasswordHashing(t *testing.T) {
	service, err := NewJWTService([]byte("test-secret-key"))
	require.NoError(t, err)

	t.Run("password hashing and verification", func(t *testing.T) {
		password := "test-password"

		// Hash password
		hash, err := service.HashPassword(password)
		require.NoError(t, err)
		assert.NotEqual(t, password, hash)

		// Verify correct password
		err = service.VerifyPassword(hash, password)
		assert.NoError(t, err)

		// Verify incorrect password
		err = service.VerifyPassword(hash, "wrong-password")
		assert.Error(t, err)
	})
}

func TestJWTService_GenerateAPIKey(t *testing.T) {
	service, err := NewJWTService([]byte("test-secret-key"))
	require.NoError(t, err)

	t.Run("api key generation", func(t *testing.T) {
		key1, err := service.GenerateAPIKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key1)

		key2, err := service.GenerateAPIKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key2)

		// Keys should be different
		assert.NotEqual(t, key1, key2)
	})
}

func TestJWTService_Permissions(t *testing.T) {
	service, err := NewJWTService([]byte("test-secret-key"))
	require.NoError(t, err)

	t.Run("check permissions", func(t *testing.T) {
		// Admin should have all permissions
		assert.True(t, service.HasPermission(domain.RoleAdmin, domain.PermissionManageUsers))
		assert.True(t, service.HasPermission(domain.RoleAdmin, domain.PermissionCreateTrack))

		// Regular user should have limited permissions
		assert.True(t, service.HasPermission(domain.RoleUser, domain.PermissionReadTrack))
		assert.False(t, service.HasPermission(domain.RoleUser, domain.PermissionManageUsers))

		// Guest should only have read permission
		assert.True(t, service.HasPermission(domain.RoleGuest, domain.PermissionReadTrack))
		assert.False(t, service.HasPermission(domain.RoleGuest, domain.PermissionCreateTrack))
	})

	t.Run("get permissions", func(t *testing.T) {
		adminPerms := service.GetPermissions(domain.RoleAdmin)
		assert.Contains(t, adminPerms, domain.PermissionManageUsers)
		assert.Contains(t, adminPerms, domain.PermissionCreateTrack)

		userPerms := service.GetPermissions(domain.RoleUser)
		assert.Contains(t, userPerms, domain.PermissionReadTrack)
		assert.NotContains(t, userPerms, domain.PermissionManageUsers)

		guestPerms := service.GetPermissions(domain.RoleGuest)
		assert.Contains(t, guestPerms, domain.PermissionReadTrack)
		assert.NotContains(t, guestPerms, domain.PermissionCreateTrack)
	})
}
