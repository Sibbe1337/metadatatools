// Package middleware provides HTTP middleware functions
package middleware

import (
	"fmt"
	"metadatatool/internal/pkg/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware returns a middleware that collects HTTP metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Increment active users
		metrics.ActiveUsers.Inc()
		defer metrics.ActiveUsers.Dec()

		// Process request
		c.Next()

		// Record metrics after request is processed
		duration := time.Since(start).Seconds()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		// Record request total
		metrics.HttpRequestsTotal.WithLabelValues(
			status,
			method,
			path,
		).Inc()

		// Record request duration
		metrics.HttpRequestDuration.WithLabelValues(
			method,
			path,
		).Observe(duration)

		// Record errors if any
		if len(c.Errors) > 0 {
			// Log errors for monitoring
			for _, err := range c.Errors {
				fmt.Printf("Error in request: %v\n", err)
			}
		}
	}
}

// DatabaseMetricsMiddleware wraps database operations with metrics
func DatabaseMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record database metrics if any database operation was performed
		if dbOp, exists := c.Get("db_operation"); exists {
			duration := time.Since(start).Seconds()
			operation := dbOp.(string)

			metrics.DatabaseQueryDuration.WithLabelValues(operation).Observe(duration)
			metrics.DatabaseOperationsTotal.WithLabelValues(operation, "success").Inc()

			// Record rows affected if available
			if rowsAffected, exists := c.Get("db_rows_affected"); exists {
				metrics.DatabaseRowsAffected.WithLabelValues(operation).Observe(float64(rowsAffected.(int64)))
			}

			// Record errors if any
			if err, exists := c.Get("db_error"); exists && err != nil {
				metrics.DatabaseErrors.WithLabelValues(operation, "error").Inc()
			}
		}
	}
}

// AIMetricsMiddleware wraps AI service operations with metrics
func AIMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record AI metrics if any AI operation was performed
		if aiOp, exists := c.Get("ai_operation"); exists {
			duration := time.Since(start).Seconds()
			operation := aiOp.(string)

			metrics.AIRequestDuration.WithLabelValues(operation).Observe(duration)

			// Record provider-specific metrics
			if provider, exists := c.Get("ai_provider"); exists {
				providerStr := provider.(string)
				metrics.AIRequestTotal.WithLabelValues(providerStr, "success").Inc()

				// Record confidence score if available
				if confidence, exists := c.Get("ai_confidence"); exists {
					metrics.AIConfidenceScore.WithLabelValues(providerStr).Observe(confidence.(float64))
				}

				// Record batch size if available
				if batchSize, exists := c.Get("ai_batch_size"); exists {
					metrics.AIBatchSize.WithLabelValues(providerStr).Observe(float64(batchSize.(int)))
				}

				// Record fallback if used
				if fallbackProvider, exists := c.Get("ai_fallback_provider"); exists {
					metrics.AIFallbackTotal.WithLabelValues(providerStr, fallbackProvider.(string)).Inc()
				}

				// Record errors if any
				if err, exists := c.Get("ai_error"); exists && err != nil {
					metrics.AIErrorTotal.WithLabelValues(providerStr, "error").Inc()
				}
			}
		}
	}
}

// CacheMetricsMiddleware wraps cache operations with metrics
func CacheMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()

		// Record cache metrics if any cache operation was performed
		if cacheType, exists := c.Get("cache_type"); exists {
			cacheTypeStr := cacheType.(string)

			// Record hits/misses
			if hit, exists := c.Get("cache_hit"); exists {
				if hit.(bool) {
					metrics.CacheHits.WithLabelValues(cacheTypeStr).Inc()
				} else {
					metrics.CacheMisses.WithLabelValues(cacheTypeStr).Inc()
				}
			}
		}
	}
}
