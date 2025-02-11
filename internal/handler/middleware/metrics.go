package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Metrics singleton instance
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	initOnce            sync.Once
	metricsMutex        sync.Mutex
)

func initMetrics() {
	metricsMutex.Lock()
	defer metricsMutex.Unlock()

	// Unregister existing metrics if they exist
	prometheus.Unregister(httpRequestsTotal)
	prometheus.Unregister(httpRequestDuration)

	// Create new metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// Register new metrics
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

// Metrics middleware collects HTTP metrics
func Metrics() gin.HandlerFunc {
	// Initialize metrics only once
	initOnce.Do(initMetrics)

	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, c.FullPath()).Observe(duration)
	}
}
