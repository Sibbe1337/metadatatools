package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	redis *redis.Client
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(redis *redis.Client) *HealthHandler {
	return &HealthHandler{
		redis: redis,
	}
}

// ServiceStatus represents the status of an individual service
type ServiceStatus struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Latency   int64  `json:"latency_ms"`
	Timestamp string `json:"timestamp"`
}

// HealthStatus represents the overall health check response
type HealthStatus struct {
	Status   string                   `json:"status"`
	Services map[string]ServiceStatus `json:"services"`
}

// Check performs health checks on all services
func (h *HealthHandler) Check(c *gin.Context) {
	status := HealthStatus{
		Status:   "healthy",
		Services: make(map[string]ServiceStatus),
	}

	// Check Redis
	redisCtx := c.Request.Context()
	redisStart := time.Now()
	_, err := h.redis.Ping(redisCtx).Result()
	if err != nil {
		status.Services["redis"] = ServiceStatus{
			Status:    "error",
			Message:   "Failed to ping Redis",
			Latency:   time.Since(redisStart).Milliseconds(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		status.Status = "degraded"
	} else {
		status.Services["redis"] = ServiceStatus{
			Status:    "healthy",
			Latency:   time.Since(redisStart).Milliseconds(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
	}

	// Set appropriate status code
	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
		return
	}
	c.JSON(http.StatusServiceUnavailable, status)
}
