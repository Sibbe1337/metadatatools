package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"metadatatool/internal/domain"
	pkgdomain "metadatatool/internal/pkg/domain"
	"metadatatool/internal/usecase"
)

// AuthHandler handles HTTP requests related to authentication
type AuthHandler struct {
	authUseCase  usecase.AuthUseCaseInterface
	userUseCase  *usecase.UserUseCase
	sessionStore domain.SessionStore
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authUseCase usecase.AuthUseCaseInterface, userUseCase *usecase.UserUseCase, sessionStore domain.SessionStore) *AuthHandler {
	return &AuthHandler{
		authUseCase:  authUseCase,
		userUseCase:  userUseCase,
		sessionStore: sessionStore,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var input usecase.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate input
	if input.Email == "" || input.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	user, err := h.authUseCase.Register(c.Request.Context(), input)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error registering user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": user})
}

// Login handles user login and returns JWT tokens
func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	loginOutput, err := h.authUseCase.Login(c.Request.Context(), usecase.LoginInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Create session for authenticated user
	session := &domain.Session{
		ID:          uuid.New().String(),
		UserID:      loginOutput.User.ID,
		Role:        loginOutput.User.Role,
		Permissions: loginOutput.User.Permissions,
		UserAgent:   c.Request.UserAgent(),
		IP:          c.ClientIP(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
	}

	if err := h.sessionStore.Create(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.SetCookie("session_id", session.ID, int(24*time.Hour.Seconds()), "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  loginOutput.AccessToken,
		"refresh_token": loginOutput.RefreshToken,
		"user": gin.H{
			"id":          loginOutput.User.ID,
			"email":       loginOutput.User.Email,
			"role":        loginOutput.User.Role,
			"permissions": loginOutput.User.Permissions,
		},
	})
}

// ContextKey is a custom type for context keys to avoid SA1029
type ContextKey string

const (
	SessionIDKey ContextKey = "session_id"
)

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	if err := h.authUseCase.Logout(c.Request.Context(), sessionID.(string)); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error logging out"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	user, err := h.userUseCase.GetUser(c.Request.Context(), userID.(string))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

// RefreshToken handles token refresh requests
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	newAccessToken, newRefreshToken, user, err := h.authUseCase.RefreshToken(c.Request.Context(), input.RefreshToken)
	if err != nil {
		var statusCode int
		var message string

		switch {
		case errors.Is(err, domain.ErrInvalidToken):
			statusCode = http.StatusUnauthorized
			message = "invalid refresh token"
		case errors.Is(err, domain.ErrUserNotFound):
			statusCode = http.StatusUnauthorized
			message = "invalid refresh token"
		default:
			statusCode = http.StatusInternalServerError
			message = "internal error"
		}

		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	// Create new session
	internalSession := &domain.Session{
		ID:          uuid.New().String(),
		UserID:      user.ID,
		Role:        user.Role,
		Permissions: user.Permissions,
		UserAgent:   c.Request.UserAgent(),
		IP:          c.ClientIP(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
		CreatedAt:   time.Now(),
		LastSeenAt:  time.Now(),
	}

	if err := h.sessionStore.Create(c.Request.Context(), internalSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.SetCookie("session_id", internalSession.ID, int(24*time.Hour.Seconds()), "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"role":        user.Role,
			"permissions": user.Permissions,
		},
	})
}

// hasPermission checks if a session has a specific permission
func hasPermission(s *domain.Session, permission pkgdomain.Permission) bool {
	for _, p := range s.Permissions {
		if pkgdomain.Permission(p) == permission {
			return true
		}
	}
	return false
}

// GenerateAPIKey handles API key generation requests
func (h *AuthHandler) GenerateAPIKey(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	s, ok := session.(*domain.Session)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid session type"})
		return
	}

	// Check if user has permission to generate API keys
	if !hasPermission(s, pkgdomain.PermissionManageAPIKeys) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		return
	}

	// Get user from database
	user, err := h.userUseCase.GetUser(c.Request.Context(), s.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	// Generate new API key
	apiKey, err := h.authUseCase.GenerateAPIKey(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
}

// GetActiveSessions returns all active sessions for the current user
func (h *AuthHandler) GetActiveSessions(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	s, ok := session.(*domain.Session)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session type"})
		return
	}

	sessions, err := h.sessionStore.GetUserSessions(c.Request.Context(), s.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// RevokeSession revokes a specific session
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	s, ok := session.(*domain.Session)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session type"})
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Get the session to be revoked
	targetSession, err := h.sessionStore.Get(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Only allow users to revoke their own sessions
	if targetSession.UserID != s.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot revoke other users' sessions"})
		return
	}

	if err := h.sessionStore.Delete(c.Request.Context(), sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked successfully"})
}

// RevokeAllSessions revokes all sessions for the current user
func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No active session"})
		return
	}

	s, ok := session.(*domain.Session)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session type"})
		return
	}

	if err := h.sessionStore.DeleteUserSessions(c.Request.Context(), s.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked successfully"})
}

// AuthMiddleware authenticates requests
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try to get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		var token string
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// If no token in header, try to get from session cookie
		if token == "" {
			sessionID, err := c.Cookie("session_id")
			if err == nil {
				token = sessionID
			}
		}

		// If still no token, return unauthorized
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization"})
			c.Abort()
			return
		}

		// Validate token
		user, err := h.authUseCase.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Add user and session info to context
		c.Set("user_id", user.ID)
		c.Set("user_role", user.Role)
		c.Set("session_id", token)

		c.Next()
	}
}

// RoleMiddleware checks if the user has the required role
func (h *AuthHandler) RoleMiddleware(role domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if userRole.(domain.Role) != role && userRole.(domain.Role) != domain.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission middleware checks if the user has the required permission
func RequirePermission(permission domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, exists := c.Get("session")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		s, ok := session.(*domain.Session)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid session type"})
			return
		}

		// Check if the permission exists in the session's permissions
		hasPermission := false
		for _, p := range s.Permissions {
			if p == permission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}
