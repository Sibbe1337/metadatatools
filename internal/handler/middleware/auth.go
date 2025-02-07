package middleware

import (
	"metadatatool/internal/pkg/domain"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Auth creates a middleware for JWT authentication
func Auth(authService domain.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Store claims in context for later use
		c.Set("user", claims)
		c.Next()
	}
}

// RequirePermission creates a middleware that checks for specific permissions
func RequirePermission(authService domain.AuthService, requiredPermission domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userClaims := claims.(*domain.Claims)
		if !authService.HasPermission(userClaims.Role, requiredPermission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole creates a middleware that checks for specific roles
func RequireRole(requiredRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		userClaims := claims.(*domain.Claims)
		if userClaims.Role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// APIKeyAuth creates a middleware for API key authentication
func APIKeyAuth(userRepo domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			c.Abort()
			return
		}

		user, err := userRepo.GetByAPIKey(c, apiKey)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			c.Abort()
			return
		}

		// Create claims for API key authentication
		claims := &domain.Claims{
			UserID:      user.ID,
			Email:       user.Email,
			Role:        user.Role,
			Permissions: domain.RolePermissions[user.Role],
		}

		// Store claims in context
		c.Set("user", claims)
		c.Next()
	}
}
