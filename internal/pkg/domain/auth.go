package domain

// Claims represents JWT claims
type Claims struct {
	UserID string
	Email  string
	Role   Role
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
}
