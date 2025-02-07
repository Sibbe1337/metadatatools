package middleware

import (
	"metadatatool/internal/pkg/domain"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Session creates a middleware for session management
func Session(sessionStore domain.SessionStore, cfg *domain.SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get session ID from cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			// No session cookie, create new session after the request
			c.Next()
			return
		}

		// Get session from store
		session, err := sessionStore.Get(c, sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
			c.Abort()
			return
		}

		if session == nil {
			// Invalid or expired session, remove cookie
			c.SetCookie("session_id", "", -1, "/", "", true, true)
			c.Next()
			return
		}

		// Update last seen time
		if err := sessionStore.Touch(c, sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
			c.Abort()
			return
		}

		// Store session in context
		c.Set("session", session)
		c.Next()
	}
}

// CreateSession creates a new session after successful authentication
func CreateSession(sessionStore domain.SessionStore, cfg *domain.SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user claims from context (set by Auth middleware)
		claims, exists := c.Get("user")
		if !exists {
			c.Next()
			return
		}

		userClaims := claims.(*domain.Claims)

		// Create new session
		session := &domain.Session{
			ID:         uuid.NewString(),
			UserID:     userClaims.UserID,
			UserAgent:  c.Request.UserAgent(),
			IP:         c.ClientIP(),
			CreatedAt:  time.Now(),
			LastSeenAt: time.Now(),
			ExpiresAt:  time.Now().Add(cfg.SessionDuration),
		}

		if err := sessionStore.Create(c, session); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			c.Abort()
			return
		}

		// Set session cookie
		c.SetCookie(
			"session_id",
			session.ID,
			int(cfg.SessionDuration.Seconds()),
			"/",
			"",
			true, // Secure
			true, // HttpOnly
		)

		c.Next()
	}
}

// ClearSession removes the current session
func ClearSession(sessionStore domain.SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.Next()
			return
		}

		if err := sessionStore.Delete(c, sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear session"})
			c.Abort()
			return
		}

		// Remove cookie
		c.SetCookie("session_id", "", -1, "/", "", true, true)
		c.Next()
	}
}

// RequireSession ensures a valid session exists
func RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, exists := c.Get("session")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
