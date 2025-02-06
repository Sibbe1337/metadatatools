package middleware

import (
	"metadatatool/internal/pkg/metrics"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// MetricsMiddleware creates a middleware for collecting Prometheus metrics
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		method := c.Method()
		path := c.Route().Path // Use the route path instead of the actual path to avoid cardinality issues

		// Handle panic recovery
		defer func() {
			if err := recover(); err != nil {
				metrics.HttpRequestsTotal.WithLabelValues(method, path, "500").Inc()
				panic(err) // Re-panic after metrics are collected
			}
		}()

		// Process request
		err := c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())

		metrics.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(method, path).Observe(duration)

		// Update active users (if authenticated)
		if user := c.Locals("user"); user != nil {
			metrics.ActiveUsers.Inc()
		}

		return err
	}
}
