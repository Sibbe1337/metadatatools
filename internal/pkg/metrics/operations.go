package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AudioOps tracks all audio operations
	AudioOps = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audio_operations_total",
			Help: "Total number of audio operations by type and status",
		},
		[]string{"operation", "status"},
	)

	// AudioOpDurations tracks durations of audio operations
	AudioOpDurations = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "audio_operation_duration_seconds",
			Help:    "Duration of audio operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"operation"},
	)

	// AudioOpErrors tracks errors in audio operations
	AudioOpErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "audio_operation_errors_total",
			Help: "Total number of audio operation errors",
		},
		[]string{"operation", "error_type"},
	)
)
