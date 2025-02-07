package middleware

import (
	"metadatatool/internal/pkg/domain"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a new authentication middleware
func AuthMiddleware(authService domain.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing authorization header",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Validate the token
		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token",
			})
			c.Abort()
			return
		}

		// Store claims in context for later use
		c.Set("user", claims)
		c.Next()
	}
}

// RoleGuard creates a middleware for role-based access control
func RoleGuard(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing user claims",
			})
			c.Abort()
			return
		}

		userClaims, ok := claims.(*domain.Claims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "invalid user claims",
			})
			c.Abort()
			return
		}

		// Check if user's role is allowed
		for _, role := range roles {
			if userClaims.Role == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "insufficient permissions",
		})
		c.Abort()
	}
}

// APIKeyAuth creates a middleware for API key authentication
func APIKeyAuth(userRepo domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing API key",
			})
			c.Abort()
			return
		}

		user, err := userRepo.GetByAPIKey(c, apiKey)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", &domain.Claims{
			UserID: user.ID,
			Email:  user.Email,
			Role:   user.Role,
		})
		c.Next()
	}
}
