// Package handler provides HTTP request handlers
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler handles Prometheus metrics requests
type MetricsHandler struct{}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// PrometheusHandler exposes Prometheus metrics
func (h *MetricsHandler) PrometheusHandler() gin.HandlerFunc {
	handler := promhttp.Handler()

	return func(c *gin.Context) {
		// Disable Gin's default recovery for this endpoint
		// as Prometheus handler has its own recovery
		defer func() {
			if err := recover(); err != nil {
				c.AbortWithStatus(500)
			}
		}()

		handler.ServeHTTP(c.Writer, c.Request)
	}
}

// HealthCheck provides a basic health check endpoint
func (h *MetricsHandler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}
