package middleware

import (
	"metadatatool/internal/pkg/domain"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Auth middleware validates JWT tokens and sets claims in the context
func Auth(authService domain.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix if present
		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

// RequireRole middleware ensures the user has the required role
func RequireRole(requiredRole domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No role found"})
			return
		}

		userRole, ok := role.(domain.Role)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid role type"})
			return
		}

		// Admin role can access everything
		if userRole == domain.RoleAdmin {
			c.Next()
			return
		}

		// For other roles, check if they match the required role
		if userRole != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}

		c.Next()
	}
}

// RequirePermission middleware ensures the user has the required permission
func RequirePermission(permission domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No claims found"})
			return
		}

		userClaims := claims.(*domain.Claims)
		for _, p := range userClaims.Permissions {
			if p == permission {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	}
}

// APIKeyAuth middleware validates API keys
func APIKeyAuth(userRepo domain.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		user, err := userRepo.GetByAPIKey(c.Request.Context(), apiKey)
		if err != nil || user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set("user", user)
		c.Set("role", user.Role)
		c.Next()
	}
}
