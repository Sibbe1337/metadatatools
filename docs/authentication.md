# Authentication System Documentation

## Overview
The authentication system provides a secure, JWT-based authentication mechanism with role-based access control (RBAC) and session management. It implements industry best practices for token management, password security, and API key handling.

## Components

### JWT Service (`internal/repository/auth/jwt_service.go`)

The JWT service handles token generation, validation, and management. It implements the `domain.AuthService` interface.

#### Key Features
- Token pair generation (access + refresh tokens)
- Secure token validation
- Password hashing using bcrypt
- API key generation
- Role-based permission management

#### Token Configuration
```go
const (
    accessTokenDuration  = 15 * time.Minute
    refreshTokenDuration = 7 * 24 * time.Hour
)
```

### Token Types

#### Access Token
- Short-lived (15 minutes)
- Used for API authentication
- Contains user claims:
  - User ID
  - Email
  - Role
  - Permissions
  - Standard JWT claims (exp, iat, nbf, jti)

#### Refresh Token
- Long-lived (7 days)
- Used to obtain new access tokens
- Contains minimal claims for security

### Claims Structure
```go
type customClaims struct {
    jwt.RegisteredClaims
    UserID      string
    Email       string
    Role        domain.Role
    Permissions []domain.Permission
}
```

### Roles and Permissions

The system implements a hierarchical role-based access control:

#### Roles
- `RoleAdmin`: Full system access
- `RoleUser`: Standard user access
- `RoleGuest`: Limited read-only access
- `RoleSystem`: System-level operations

#### Permissions
```go
const (
    PermissionCreateTrack    = "track:create"
    PermissionReadTrack      = "track:read"
    PermissionUpdateTrack    = "track:update"
    PermissionDeleteTrack    = "track:delete"
    PermissionEnrichMetadata = "metadata:enrich"
    PermissionExportDDEX     = "metadata:export_ddex"
    PermissionManageUsers    = "users:manage"
    PermissionManageRoles    = "roles:manage"
    PermissionManageAPIKeys  = "apikeys:manage"
)
```

## Usage Examples

### Token Generation
```go
authService, err := auth.NewJWTService(secretKey)
if err != nil {
    return err
}

tokens, err := authService.GenerateTokens(user)
if err != nil {
    return err
}

// Use tokens.AccessToken and tokens.RefreshToken
```

### Token Validation
```go
claims, err := authService.ValidateToken(tokenString)
if err != nil {
    return err
}

// Access claims.UserID, claims.Role, etc.
```

### Password Management
```go
// Hash password
hashedPassword, err := authService.HashPassword(password)
if err != nil {
    return err
}

// Verify password
err = authService.VerifyPassword(hashedPassword, password)
if err != nil {
    // Invalid password
}
```

### Permission Checking
```go
if authService.HasPermission(user.Role, domain.PermissionCreateTrack) {
    // User has permission to create tracks
}
```

## Security Considerations

1. **Token Security**
   - Tokens include unique JWT IDs (jti)
   - Implements token expiration
   - Uses secure random generation for keys
   - Validates signing method

2. **Password Security**
   - Uses bcrypt with appropriate cost factor
   - Implements secure password hashing
   - Never stores plain text passwords

3. **API Key Security**
   - Uses cryptographically secure random generation
   - Implements base64 URL-safe encoding
   - 32-byte length for strong security

## Testing

The authentication system includes comprehensive tests:
- Token generation and validation
- Password hashing and verification
- Permission checks
- API key generation
- Error cases and edge conditions

Run tests with:
```bash
go test -v ./internal/repository/auth
```

## Metrics and Monitoring

The system exports Prometheus metrics for:
- Authentication attempts
- Token operations
- Session management
- Permission checks

Metric names:
- `auth_attempts_total`
- `token_operations_total`
- `session_operations_total`
- `permission_checks_total`

## Error Handling

The system provides clear error types for different failure scenarios:
- `ErrInvalidCredentials`
- `ErrInvalidToken`
- `ErrSessionNotFound`
- `ErrUnauthorized`
- `ErrForbidden`

## Best Practices

1. **Token Management**
   - Use short-lived access tokens
   - Implement token refresh flow
   - Validate tokens on every request
   - Include minimal claims in refresh tokens

2. **Security**
   - Store secrets securely
   - Use environment variables for configuration
   - Implement rate limiting
   - Log security events

3. **Implementation**
   - Follow interface segregation
   - Implement proper error handling
   - Use dependency injection
   - Maintain test coverage 