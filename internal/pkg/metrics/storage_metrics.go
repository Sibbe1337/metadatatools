package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Storage metrics
var (
	// StorageOperationDuration tracks duration of storage operations
	StorageOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "storage_operation_duration_seconds",
			Help:    "Duration of storage operations",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"operation"},
	)

	// StorageOperationErrors tracks storage operation errors
	StorageOperationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_operation_errors_total",
			Help: "Total number of storage operation errors",
		},
		[]string{"operation"},
	)

	// StorageOperationSuccess tracks successful storage operations
	StorageOperationSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_operation_success_total",
			Help: "Total number of successful storage operations",
		},
		[]string{"operation"},
	)

	// StorageQuotaUsage tracks storage quota usage by user
	StorageQuotaUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "storage_quota_usage_bytes",
			Help: "Current storage quota usage in bytes",
		},
		[]string{"user_id"},
	)

	// StorageCleanupFilesDeleted tracks number of files deleted during cleanup
	StorageCleanupFilesDeleted = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "storage_cleanup_files_deleted_total",
			Help: "Total number of temporary files deleted during cleanup",
		},
	)

	// StorageCleanupBytesReclaimed tracks storage space reclaimed during cleanup
	StorageCleanupBytesReclaimed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "storage_cleanup_bytes_reclaimed_total",
			Help: "Total bytes reclaimed from deleted temporary files",
		},
	)

	// StorageTempFileCount tracks the number of temporary files
	StorageTempFileCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "storage_temp_file_count",
			Help: "Number of temporary files in storage",
		},
	)
)
