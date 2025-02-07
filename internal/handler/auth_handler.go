package handler

import (
	"metadatatool/internal/pkg/domain"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService  domain.AuthService
	userRepo     domain.UserRepository
	sessionStore domain.SessionStore
}

func NewAuthHandler(
	authService domain.AuthService,
	userRepo domain.UserRepository,
	sessionStore domain.SessionStore,
) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		userRepo:     userRepo,
		sessionStore: sessionStore,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Name     string      `json:"name"`
	Company  string      `json:"company"`
	Role     domain.Role `json:"role"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing required fields",
		})
		return
	}

	// Check if user already exists
	existingUser, err := h.userRepo.GetByEmail(c, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to check existing user",
		})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "email already registered",
		})
		return
	}

	// Hash password
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to hash password",
		})
		return
	}

	// Create user
	user := &domain.User{
		Email:          req.Email,
		Password:       hashedPassword,
		Name:           req.Name,
		Company:        req.Company,
		Role:           req.Role,
		Plan:           "free",
		TrackQuota:     10,
		TracksUsed:     0,
		QuotaResetDate: time.Now().AddDate(0, 1, 0), // Reset in 1 month
	}

	if err := h.userRepo.Create(c, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create user",
		})
		return
	}

	// Generate tokens
	tokens, err := h.authService.GenerateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate tokens",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":   user,
		"tokens": tokens,
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Get user by email
	user, err := h.userRepo.GetByEmail(c, req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	// Verify password
	if err := h.authService.VerifyPassword(user.Password, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid credentials",
		})
		return
	}

	// Update last login
	user.LastLoginAt = time.Now()
	if err := h.userRepo.Update(c, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update last login",
		})
		return
	}

	// Generate tokens
	tokens, err := h.authService.GenerateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate tokens",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":   user,
		"tokens": tokens,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get session from context
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"message": "logged out",
		})
		return
	}

	// Delete session
	currentSession := session.(*domain.Session)
	if err := h.sessionStore.Delete(c, currentSession.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	refreshToken := c.GetHeader("X-Refresh-Token")
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing refresh token",
		})
		return
	}

	tokens, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
	})
}

// GenerateAPIKey generates a new API key for the user
func (h *AuthHandler) GenerateAPIKey(c *gin.Context) {
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}
	userClaims := claims.(*domain.Claims)

	user, err := h.userRepo.GetByID(c, userClaims.UserID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}

	apiKey, err := h.authService.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to generate API key",
		})
		return
	}

	user.APIKey = apiKey
	if err := h.userRepo.Update(c, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save API key",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": apiKey,
	})
}

// GetActiveSessions returns all active sessions for the current user
func (h *AuthHandler) GetActiveSessions(c *gin.Context) {
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}
	userClaims := claims.(*domain.Claims)

	sessions, err := h.sessionStore.GetUserSessions(c, userClaims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
	})
}

// RevokeSession revokes a specific session
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing session ID",
		})
		return
	}

	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}
	userClaims := claims.(*domain.Claims)

	// Get session to verify ownership
	session, err := h.sessionStore.Get(c, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get session",
		})
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "session not found",
		})
		return
	}

	// Verify session belongs to user
	if session.UserID != userClaims.UserID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "cannot revoke other user's session",
		})
		return
	}

	// Delete session
	if err := h.sessionStore.Delete(c, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to revoke session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "session revoked",
	})
}

// RevokeAllSessions revokes all sessions for the current user
func (h *AuthHandler) RevokeAllSessions(c *gin.Context) {
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}
	userClaims := claims.(*domain.Claims)

	if err := h.sessionStore.DeleteUserSessions(c, userClaims.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to revoke sessions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "all sessions revoked",
	})
}
