package domain

// Permission represents a specific action that can be performed
type Permission string

const (
	// Track-related permissions
	PermissionCreateTrack Permission = "track:create"
	PermissionReadTrack   Permission = "track:read"
	PermissionUpdateTrack Permission = "track:update"
	PermissionDeleteTrack Permission = "track:delete"

	// Metadata-related permissions
	PermissionEnrichMetadata Permission = "metadata:enrich"
	PermissionExportDDEX     Permission = "metadata:export_ddex"

	// User management permissions
	PermissionManageUsers Permission = "users:manage"
	PermissionManageRoles Permission = "roles:manage"

	// API key permissions
	PermissionManageAPIKeys Permission = "apikeys:manage"
)

// RolePermissions maps roles to their allowed permissions
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		// Admin has all permissions
		PermissionCreateTrack, PermissionReadTrack, PermissionUpdateTrack, PermissionDeleteTrack,
		PermissionEnrichMetadata, PermissionExportDDEX,
		PermissionManageUsers, PermissionManageRoles,
		PermissionManageAPIKeys,
	},
	RoleUser: {
		// Regular user has basic track and metadata permissions
		PermissionCreateTrack, PermissionReadTrack, PermissionUpdateTrack,
		PermissionEnrichMetadata, PermissionExportDDEX,
	},
	RoleGuest: {
		// Guest can only read tracks
		PermissionReadTrack,
	},
	RoleSystem: {
		// System has all track and metadata permissions but no user management
		PermissionCreateTrack, PermissionReadTrack, PermissionUpdateTrack, PermissionDeleteTrack,
		PermissionEnrichMetadata, PermissionExportDDEX,
	},
}

// Claims represents JWT claims with added role information
type Claims struct {
	UserID      string
	Email       string
	Role        Role
	Permissions []Permission
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthService handles authentication and authorization
type AuthService interface {
	// GenerateTokens creates a new pair of access and refresh tokens
	GenerateTokens(user *User) (*TokenPair, error)

	// ValidateToken validates and parses a JWT token
	ValidateToken(token string) (*Claims, error)

	// RefreshToken validates a refresh token and generates new token pair
	RefreshToken(refreshToken string) (*TokenPair, error)

	// HashPassword creates a bcrypt hash of the password
	HashPassword(password string) (string, error)

	// VerifyPassword checks if the provided password matches the hash
	VerifyPassword(hashedPassword, password string) error

	// GenerateAPIKey creates a new API key
	GenerateAPIKey() (string, error)

	// HasPermission checks if a role has a specific permission
	HasPermission(role Role, permission Permission) bool

	// GetPermissions returns all permissions for a role
	GetPermissions(role Role) []Permission
}
