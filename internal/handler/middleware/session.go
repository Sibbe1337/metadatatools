package middleware

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Session middleware attaches the session to the context if present
func Session(store domain.SessionStore, config domain.SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(config.CookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read session cookie"})
			return
		}

		session, err := store.Get(c.Request.Context(), cookie)
		if err != nil {
			// Only try to delete if the session exists but there was an error retrieving it
			if err != domain.ErrSessionNotFound {
				if delErr := store.Delete(c.Request.Context(), cookie); delErr != nil {
					c.Error(fmt.Errorf("failed to delete invalid session: %w", delErr))
				}
			}
			clearSessionCookie(c, config)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
			return
		}

		if session == nil {
			clearSessionCookie(c, config)
			c.Next()
			return
		}

		if session.IsExpired() {
			if err := store.Delete(c.Request.Context(), session.ID); err != nil {
				c.Error(fmt.Errorf("failed to delete expired session: %w", err))
			}
			clearSessionCookie(c, config)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		if err := store.Touch(c.Request.Context(), session.ID); err != nil {
			c.Error(fmt.Errorf("failed to touch session: %w", err))
		}

		c.Set("session", session)
		c.Set("session_id", session.ID)
		c.Set("user_id", session.UserID)
		c.Set("role", session.Role)
		c.Set("permissions", session.Permissions)

		c.Next()
	}
}

// CreateSession middleware creates a new session for authenticated users
func CreateSession(store domain.SessionStore, cfg domain.SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if there's already a valid session
		if sessionID, err := c.Cookie(cfg.CookieName); err == nil && sessionID != "" {
			if session, err := store.Get(c.Request.Context(), sessionID); err == nil && session != nil && !session.IsExpired() {
				// Valid session exists, continue
				c.Next()
				return
			}
		}

		claims, exists := c.Get("claims")
		if !exists {
			// No claims means no authentication, just continue
			c.Next()
			return
		}

		userClaims, ok := claims.(*domain.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid claims type",
			})
			return
		}

		// Get existing sessions count
		sessions, err := store.GetUserSessions(c.Request.Context(), userClaims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check existing sessions",
			})
			return
		}

		// Check session limit and delete oldest if needed
		if len(sessions) >= cfg.MaxSessionsPerUser {
			oldestSession := sessions[0]
			for _, s := range sessions[1:] {
				if s.CreatedAt.Before(oldestSession.CreatedAt) {
					oldestSession = s
				}
			}
			if err := store.Delete(c.Request.Context(), oldestSession.ID); err != nil {
				c.Error(fmt.Errorf("failed to delete old session: %w", err))
				// Continue anyway since this is not critical
			}
		}

		// Create new session
		now := time.Now()
		session := &domain.Session{
			ID:          uuid.New().String(),
			UserID:      userClaims.UserID,
			Role:        userClaims.Role,
			Permissions: userClaims.Permissions,
			ExpiresAt:   now.Add(cfg.SessionDuration),
			CreatedAt:   now,
			LastSeenAt:  now,
		}

		if err := store.Create(c.Request.Context(), session); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create session",
			})
			return
		}

		// Set session cookie
		c.SetCookie(
			cfg.CookieName,
			session.ID,
			int(cfg.SessionDuration.Seconds()),
			cfg.CookiePath,
			cfg.CookieDomain,
			cfg.CookieSecure,
			cfg.CookieHTTPOnly,
		)

		// Set session data in context
		c.Set("session", session)
		c.Set("session_id", session.ID)
		c.Set("user_id", session.UserID)
		c.Set("role", session.Role)
		c.Set("permissions", session.Permissions)

		c.Next()
	}
}

// ClearSession middleware clears the session cookie and deletes the session
func ClearSession(store domain.SessionStore, config domain.SessionConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(config.CookieName)
		if err != nil {
			if err == http.ErrNoCookie {
				clearSessionCookie(c, config)
				c.Next()
				return
			}
			clearSessionCookie(c, config)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read session cookie"})
			return
		}

		// Try to get the session first to verify it exists
		session, err := store.Get(c.Request.Context(), cookie)
		if err != nil || session == nil {
			clearSessionCookie(c, config)
			c.Next()
			return
		}

		// Delete the session
		if err := store.Delete(c.Request.Context(), cookie); err != nil {
			clearSessionCookie(c, config)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
			return
		}

		clearSessionCookie(c, config)
		c.Next()
	}
}

// RequireSession middleware ensures a valid session exists
func RequireSession(store domain.SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := c.Get("session")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
			return
		}

		s, ok := session.(*domain.Session)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid session type"})
			return
		}

		if s.IsExpired() {
			if err := store.Delete(c.Request.Context(), s.ID); err != nil {
				c.Error(fmt.Errorf("failed to delete expired session: %w", err))
			}
			clearSessionCookie(c, domain.SessionConfig{
				CookieName: "session_id",
				CookiePath: "/",
			})
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		// Verify session still exists in store
		storedSession, err := store.Get(c.Request.Context(), s.ID)
		if err != nil || storedSession == nil {
			clearSessionCookie(c, domain.SessionConfig{
				CookieName: "session_id",
				CookiePath: "/",
			})
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session not found"})
			return
		}

		// Update last seen time
		if err := store.Touch(c.Request.Context(), s.ID); err != nil {
			c.Error(fmt.Errorf("failed to touch session: %w", err))
		}

		c.Next()
	}
}

func clearSessionCookie(c *gin.Context, config domain.SessionConfig) {
	c.SetCookie(config.CookieName, "", -1, config.CookiePath, config.CookieDomain, config.CookieSecure, config.CookieHTTPOnly)
}
