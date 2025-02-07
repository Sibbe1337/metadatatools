// Package metrics provides Prometheus metrics for the application
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HttpRequestsTotal tracks total HTTP requests
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by status code and method",
		},
		[]string{"status", "method", "path"},
	)

	// HttpRequestDuration tracks HTTP request duration
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// ActiveUsers tracks current active users
	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of currently active users",
		},
	)

	// DatabaseQueryDuration tracks database query duration
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// CacheHits tracks cache hit count
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache"},
	)

	// CacheMisses tracks cache miss count
	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache"},
	)

	// AIRequestDuration tracks AI service request duration
	AIRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_request_duration_seconds",
			Help:    "AI service request duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 20, 30},
		},
		[]string{"operation"},
	)

	// TracksProcessed tracks track processing status
	TracksProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tracks_processed_total",
			Help: "Total number of tracks processed by status",
		},
		[]string{"status"},
	)

	// DatabaseConnections tracks database connection pool stats
	DatabaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of database connections by state",
		},
		[]string{"state"},
	)

	// SubscriptionsByPlan tracks user subscriptions by plan
	SubscriptionsByPlan = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "subscriptions_by_plan",
			Help: "Number of subscriptions by plan type",
		},
		[]string{"plan"},
	)

	// DatabaseErrors tracks database errors by type
	DatabaseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_errors_total",
			Help: "Total number of database errors by type",
		},
		[]string{"operation", "error_type"},
	)

	// DatabaseOperationsTotal tracks total database operations
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "status"},
	)

	// DatabaseRowsAffected tracks the number of rows affected by operations
	DatabaseRowsAffected = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_rows_affected",
			Help: "Number of rows affected by database operations",
			Buckets: []float64{
				1, 5, 10, 25, 50, 100, 250, 500, 1000,
			},
		},
		[]string{"operation"},
	)

	// AIRequestTotal tracks the total number of AI service requests
	AIRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_request_total",
			Help: "Total number of AI service requests",
		},
		[]string{"provider", "status"},
	)

	// AIBatchSize tracks the size of batch processing requests
	AIBatchSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ai_batch_size",
			Help: "Size of AI service batch processing requests",
			Buckets: []float64{
				1, 5, 10, 25, 50, 100, 250, 500,
			},
		},
		[]string{"provider"},
	)

	// AIConfidenceScore tracks the confidence scores returned by AI services
	AIConfidenceScore = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "ai_confidence_score",
			Help: "Confidence scores returned by AI services",
			Buckets: []float64{
				0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99,
			},
		},
		[]string{"provider"},
	)

	// AIFallbackTotal tracks the number of times fallback was used
	AIFallbackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_fallback_total",
			Help: "Total number of times fallback was used",
		},
		[]string{"primary_provider", "fallback_provider"},
	)

	// AIErrorTotal tracks the number of errors by type
	AIErrorTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_error_total",
			Help: "Total number of AI service errors by type",
		},
		[]string{"provider", "error_type"},
	)

	// AI Service metrics
	AIBatchProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ai_batch_processing_duration_seconds",
		Help:    "Duration of AI batch processing operations",
		Buckets: prometheus.DefBuckets,
	})

	BatchProcessingTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ai_batch_processing_total",
		Help: "Total number of batch processing operations",
	})

	TracksProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ai_tracks_processed_total",
		Help: "Total number of tracks processed",
	})

	AIEnrichmentErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ai_enrichment_errors_total",
		Help: "Total number of AI enrichment errors",
	})

	AIValidationErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ai_validation_errors_total",
		Help: "Total number of AI validation errors",
	})

	AIRetryAttempts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ai_retry_attempts_total",
		Help: "Total number of AI operation retry attempts",
	})

	AIBatchErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ai_batch_errors_total",
		Help: "Total number of errors in batch processing",
	})

	// Audio processing metrics
	AudioProcessingTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audio_processing_total",
			Help: "Total number of audio processing attempts",
		},
		[]string{"operation", "status"},
	)

	AudioProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "audio_processing_duration_seconds",
			Help:    "Duration of audio processing operations",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
		[]string{"operation"},
	)

	AudioProcessingSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audio_processing_success_total",
			Help: "Total number of successful audio processing operations",
		},
		[]string{"operation"},
	)

	AudioProcessingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audio_processing_errors_total",
			Help: "Total number of audio processing errors",
		},
		[]string{"operation", "error_type"},
	)

	// AI processing metrics
	AIProcessingTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_processing_total",
			Help: "Total number of AI processing attempts",
		},
		[]string{"model", "operation"},
	)

	AIProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_processing_duration_seconds",
			Help:    "Duration of AI processing operations",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"model", "operation"},
	)

	AIProcessingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_processing_errors_total",
			Help: "Total number of AI processing errors",
		},
		[]string{"model", "error_type"},
	)
)

// init registers all metrics with Prometheus
func init() {
	// Pre-create some common label combinations to avoid runtime initialization
	HttpRequestsTotal.WithLabelValues("200", "GET", "/health").Add(0)
	DatabaseQueryDuration.WithLabelValues("track_get").Observe(0)
	CacheHits.WithLabelValues("track").Add(0)
	CacheMisses.WithLabelValues("track").Add(0)
	AIRequestDuration.WithLabelValues("openai").Observe(0)
	TracksProcessed.WithLabelValues("success").Add(0)
	DatabaseConnections.WithLabelValues("active").Set(0)
	SubscriptionsByPlan.WithLabelValues("free").Set(0)
}
