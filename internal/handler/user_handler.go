package handler

import (
	"fmt"
	"metadatatool/internal/pkg/domain"
	"metadatatool/internal/pkg/errortracking"
	"metadatatool/internal/pkg/metrics"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userRepo     domain.UserRepository
	errorTracker *errortracking.ErrorTracker
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userRepo domain.UserRepository,
	errorTracker *errortracking.ErrorTracker,
) *UserHandler {
	return &UserHandler{
		userRepo:     userRepo,
		errorTracker: errorTracker,
	}
}

// CreateUser handles user creation requests
func (h *UserHandler) CreateUser(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("create_user", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("create_user").Observe(time.Since(start).Seconds())
	}()

	var user domain.User
	if err := c.ShouldBindJSON(&user); err != nil {
		h.handleError(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := validateUser(&user); err != nil {
		h.handleError(c, http.StatusBadRequest, "invalid user data", err)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to hash password", err)
		return
	}
	user.Password = string(hashedPassword)

	// Generate API key
	user.APIKey = uuid.New().String()

	// Set default values
	user.Role = domain.RoleUser
	user.Plan = domain.PlanBasic
	user.TrackQuota = 100
	user.TracksUsed = 0
	user.QuotaResetDate = time.Now().AddDate(0, 1, 0) // Reset in 1 month
	user.LastLoginAt = time.Now()

	if err := h.userRepo.Create(c, &user); err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	// Don't return the hashed password
	user.Password = ""
	c.JSON(http.StatusCreated, user)
}

// GetUser retrieves a user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("get_user", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("get_user").Observe(time.Since(start).Seconds())
	}()

	id := c.Param("id")
	if id == "" {
		h.handleError(c, http.StatusBadRequest, "missing user ID", nil)
		return
	}

	user, err := h.userRepo.GetByID(c, id)
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to get user", err)
		return
	}

	if user == nil {
		h.handleError(c, http.StatusNotFound, "user not found", nil)
		return
	}

	// Don't return the hashed password
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

// UpdateUser modifies an existing user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("update_user", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("update_user").Observe(time.Since(start).Seconds())
	}()

	id := c.Param("id")
	if id == "" {
		h.handleError(c, http.StatusBadRequest, "missing user ID", nil)
		return
	}

	var user domain.User
	if err := c.ShouldBindJSON(&user); err != nil {
		h.handleError(c, http.StatusBadRequest, "invalid request body", err)
		return
	}

	user.ID = id

	if err := validateUser(&user); err != nil {
		h.handleError(c, http.StatusBadRequest, "invalid user data", err)
		return
	}

	// If password is provided, hash it
	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			h.handleError(c, http.StatusInternalServerError, "failed to hash password", err)
			return
		}
		user.Password = string(hashedPassword)
	}

	if err := h.userRepo.Update(c, &user); err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to update user", err)
		return
	}

	// Don't return the hashed password
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

// DeleteUser removes a user
func (h *UserHandler) DeleteUser(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("delete_user", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("delete_user").Observe(time.Since(start).Seconds())
	}()

	id := c.Param("id")
	if id == "" {
		h.handleError(c, http.StatusBadRequest, "missing user ID", nil)
		return
	}

	if err := h.userRepo.Delete(c, id); err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to delete user", err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUsers retrieves a paginated list of users
func (h *UserHandler) ListUsers(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.DatabaseOperationsTotal.WithLabelValues("list_users", "total").Inc()
		metrics.DatabaseQueryDuration.WithLabelValues("list_users").Observe(time.Since(start).Seconds())
	}()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	users, err := h.userRepo.List(c, offset, limit)
	if err != nil {
		h.handleError(c, http.StatusInternalServerError, "failed to list users", err)
		return
	}

	// Don't return hashed passwords
	for _, user := range users {
		user.Password = ""
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Users: users,
		Page:  page,
		Limit: limit,
	})
}

// Helper functions and types

func (h *UserHandler) handleError(c *gin.Context, status int, message string, err error) {
	if err != nil {
		h.errorTracker.CaptureError(err, map[string]string{
			"status":    strconv.Itoa(status),
			"message":   message,
			"path":      c.FullPath(),
			"method":    c.Request.Method,
			"client_ip": c.ClientIP(),
		})
	}

	metrics.DatabaseOperationsTotal.WithLabelValues(c.Request.Method, "error").Inc()

	response := ErrorResponse{
		Error: message,
	}
	if err != nil && status >= 500 {
		response.Details = err.Error()
	}

	c.JSON(status, response)
}

func validateUser(user *domain.User) error {
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}
	if user.Password == "" {
		return fmt.Errorf("password is required")
	}
	if user.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

type ListUsersResponse struct {
	Users []*domain.User `json:"users"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}
