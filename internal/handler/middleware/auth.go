package middleware

import (
	"metadatatool/internal/pkg/domain"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware creates a new authentication middleware
func AuthMiddleware(authService domain.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		// Validate the token
		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		// Store claims in context for later use
		c.Locals("user", claims)
		return c.Next()
	}
}

// RoleGuard creates a middleware for role-based access control
func RoleGuard(roles ...domain.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("user").(*domain.Claims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing user claims",
			})
		}

		// Check if user's role is allowed
		for _, role := range roles {
			if claims.Role == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "insufficient permissions",
		})
	}
}

// APIKeyAuth creates a middleware for API key authentication
func APIKeyAuth(userRepo domain.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing API key",
			})
		}

		user, err := userRepo.GetByAPIKey(c.Context(), apiKey)
		if err != nil || user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid API key",
			})
		}

		// Store user in context
		c.Locals("user", &domain.Claims{
			UserID: user.ID,
			Email:  user.Email,
			Role:   user.Role,
		})
		return c.Next()
	}
}
