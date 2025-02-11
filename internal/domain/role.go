package domain

// Role represents a user role
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

// Permission represents a specific permission
type Permission string

const (
	PermissionReadTrack     Permission = "read:track"
	PermissionWriteTrack    Permission = "write:track"
	PermissionDeleteTrack   Permission = "delete:track"
	PermissionReadLabel     Permission = "read:label"
	PermissionWriteLabel    Permission = "write:label"
	PermissionDeleteLabel   Permission = "delete:label"
	PermissionManageAPIKeys Permission = "manage:api_keys"
)
