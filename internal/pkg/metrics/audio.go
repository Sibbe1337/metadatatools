package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// AudioProcessingDuration tracks the duration of audio processing operations
	AudioProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "audio_processing_duration_seconds",
		Help:    "Duration of audio processing operations",
		Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
	}, []string{"operation"})

	// AudioProcessingErrors tracks the number of audio processing errors
	AudioProcessingErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audio_processing_errors_total",
		Help: "Total number of audio processing errors",
	}, []string{"operation", "error_type"})

	// AudioProcessingSuccess tracks the number of successful audio processing operations
	AudioProcessingSuccess = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audio_processing_success_total",
		Help: "Total number of successful audio processing operations",
	}, []string{"operation"})

	// AudioFilesProcessed tracks the number of audio files processed
	AudioFilesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audio_files_processed_total",
		Help: "Total number of audio files processed",
	}, []string{"format"})

	// AudioFileSize tracks the size of processed audio files
	AudioFileSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "audio_file_size_bytes",
		Help:    "Size of processed audio files in bytes",
		Buckets: prometheus.ExponentialBuckets(1024*1024, 2, 10), // Start at 1MB
	}, []string{"format"})
)
