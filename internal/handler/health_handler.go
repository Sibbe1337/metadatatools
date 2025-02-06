package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db    *gorm.DB
	redis redis.UniversalClient
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(db *gorm.DB, redis redis.UniversalClient) *HealthHandler {
	return &HealthHandler{
		db:    db,
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
	Status    string                   `json:"status"`
	Timestamp string                   `json:"timestamp"`
	Services  map[string]ServiceStatus `json:"services"`
}

// Check handles the health check request
func (h *HealthHandler) Check(c *gin.Context) {
	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Services:  make(map[string]ServiceStatus),
	}

	// Check database
	dbStart := time.Now()
	sqlDB, err := h.db.DB()
	if err != nil {
		status.Services["database"] = ServiceStatus{
			Status:    "error",
			Message:   "Failed to get database instance",
			Latency:   time.Since(dbStart).Milliseconds(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		status.Status = "degraded"
	} else {
		err = sqlDB.PingContext(c)
		if err != nil {
			status.Services["database"] = ServiceStatus{
				Status:    "error",
				Message:   "Failed to ping database",
				Latency:   time.Since(dbStart).Milliseconds(),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
			status.Status = "degraded"
		} else {
			status.Services["database"] = ServiceStatus{
				Status:    "healthy",
				Latency:   time.Since(dbStart).Milliseconds(),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			}
		}
	}

	// Check Redis
	redisStart := time.Now()
	redisCtx, cancel := context.WithTimeout(c, 5*time.Second)
	defer cancel()

	_, err = h.redis.Ping(redisCtx).Result()
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
