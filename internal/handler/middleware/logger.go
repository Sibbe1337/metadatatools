package middleware

import (
	"fmt"
	"metadatatool/internal/pkg/logger"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger creates a logging middleware for Fiber
func Logger() fiber.Handler {
	log := logger.NewLogger()

	return func(c *fiber.Ctx) error {
		start := time.Now()
		path := c.Path()
		method := c.Method()

		// Log request
		log.WithFields(logger.Fields{
			"method": method,
			"path":   path,
			"ip":     c.IP(),
		}).Info("Incoming request")

		// Handle request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get response status
		status := c.Response().StatusCode()

		// Log fields
		fields := logger.Fields{
			"method":   method,
			"path":     path,
			"status":   status,
			"duration": fmt.Sprintf("%dms", duration.Milliseconds()),
			"ip":       c.IP(),
		}

		// Add error if present
		if err != nil {
			fields["error"] = err.Error()
			log.WithFields(fields).Error("Request failed")
			return err
		}

		// Log success
		log.WithFields(fields).Info("Request completed")
		return nil
	}
}

// RequestID adds a unique request ID to each request
func RequestID() fiber.Handler {
	log := logger.NewLogger()

	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			c.Set("X-Request-ID", requestID)
		}

		log.WithFields(logger.Fields{
			"request_id": requestID,
		}).Info("Request ID assigned")

		return c.Next()
	}
}
