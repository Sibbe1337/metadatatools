package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// JobsProcessed tracks the total number of processed jobs
	JobsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_processed_total",
			Help: "The total number of processed jobs",
		},
		[]string{"type", "status"},
	)

	// JobsInQueue tracks the current number of jobs in queue
	JobsInQueue = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jobs_in_queue",
			Help: "The current number of jobs in queue",
		},
		[]string{"type", "priority"},
	)

	// JobProcessingDuration tracks job processing duration
	JobProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "job_processing_duration_seconds",
			Help:    "Time spent processing jobs",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
		},
		[]string{"type"},
	)

	// JobRetries tracks the number of job retries
	JobRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "job_retries_total",
			Help: "The total number of job retries",
		},
		[]string{"type"},
	)

	// JobErrors tracks the number of job errors
	JobErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "job_errors_total",
			Help: "The total number of job errors",
		},
		[]string{"type", "error_type"},
	)

	// JobStatusTransitions tracks job status changes
	JobStatusTransitions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "job_status_transitions_total",
			Help: "The total number of job status transitions",
		},
		[]string{"type", "from", "to"},
	)

	// JobQueueLatency tracks how long jobs wait in queue
	JobQueueLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "job_queue_latency_seconds",
			Help:    "Time spent by jobs waiting in queue",
			Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600},
		},
		[]string{"type", "priority"},
	)

	// JobBatchSize tracks the size of job batches
	JobBatchSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "job_batch_size",
			Help:    "Size of job batches",
			Buckets: []float64{1, 5, 10, 20, 50, 100, 200, 500},
		},
		[]string{"type"},
	)

	// JobCleanupOperations tracks cleanup operations
	JobCleanupOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "job_cleanup_operations_total",
			Help: "The total number of job cleanup operations",
		},
		[]string{"operation", "status"},
	)
)
